
;;;; From the slip repo:
;;;; go run cmd/slip/main.go -i ../slip-ggql/examples/sample.lisp
;;;;
;;;; The use curl to verify with:
;;;; curl -w "\n" "localhost:5555/graphql?query=\{hello\}"
;;;;
;;;; which should return:
;;;; {"data":{"hello":"Hello"}}
;;;;

(defflavor query-flavor ()())
(defmethod (query-flavor :hello) () "Hello")

(defflavor top-flavor ((query (make-instance 'query-flavor))) ()
           :gettable-instance-variables)

(setq top (make-instance 'top-flavor))

(require "ggql" "../slip-ggql")
(setq gs (make-instance 'ggql-server-flavor
                        :port 5555
                        :asset-directory "../slip-ggql/testassets"
                        :schema-instance top
                        :schema-files "../slip-ggql/examples/sample.graphql"))
;(send gs :trace t)
(send gs :start)
