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
			"asset-directory": nil,
		},
		nil,
		slip.List{
			slip.List{
				slip.Symbol(":documentation"),
				slip.String(`Is a GraphQL server based on the GGql golang package. The resolvers for
requests are instances of a flavor that responds to methods that match the fields in each GraphQL type.
The root resolver will also resolve data held in a __bag__ instance.
`),
			},
			slip.Symbol(":inittable-instance-variables"),
			slip.Symbol(":gettable-instance-variables"),
			slip.Symbol(":settable-instance-variables"),
			slip.List{
				slip.Symbol(":init-keywords"),
				slip.Symbol(":schema-instance"),
				slip.Symbol(":schema-files"),
				slip.Symbol(":schema-stream"),
			},
		},
	)
	serverFlavor.DefMethod(":init", "", initCaller(true))
	serverFlavor.DefMethod(":start", "", startCaller(true))
	serverFlavor.DefMethod(":stop", "", stopCaller(true))
	serverFlavor.DefMethod(":activep", "", activepCaller(true))
	serverFlavor.DefMethod(":trace", "", traceCaller(true))
	serverFlavor.DefMethod(":set-schema-instance", "", setSchemaInstanceCaller(true))
	serverFlavor.DefMethod(":schema-instance", "", schemaInstanceCaller(true))
	serverFlavor.DefMethod(":set-schema-files", "", setSchemaFilesCaller(true))
	serverFlavor.DefMethod(":schema", "", schemaCaller(true))
}

// ServerFlavor returns the ggql-server-flavor.
func ServerFlavor() *flavors.Flavor {
	return serverFlavor
}

type initCaller bool

func (caller initCaller) Call(s *slip.Scope, args slip.List, _ int) slip.Object {
	obj := s.Get("self").(*flavors.Instance)
	sw := &serverWrap{}
	obj.Any = sw
	list := args[0].(slip.List)
	for i := 0; i < len(list)-1; i += 2 {
		if sym, ok := list[i].(slip.Symbol); ok {
			switch string(sym) {
			case ":schema-files":
				sw.schema = readFiles(nil, list[i+1])
				continue
			case ":schema-stream":
				var r io.Reader
				if r, ok = list[i+1].(io.Reader); ok {
					sw.schema = readStream(r)
					continue
				}
			case ":schema-instance":
				var top *flavors.Instance
				if top, ok = list[i+1].(*flavors.Instance); !ok {
					panic(fmt.Sprintf("schema-instance must be an instance of a flavor not %s", list[i+1]))
				}
				obj.Set(slip.Symbol("schema-instance"), top)
				continue
			}
		}
		panic(fmt.Sprintf("%s is not a valid keyword and value", list[i]))
	}
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
		if len(sw.schema) == 0 {
			panic("schema not yet loaded. Call :set-schema-files or :set-schema-stream first")
		}
		sw.makeRoot(top)
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
	if 0 < len(args) && args[0] != nil {
		_ = sw.server.ListenAndServe()
	} else {
		go func() { _ = sw.server.ListenAndServe() }()
	}
	return nil
}

func (caller startCaller) Docs() string {
	return `__:start__ &optional block
  _block_ if non-nil will cause the server to not return until stopped.

Starts listening for connection on the _:port_.
`
}

type stopCaller bool

func (caller stopCaller) Call(s *slip.Scope, args slip.List, _ int) slip.Object {
	obj := s.Get("self").(*flavors.Instance)
	sw := obj.Any.(*serverWrap)
	if sw.server != nil {
		_ = sw.server.Shutdown(context.Background())
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

type setSchemaFilesCaller bool

func (caller setSchemaFilesCaller) Call(s *slip.Scope, args slip.List, _ int) slip.Object {
	obj := s.Get("self").(*flavors.Instance)
	sw := obj.Any.(*serverWrap)
	sw.schema = readFiles(nil, args)

	si := obj.Get(slip.Symbol("schema-instance"))
	if si != nil {
		top, ok := si.(*flavors.Instance)
		if !ok {
			panic(fmt.Sprintf("schema-instance must be an instance of a flavor not %s", si))
		}
		sw.makeRoot(top)
	}
	return nil
}

func (caller setSchemaFilesCaller) Docs() string {
	return `__:set-schema-files__ &rest files
  __files__ filename or glob to use as the schema. A list of filenames and globs is also supported.

Concatenates the contents of the files to for the GraphQL schema for the server.
`
}

type setSchemaInstanceCaller bool

func (caller setSchemaInstanceCaller) Call(s *slip.Scope, args slip.List, _ int) slip.Object {
	if len(args) != 1 {
		panic(fmt.Sprintf(
			"Method ggql-server-flavor :set-schema-instance method expects at one argument but received %d.",
			len(args)))
	}
	obj := s.Get("self").(*flavors.Instance)
	sw := obj.Any.(*serverWrap)

	top, ok := args[0].(*flavors.Instance)
	if !ok {
		panic(fmt.Sprintf("schema-instance must be an instance of a flavor not %s", args[0]))
	}
	obj.Set(slip.Symbol("schema-instance"), top)
	if 0 < len(sw.schema) {
		sw.makeRoot(top)
	}
	return nil
}

func (caller setSchemaInstanceCaller) Docs() string {
	return `__:set-schema-instance__ __instance__
  __instance__ of a flavor that with respond to the top level GraphQL resolve requests.

Set the top or root level instance for queries. It must respond to :query and optionally :mutation.
`
}

type schemaInstanceCaller bool

func (caller schemaInstanceCaller) Call(s *slip.Scope, args slip.List, _ int) slip.Object {
	obj := s.Get("self").(*flavors.Instance)
	return obj.Get(slip.Symbol("schema-instance"))
}

func (caller schemaInstanceCaller) Docs() string {
	return `__:set-schema-instance__ __instance__
  __instance__ of a flavor that with respond to the top level GraphQL resolve requests.

Set the top or root level instance for queries. It must respond to :query and optionally :mutation.
`
}

type schemaCaller bool

func (caller schemaCaller) Call(s *slip.Scope, args slip.List, _ int) slip.Object {
	obj := s.Get("self").(*flavors.Instance)
	sw := obj.Any.(*serverWrap)

	return slip.String(sw.schema)
}

func (caller schemaCaller) Docs() string {
	return `__:schema__

Returns the schema as a string.
`
}
