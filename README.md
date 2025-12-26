# slip-ggql

A GraphQL plug-in for SLIP.

This plugin provides a server that use [GGql](https://github.com/uhn/ggql).

-------------

## Future Plans

 - Add and example such as the song example from ggql.
 - Add tests making use of the song example.

 - Optimize schema loading by immediatetly into a []byte.
 - Add ability to provide schema as a string or a stream with :schema-input and :schema-stream options.
 - Only re-root if not :init and schema instance is not nil
