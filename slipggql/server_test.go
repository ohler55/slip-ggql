// Copyright (c) 2023, Peter Ohler, All rights reserved.

package slipggql_test

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/ohler55/ojg/tt"
	"github.com/ohler55/slip"
	"github.com/ohler55/slip/sliptest"
)

func TestServerAttrs(t *testing.T) {
	scope := slip.NewScope()
	scope.Let("server", nil)
	_ = slip.ReadString(`(setq server
                               (make-instance 'ggql-server-flavor
                                              :schema-files "examples/sample.graphql"
                                              :port 5555
                                              :base "gg"))`).Eval(scope, nil)
	(&sliptest.Function{
		Scope:  scope,
		Source: `(send server :port)`,
		Expect: "5555",
	}).Test(t)
	(&sliptest.Function{
		Scope:  scope,
		Source: `(send server :base)`,
		Expect: `"gg"`,
	}).Test(t)
}

func TestServerStart(t *testing.T) {
	// TBD get free port
	scope := slip.NewScope()
	scope.Let("server", nil)
	_ = slip.ReadString(`(setq
                           top (make-instance 'vanilla-flavor)
                           server
                               (make-instance 'ggql-server-flavor
                                              :port 15555
                                              :asset-directory "testassets"
                                              :schema-instance top
                                              :schema-files "examples/sample.graphql"))
`).Eval(scope, nil)

	_ = slip.ReadString(`(send server :start)`).Eval(scope, nil)
	for i := 10; 0 < i; i-- {
		if _, err := http.Get("http://localhost:15555/sample.text"); err == nil {
			break
		}
		time.Sleep(time.Millisecond * 10)
	}
	resp, err := http.Get("http://localhost:15555/sample.text")
	tt.Nil(t, err)
	defer resp.Body.Close()
	tt.Equal(t, 200, resp.StatusCode)
	var body []byte
	body, err = io.ReadAll(resp.Body)
	tt.Nil(t, err)
	_ = slip.ReadString(`(send server :stop)`).Eval(scope, nil)
	tt.Equal(t, "This is a sample file.\n", string(body))
}
