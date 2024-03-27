
;;;; From the slip repo:
;;;; go run cmd/slip/main.go -i ../slip-ggql/examples/song.lisp
;;;;
;;;; The use curl to verify with:
;;;; curl -w "\n" "localhost:5555/graphql?query=\{hello\}"
;;;;
;;;; which should return:
;;;; {"data":{"hello":"Hello"}}
;;;;

(defflavor song-flavor (name duration (likes 0)) ()
           :gettable-instance-variables)

(defmethod (song-flavor :like) ()
  (setq likes (1+ likes)))

(defflavor artist-flavor (name songs origin) ()
           :gettable-instance-variables)

(defflavor query-flavor ((artists (list
                                   (make-instance 'artist-flavor
                                                  :name "Fazerdaze"
                                                  :origin '("Morningside" "Auckland" "New Zealand")
                                                  :songs (list
                                                          (make-instance 'song-flavor
                                                                         :name "Jennifer"
                                                                         :duration 240)
                                                          (make-instance 'song-flavor
                                                                         :name "Lucky Girl"
                                                                         :duration 170)
                                                          (make-instance 'song-flavor
                                                                         :name "Friends"
                                                                         :duration 194)
                                                          (make-instance 'song-flavor
                                                                         :name "Reel"
                                                                         :duration 193)))
                                   (make-instance 'artist-flavor
                                                  :name "Viagra Boys"
                                                  :origin '("Stockholm" "Sweden")
                                                  :songs (list
                                                          (make-instance 'song-flavor
                                                                         :name "Down In The Basement"
                                                                         :duration 216)
                                                          (make-instance 'song-flavor
                                                                         :name "Frogstrap"
                                                                         :duration 195)
                                                          (make-instance 'song-flavor
                                                                         :name "Worms"
                                                                         :duration 208)
                                                          (make-instance 'song-flavor
                                                                         :name "Amphetanarchy"
                                                                         :duration 346))))))
  ()
  :gettable-instance-variables)

(defflavor top-flavor (query mutation) ()
           :gettable-instance-variables)

(defvar query)
(defvar top)

(setq
 query (make-instance 'query-flavor)
 top (make-instance 'top-flavor :query query))

(require "ggql" "../slip-ggql")
(setq gs (make-instance 'ggql-server-flavor
                        :port 5555
                        :asset-directory "../slip-ggql/testassets"
                        :schema-instance top
                        :schema-files "../slip-ggql/examples/song.graphql"))
;(send gs :trace t)
(send gs :start)
