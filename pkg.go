// Copyright (c) 2023, Peter Ohler, All rights reserved.

package main

import (
	"github.com/ohler55/slip"
)

var (
	// Pkg is the ggql package.
	Pkg = slip.Package{
		Name:      "ggql",
		Nicknames: []string{},
		Doc:       "GraphQL package for slip.",
		PreSet:    slip.DefaultPreSet,
		Vars:      map[string]*slip.VarVal{},
	}
)

func init() {
	Pkg.Initialize()
	slip.AddPackage(&Pkg)
	slip.UserPkg.Use(&Pkg)
	Pkg.Set("*ggql*", &Pkg)
}
