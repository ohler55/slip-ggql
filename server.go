// Copyright (c) 2023, Peter Ohler, All rights reserved.

package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/ohler55/ojg"
	"github.com/ohler55/ojg/alt"
	"github.com/ohler55/ojg/pretty"
	"github.com/ohler55/slip"
	"github.com/ohler55/slip/pkg/flavors"
)

type serverWrap struct {
	server   *http.Server
	mux      *http.ServeMux
	out      io.Writer
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
	// TBD trace (state &optional stream) - print request and response
	//   state can be nil for off, t for on and :detailed for more

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
	sw.mux = http.NewServeMux()
	// TBD base/graphql
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

	/*
		TBD
			http.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
				handleGraphQL(w, r, root)
			})
	*/

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
