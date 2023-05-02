# slip-ggql

A Graphql plug-in for SLIP.

-------------

- make song example
 - use for tests as well

- schema
 - load immediatetly into a []byte
 - add ability to provide schema as a string, maybe as a stream?
  - :schema-input or :schema-stream
 - only re-root if not :init and schema instance is not nil
