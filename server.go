// Copyright (c) 2023, Peter Ohler, All rights reserved.

package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ohler55/ojg"
	"github.com/ohler55/ojg/alt"
	"github.com/ohler55/ojg/pretty"
	"github.com/ohler55/slip"
	"github.com/ohler55/slip/pkg/flavors"
	"github.com/uhn/ggql/pkg/ggql"
)

type serverWrap struct {
	server   *http.Server
	mux      *http.ServeMux
	out      io.Writer
	root     *ggql.Root
	mu       sync.Mutex
	trace    bool
	detailed bool
}

var serverFlavor *flavors.Flavor

func init() {
	serverFlavor = flavors.DefFlavor("ggql-server-flavor",
		map[string]slip.Object{
			"port":            nil,
			"base":            nil,
			"root":            nil,
			"asset-directory": nil,
			"schema-instance": nil,
			"schema-files":    nil,
			// TBD schema instance
			// TBD graphql file or directory (list of)
			// TBD if set while running then make a new root
			//     maybe use after set for both
		},
		nil,
		slip.List{
			slip.List{
				slip.Symbol(":documentation"),
				slip.String(`TBD`),
			},
			slip.Symbol(":inittable-instance-variables"),
			slip.Symbol(":gettable-instance-variables"),
			slip.Symbol(":settable-instance-variables"),
		},
	)
	serverFlavor.DefMethod(":init", "", initCaller(true))
	serverFlavor.DefMethod(":start", "", startCaller(true))
	serverFlavor.DefMethod(":stop", "", stopCaller(true))
	serverFlavor.DefMethod(":activep", "", activepCaller(true))
	serverFlavor.DefMethod(":trace", "", traceCaller(true))
}

// ServerFlavor returns the ggql-server-flavor.
func ServerFlavor() *flavors.Flavor {
	return serverFlavor
}

type initCaller bool

func (caller initCaller) Call(s *slip.Scope, args slip.List, _ int) slip.Object {
	obj := s.Get("self").(*flavors.Instance)
	obj.Any = &serverWrap{}

	return nil
}

func (caller initCaller) Docs() string {
	return `__:init__`
}

type startCaller bool

func (caller startCaller) Call(s *slip.Scope, args slip.List, _ int) slip.Object {
	obj := s.Get("self").(*flavors.Instance)
	var port int
	if po, ok := obj.Get(slip.Symbol("port")).(slip.Fixnum); ok {
		port = int(po)
	} else {
		panic(fmt.Sprintf("port must be a fixnum not %s", obj.Get(slip.Symbol("port"))))
	}
	sw := obj.Any.(*serverWrap)
	if sw.server != nil {
		return nil
	}
	if sw.root == nil {
		var (
			top *flavors.Instance
			ok  bool
		)
		if top, ok = obj.Get(slip.Symbol("schema-instance")).(*flavors.Instance); !ok {
			panic(fmt.Sprintf("schema-instance must be an instance of a flavor not %s", obj.Get(slip.Symbol("schema-instance"))))
		}
		sw.makeRoot(top, obj.Get(slip.Symbol("schema-files")))
	}
	sw.mux = http.NewServeMux()
	path := "/graphql"
	base := obj.Get(slip.Symbol("base"))
	if base != nil {
		if ps, ok := base.(slip.String); ok {
			path = string(ps)
		} else {
			panic(fmt.Sprintf("base must be a string or nil not %s", obj.Get(slip.Symbol("base"))))
		}
	}
	sw.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		sw.handleGraphQL(w, r)
	})
	if assetDir := obj.Get(slip.Symbol("asset-directory")); assetDir != nil {
		if dir, ok := assetDir.(slip.String); ok {
			sw.mux.Handle("/", http.FileServer(http.Dir(string(dir))))
		} else {
			panic(fmt.Sprintf("asset-directory must be a string not %s", assetDir))
		}
	}
	sw.server = &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        sw,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go func() { _ = sw.server.ListenAndServe() }()

	return nil
}

func (caller startCaller) Docs() string {
	return `__:start__

Starts listening for connection on the _:port_.
`
}

type stopCaller bool

func (caller stopCaller) Call(s *slip.Scope, args slip.List, _ int) slip.Object {
	obj := s.Get("self").(*flavors.Instance)
	sw := obj.Any.(*serverWrap)
	if sw.server != nil {
		sw.server.Shutdown(context.Background())
		sw.server = nil
	}
	return nil
}

func (caller stopCaller) Docs() string {
	return `__:stop__

Stops listening for connection.
`
}

type activepCaller bool

func (caller activepCaller) Call(s *slip.Scope, args slip.List, _ int) slip.Object {
	obj := s.Get("self").(*flavors.Instance)
	sw := obj.Any.(*serverWrap)
	if sw.server != nil {
		return slip.True
	}
	return nil
}

func (caller activepCaller) Docs() string {
	return `__:activep__

Returns _t_ if the server is listening for connections.
`
}

type traceCaller bool

func (caller traceCaller) Call(s *slip.Scope, args slip.List, _ int) slip.Object {
	if len(args) == 0 {
		panic(fmt.Sprintf("Method ggql-server-flavor :trace method expects at least one arguments but received %d.",
			len(args)))
	}
	obj := s.Get("self").(*flavors.Instance)
	sw := obj.Any.(*serverWrap)
	switch args[0] {
	case nil:
		sw.trace = false
		sw.detailed = false
	case slip.True:
		sw.trace = true
		sw.detailed = false
	default:
		sw.trace = true
		sw.detailed = true
	}
	if 1 < len(args) {
		switch ta := args[1].(type) {
		case nil:
			sw.out = nil
		case io.Writer:
			sw.out = ta
		default:
			panic(fmt.Sprintf("Method ggql-server-flavor :trace method stream argument must be an ouput-stream not %s.",
				ta))
		}
	}
	return nil
}

func (caller traceCaller) Docs() string {
	return `__:trace__ __mode__ &optional __stream__
  __mode__ is the trace level _nil_ to turn off tracing, _t_ to turn on basic tracing, and _:detailed_ for much more detail.
  __stream__ if _nil_ then trace output is to _*standard-output*_ else to the provided _output-stream_.

Sets the trace mode and optionally the output stream.
`
}

var traceOptions = &ojg.Options{Sort: true, OmitEmpty: true, OmitNil: true}

type responseTracer struct {
	writer   http.ResponseWriter
	sent     []byte
	status   int
	detailed bool
}

func (rt *responseTracer) Header() http.Header {
	return rt.writer.Header()
}

func (rt *responseTracer) Write(b []byte) (int, error) {
	if rt.detailed {
		rt.sent = append(rt.sent, b...)
	}
	return rt.writer.Write(b)
}

func (rt *responseTracer) WriteHeader(status int) {
	rt.status = status
	rt.writer.WriteHeader(status)
}

// ServeHTTP handles an HTTP request.
func (sw *serverWrap) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// TBD handle panics

	if !sw.trace {
		sw.mux.ServeHTTP(w, r)
		return
	}
	tw := sw.out
	if tw == nil {
		tw = slip.StandardOutput.(io.Writer)
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
	var sdl []byte
	switch tf := files.(type) {
	case slip.String:
		// TBD add load path
		if content, err := ioutil.ReadFile(string(tf)); err == nil {
			sdl = append(sdl, content...)
		} else {
			fmt.Printf("*** %s\n", err)
		}
	case slip.List:
		// TBD
	default:
		// TBD panic
	}
	if err := sw.root.Parse(sdl); err != nil {
		panic(err)
	}
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
	switch to := obj.(type) {
	case *flavors.Instance:
		// TBD build args
		// TBD convert result?
		result = to.Receive(":"+field.Name, slip.List{}, 0)
	case map[string]any:
		result = to[field.Name]
	default:
		fmt.Printf("*** resolved with %T %v\n", obj, obj)
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
