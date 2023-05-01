// Copyright (c) 2023, Peter Ohler, All rights reserved.

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/ohler55/ojg"
	"github.com/ohler55/ojg/alt"
	"github.com/ohler55/ojg/pretty"
	"github.com/ohler55/slip"
	"github.com/ohler55/slip/pkg/bag"
	"github.com/ohler55/slip/pkg/flavors"
	"github.com/uhn/ggql/pkg/ggql"
)

var traceOptions = &ojg.Options{Sort: true, OmitEmpty: true, OmitNil: true}

type serverWrap struct {
	server   *http.Server
	mux      *http.ServeMux
	out      io.Writer
	root     *ggql.Root
	mu       sync.Mutex
	trace    bool
	detailed bool
}

// ServeHTTP handles an HTTP request.
func (sw *serverWrap) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tw := sw.out
	if tw == nil {
		tw = slip.StandardOutput.(io.Writer)
	}
	defer func() {
		if rec := recover(); rec != nil {
			if !sw.trace {
				fmt.Fprintf(tw, "%v\n", rec)
			}
			w.WriteHeader(400)
			fmt.Fprintf(w, "%v\n", rec)
		}
	}()
	if !sw.trace {
		sw.mux.ServeHTTP(w, r)
		return
	}
	sw.mu.Lock()
	defer sw.mu.Unlock()
	fmt.Fprintf(tw, "Received %s %s from %s\n", r.Method, r.URL.Path, r.RemoteAddr)
	if sw.detailed {
		fmt.Fprintln(tw, pretty.SEN(alt.Decompose(r, traceOptions)))
	}
	sw.mu.Unlock()
	rt := responseTracer{
		writer:   w,
		status:   200,
		detailed: sw.detailed,
	}
	sw.mux.ServeHTTP(&rt, r)
	sw.mu.Lock()
	fmt.Fprintf(tw, "Replied to %s with a %d status\n", r.RemoteAddr, rt.status)
	if sw.detailed {
		fmt.Fprintf(tw, "headers: %s\n", pretty.SEN(w.Header()))
		fmt.Fprintf(tw, "%s\n", rt.sent)
	}
}

func (sw *serverWrap) makeRoot(top *flavors.Instance, files slip.Object) {
	ggql.Sort = true
	sw.root = ggql.NewRoot(top)
	sw.root.AnyResolver = sw
	sdl := readFiles(nil, files)
	if err := sw.root.Parse(sdl); err != nil {
		panic(err)
	}
}

func readFiles(sdl []byte, files slip.Object) []byte {
	switch tf := files.(type) {
	case slip.String:
		if content, err := ioutil.ReadFile(string(tf)); err == nil {
			sdl = append(sdl, content...)
		} else {
			// Maybe it's a glob pattern.
			matches, _ := filepath.Glob(string(tf))
			if len(matches) == 0 {
				panic(fmt.Sprintf("%s did not match any files", tf))
			}
			for _, path := range matches {
				var content []byte
				if content, err = ioutil.ReadFile(path); err == nil {
					sdl = append(sdl, content...)
				}
			}
		}
	case slip.List:
		for _, f := range tf {
			sdl = readFiles(sdl, f)
		}
	default:
		panic(fmt.Sprintf("GraphQL files must be a string or list not %s", files))
	}
	return sdl
}

func (sw *serverWrap) handleGraphQL(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Expose-Headers", "*")
	w.Header().Set("Access-Control-Max-Age", "172800")

	var result map[string]interface{}

	switch r.Method {
	case "GET":
		result = sw.root.ResolveString(r.URL.Query().Get("query"), "", nil)
	case "POST":
		defer func() { _ = r.Body.Close() }()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		result = sw.root.ResolveBytes(body, "", nil)
	}
	indent := -1
	if i, err := strconv.Atoi(r.URL.Query().Get("indent")); err == nil {
		indent = i
	}
	_ = ggql.WriteJSONValue(w, result, indent)
}

// Resolve the field into a method call.
func (sw *serverWrap) Resolve(obj any, field *ggql.Field, args map[string]any) (result any, err error) {
top:
	switch to := obj.(type) {
	case *flavors.Instance:
		method := fmt.Sprintf(":%s", strings.ReplaceAll(field.Name, "_", "-"))
		switch {
		case 0 < len(args):
			s := slip.NewScope()
			for k, v := range args {
				s.Vars[k] = coerceToLisp(v)
			}
			result = to.BoundReceive(method, s, 0)
		case to.Flavor == bag.Flavor():
			obj = to.Any
			goto top
		default:
			result = to.Receive(method, slip.List{}, 0)
		}
	case map[string]any:
		result = to[field.Name]
	default:
		return nil, fmt.Errorf("can not resolve %s on a %T\n", field.Name, to)
	}
	switch tr := result.(type) {
	case *flavors.Instance, nil:
		// ok
	case slip.Object:
		result = slip.Simplify(tr)
	}
	return
}

// Len returns the length of the list.
func (sw *serverWrap) Len(list any) int {
	switch tlist := list.(type) {
	case []any:
		return len(tlist)
	case []*flavors.Instance:
		return len(tlist)
	}
	return 0
}

// Nth returns the nth element in a list.
func (sw *serverWrap) Nth(list any, i int) (result any, err error) {
	if i < 0 {
		return 0, fmt.Errorf("index must be >= 0, not %d", i)
	}
	switch tlist := list.(type) {
	case []any:
		if len(tlist) <= i {
			return 0, fmt.Errorf("index must be less than the list length, %d > len %d", i, len(tlist))
		}
		return tlist[i], nil
	case []*flavors.Instance:
		if len(tlist) <= i {
			return 0, fmt.Errorf("index must be less than the list length, %d > len %d", i, len(tlist))
		}
		return tlist[i], nil
	}
	return 0, fmt.Errorf("expected a []any or []*flavors.Instance, not a %T", list)
}

func coerceToLisp(v any) (obj slip.Object) {
	switch tv := v.(type) {
	case nil:
	case slip.Object:
		obj = tv
	case map[string]any:
		inst := bag.Flavor().MakeInstance()
		inst.Any = tv
		obj = inst
	case []any:
		list := make(slip.List, len(tv))
		for i, lv := range list {
			list[i] = coerceToLisp(lv)
		}
		obj = list
	default:
		obj = slip.SimpleObject(tv)
	}
	return
}
