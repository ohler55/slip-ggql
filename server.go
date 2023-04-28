// Copyright (c) 2023, Peter Ohler, All rights reserved.

package main

import (
	"fmt"
	"strings"

	"github.com/ohler55/slip"
	"github.com/ohler55/slip/pkg/flavors"
)

var serverFlavor *flavors.Flavor

func init() {
	serverFlavor = flavors.DefFlavor("ggql-server-flavor", map[string]slip.Object{}, nil,
		slip.List{
			slip.List{
				slip.Symbol(":documentation"),
				slip.String(`TBD`),
			},
			slip.List{
				slip.Symbol(":init-keywords"),
				slip.Symbol(":port"),
				slip.Symbol(":base"),
			},
		},
	)
	// serverFlavor.DefMethod(":init", "", initCaller(true))
}

// ServerFlavor returns the ggql-server-flavor.
func ServerFlavor() *flavors.Flavor {
	return serverFlavor
}

type initCaller bool

func (caller initCaller) Call(s *slip.Scope, args slip.List, _ int) slip.Object {
	obj := s.Get("self").(*flavors.Instance)
	if 0 < len(args) {
		args = args[0].(slip.List)
	}
	obj.Let("base", "/")
	for i := 0; i < len(args); i++ {
		key, _ := args[i].(slip.Symbol)
		i++
		if len(args) <= i {
			panic(fmt.Sprintf("ggql-server-flavor :init method expects zero or key/value pairs but received %d.",
				len(args)))
		}
		value := args[i]
		switch {
		case strings.EqualFold(":port", string(key)):
			if _, ok := value.(slip.Fixnum); !ok {
				panic(fmt.Sprintf("ggql-server-flavor :init method keyword :port expected a fixnum not %s.",
					value))
			}
			obj.Let("port", value)
		case strings.EqualFold(":base", string(key)):
			if _, ok := value.(slip.Fixnum); !ok {
				panic(fmt.Sprintf("ggql-server-flavor :init method keyword :base expected a string not %s.",
					value))
			}
			obj.Let("base", value)
		}
	}
	return nil
}

func (caller initCaller) Docs() string {
	return `__:init__ &key _port_ _base_
   _:port_ sets the port to listen for connections on.
   _:base_ sets the base for GraphQL request. The default is an empty path.

Sets the initial value when _make-instance_ is called.
`
}
