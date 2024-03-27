// Copyright (c) 2023, Peter Ohler, All rights reserved.

package slipggql

import "net/http"

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
