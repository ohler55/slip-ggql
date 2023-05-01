# slip-ggql

Graphql for slip

-------------

- handle input args of map and list
 - convert to bag


 - ggql package (slip-ggql repo)
  - short cut instead of gi http package
  - no permissions or minimal if easy to do
   - function to take headers and return an auth which is passed around by fields
  - ggql-client
   - init
    - port
    - base
   - responses are a bag
   - get
   - post

  - ggql-server flavor
   - :port
   - :base
   - :root (an instance)
   - :asset-directory
   - start
   - shutdown or stop?
  - define root resolver
   - if basic types then convert to go types
   - others stay as instances or bag
  - make-ggql-root
   - function or instance?
   - maybe just create an instance of what ever flavor
   - ggql-root flavor
    - gets full requests so can check auth and mess with headers
    - subclass for query, mutation, and subscription
    - supports http handler api
