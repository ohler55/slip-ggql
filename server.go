// Copyright (c) 2023, Peter Ohler, All rights reserved.

package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ohler55/slip"
	"github.com/ohler55/slip/pkg/flavors"
)

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
	serverFlavor.DefMethod(":set-schema-instance", ":after", rerootCaller(true))
	serverFlavor.DefMethod(":set-schema-files", ":after", rerootCaller(true))
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
			panic(fmt.Sprintf("schema-instance must be an instance of a flavor not %s",
				obj.Get(slip.Symbol("schema-instance"))))
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

type rerootCaller bool

func (caller rerootCaller) Call(s *slip.Scope, args slip.List, _ int) slip.Object {
	obj := s.Get("self").(*flavors.Instance)
	sw := obj.Any.(*serverWrap)
	top, ok := obj.Get(slip.Symbol("schema-instance")).(*flavors.Instance)
	if !ok {
		panic(fmt.Sprintf("schema-instance must be an instance of a flavor not %s",
			obj.Get(slip.Symbol("schema-instance"))))
	}
	sw.makeRoot(top, obj.Get(slip.Symbol("schema-files")))

	return nil
}

func (caller rerootCaller) Docs() string {
	return `__:after :set-schema-instance__ and __:set-schema-files__`
}
