// Copyright (c) 2023, Peter Ohler, All rights reserved.

package main

import (
	"context"
	"fmt"
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
	serverFlavor.DefMethod(":start", "", startCaller(true))
	serverFlavor.DefMethod(":stop", "", stopCaller(true))
}

// ServerFlavor returns the ggql-server-flavor.
func ServerFlavor() *flavors.Flavor {
	return serverFlavor
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
	mux := http.NewServeMux()
	// TBD add for /graphql
	// TBD include base
	if assetDir := obj.Get(slip.Symbol("asset-directory")); assetDir != nil {
		if dir, ok := assetDir.(slip.String); ok {
			mux.Handle("/", http.FileServer(http.Dir(string(dir))))
		} else {
			panic(fmt.Sprintf("asset-directory must be a string not %s", assetDir))
		}
	}
	server := http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	obj.Any = &server
	go func() { _ = server.ListenAndServe() }()

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

TBD.
`
}

type stopCaller bool

func (caller stopCaller) Call(s *slip.Scope, args slip.List, _ int) slip.Object {
	obj := s.Get("self").(*flavors.Instance)
	if obj.Any != nil {
		server := obj.Any.(*http.Server)
		server.Shutdown(context.Background())
	}
	return nil
}

func (caller stopCaller) Docs() string {
	return `__:stop__

TBD.
`
}
