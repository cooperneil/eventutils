# EventUtils

This project provides 3 elements for manipulating cloud events.

1. Schema generation from Go structs
2. Publishing of schemas to this [schema registry](https://github.com/mikehelmick/schemaregistry)
3. The ability go generate functions and clients for those CloudEvents types.

An example deployment of the schema registry is provided at
[https://schemas.in-the-cloud.dev](https://schemas.in-the-cloud.dev)

# Walkthrough

## cmd/typegen

This binary generates [Knative Eventing](https://github.com/knative/eventing)
`EventType` objects and JSON-schema representations based on Go structs.

## cmd/publish

Command for publishing to a schema registry (see above)

## cmd/generate

Commands for generating receive functions, transform functions, and events
clients from data in the schema registries.
