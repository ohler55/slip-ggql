// Copyright (c) 2023, Peter Ohler, All rights reserved.

package main

import (
	"github.com/ohler55/slip"
)

// Pkg is the ggql package.
var Pkg = slip.Package{
	Name:      "ggql",
	Nicknames: []string{},
	Doc:       "GraphQL package for slip.",
	PreSet:    slip.DefaultPreSet,
}

func init() {
	Pkg.Initialize(map[string]*slip.VarVal{})
	slip.AddPackage(&Pkg)
	slip.UserPkg.Use(&Pkg)
	Pkg.Set("*ggql*", &Pkg)
}
