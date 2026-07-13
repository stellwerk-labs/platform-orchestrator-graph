# Stellwerk Platform Orchestrator Graph

The core graph algorithms behind the Stellwerk Platform Orchestrator.

☑️ - resource types, classes, and IDs
☑️ - resource params
☑️ - workloads
☑️ - shared resources
☑️ - matching of resources to module definitions using "rules"
☑️ - dependency resources in module definitions
☑️ - co-provisioned resources in module definitions
  ☑️ - optional "is dependent on current" flag to link co-provisioned resource to current resource
  ☑️ - optional "copy dependents on current" flag to link co-provisioned resource to ancestors of current resource

This is a shared library with zero or at least few production dependencies. This provides the core types and algorithms for:

1. The manifest (the thing the customer gives us describing their workloads and resources)
2. The rules that map requested resources to modules
3. The algorithm for building a graph from (1) and (2) in a deterministic way
4. The algorithm for building a new graph and determine drift when compared to an old graph
5. An algorithm for building a mermaid visualisation of the graph

It does _not_ include:

1. The schema or shape of a module
2. The mechanism for converting this to Terraform (since that depends on a complex HCL parser and library)
3. An implementation for placeholders, resource references, or resource selectors - yet.

This means that this core graph does not change even if we change the way modules are implemented or need to fix or change serialization.

With low dependencies, this package should change very rarely.

## API

First, convert the user manifest into a `Manifest`. The source material may have more fields, like the variable embeddings and everything, but here we just need the core elements that impact the _graph_.

```go
manifest := Manifest{
    Workloads: map[string]ManifestWorkload{
        "my-workload": {
            Resources: map[string]ManifestResource{
                "some-name": {
                   Type: "postgres",
                   # ...
                }
                # ...
            }    
        }
        # ...
    }
    # ...
}
```

Then, convert the set of module rules for the current context or environment into an array of `ModuleDefinition` and then into `NewModuleDefinitionIndex`. Note that this is generic over the type of module configuration structure, here called `ExampleConfiguration`.

```go
index := NewModuleDefinitionIndex([]ModuleDefinition[*ExampleConfiguration]{
    {ResourceType: "postgres", Rules: []Rule{{}}, Configuration: &ExampleConfiguration{ /* ... */ }},
    # ...
})
```

And you can convert the manifest and module index into the final graph with errors and drift calculated. This graph is a `Graph` instance, and you can inspect the `Nodes`, `Edges`, `Workloads`, `SharedResources`, `DepthFirstIterator()` and various variants of these methods to look over the nodes, inspect errors and drift, and look at the module configuration used for each node.

```go
graph, err := SeedAndExpandAll(context.TODO(), manifest, *index, nil)
# ...
```

Once you have a graph, you can store this and reconstruct it as needed. The last parameter of the `SeedAndExpandAll` function is the previous graph. This is used to determine "drift" where a module used to provision a resource is changing. This may be critical to know about when the provisioned infrastructure is _stateful_ like databases, object store buckets, and other sensitive things that should not be dropped or lost.

Note that this is just the first line of defence against losing stateful data. The modules themselves can set their own locking or "prevent-destroy" semantics which you should respect at implementation time.

## What about parameter and selector evaluation?!

For now, not implemented directly in this package but the tools to support it are here.

- `Graph.Nodes` contains all the nodes in the graph including resources and workloads (which are also resources)
- `Graph.Edges` contains the map of source node to target node
- `Graph.SharedResources` contains the map of alias to target node for shared resources

With these, we can support `${resources.fizz..}` and `${shared.buzz...}` resource references.

- A `BuildAdjacencyMatrix()` function that can be used to build an adjacency and distance matrix that supports functions like `FindDescendentsOf` and `FindAncestorsOf` which can be used to navigate up and down.
- Workloads represented as resources with `workload` type
- A `AddAnonymousdependency()` function that can add an arbitrary anonymous edge to enforce dependency across the graph.

With these, we can support CSS-style parent/child navigating selectors.

## Development

- `go tool gotestsum` or `go test -v ./...` to run unit tests.
- `go vet ./...` and `golangci-lint run` to lint the code.
