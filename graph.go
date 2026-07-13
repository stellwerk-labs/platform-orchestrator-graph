package orchestrator_graph

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"iter"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/stellwerk-labs/platform-orchestrator-graph/internal"
)

// DefaultClass is the default class to assign resources that haven't requested a class specifically
var DefaultClass = "default"

// InheritClassOrIdSymbol is the symbol that can be used in place of the class or id to inherit the value from the
// resource that declares the dependency or co-provisions the resource.
var InheritClassOrIdSymbol = "@"

// DefaultWorkloadResourceType is the type of the workload resource node that we inject for each workload.
var DefaultWorkloadResourceType = "workload"

// DefaultWorkloadResourceIdFunc is the function that allocates ids to the resources anchored by the workloads. The
// function by default uses the workload id and alias, but you can choose to override this.
var DefaultWorkloadResourceIdFunc = func(ctx context.Context, workload string, alias string, manifest ManifestResource) string {
	return fmt.Sprintf("workloads.%s.%s", workload, alias)
}

// DefaultSharedResourceIdFunc is the function that allocates ids to the shared resources anchored by the manifest. The
// function uses the resource alias by default.
var DefaultSharedResourceIdFunc = func(ctx context.Context, alias string, manifest ManifestResource) string {
	return fmt.Sprintf("shared.%s", alias)
}

// A Graph is the graph state resulted from applying a set of module definitions and rules to a manifest. This is done
// through the SeedAndExpandAll api OR through a combination of a single call to Seed and 'N' calls to Expand until
// IsExpanded returns true. The graph holds a set of nodes and edges along with the workloads, shared resources.
// The generic parameter D is the implementation of the ModuleConfiguration which includes things like the module,
// version, and dependency declarations. None of which particularly matter for the graph resolution itself. On each node,
// we also store an error found during expansion or a drift when we have compared the result to a previous graph from a
// previous deployment.
type Graph[D ModuleConfiguration] struct {
	// Nodes is the map of node coordinate to node details.
	Nodes map[ResourceCoordinate]ResourceNode[D] `json:"nodes"`
	// Edges is the map of node, to node via a labeled edge. The label is the 'alias' in the dependent resources.
	Edges map[ResourceCoordinate]map[string]ResourceCoordinate `json:"edges"`

	// Workloads are the set of named workloads that have been added from the manifest.
	Workloads map[string]ResourceCoordinate `json:"workloads"`
	// SharedResources are the map of aliased shared resources in the root of the manifest.
	SharedResources map[string]ResourceCoordinate `json:"shared_resources"`
	// LeafResources are leaf resources in the graph that may be workload resources, shared resources, or dependent
	// resources. The common concept is that no nodes depend on them.
	LeafResources map[ResourceCoordinate]bool `json:"leaf_resources"`

	// internal field tracking whether errors were encountered in the graph
	hasErrors bool
	// internal field tracking the expansion state between Seed and Expand calls.
	unexpanded []ResourceCoordinate
	// copyDependentsSet is the set of things waiting to copy dependencies around at the end
	// this is only practical after full graph expansion.
	copyDependentsSet map[ResourceCoordinate][]ResourceCoordinate
	// edgesLookup internal map to perform edge lookups by nodes it connects
	edgesLookup map[ResourceCoordinate]map[ResourceCoordinate]string
}

// ResourceNode is the state of a node in the graph.
type ResourceNode[D ModuleConfiguration] struct {
	// The ModuleConfiguration of this node. This is the _next_ configuration to use, not necessarily what was
	// previously used. This has already had any NextDriftResolution strategy applied.
	ModuleConfiguration D `json:"config"`
	// The Params (parameters) captured from the Manifest or the dependent node.
	Params map[string]interface{} `json:"params"`
	// ParamsDefinedBy is coordinates of the node that defined the Params, which has a direct dependency on this resource and should be used for placeholder context in the params.
	// Empty string, if there's no direct dependency.
	ParamsDefinedBy *ResourceCoordinate `json:"params_defined_by"`
	// CurrentDrift DEPRECATED use DetectedDrift, keeping it for backward compatibility
	CurrentDrift int `json:"current_drift,omitzero"`
	// DetectedDrift is the drift found in the current Expand cycle, caused by the module rules or module configuration
	// changing since the previous graph.
	DetectedDrift DriftType `json:"detected_drift,omitzero"`
	// EffectiveDrift is the final drift after applying the NextDriftResolution strategy to the DetectedDrift.
	EffectiveDrift DriftType `json:"effective_drift,omitzero"`

	// Error is an error encountered during Expand, we don't serialize this since we should not store error'ed graphs.
	Error error `json:"-"`
	// NextDriftResolution is the drift resolution to use in the _next_ graph expansion. This is usually set by the module
	// definition or by an operator/admin. We don't serialize this, since this is only used when expanding a new graph.
	NextDriftResolution DriftResolution `json:"-"`
}

func buildOrInheritFunc(inherit string) func(string) string {
	return func(s string) string {
		return strings.ReplaceAll(s, InheritClassOrIdSymbol, inherit)
	}
}

// SeedAndExpandAll is the constructor that seeds the graph and expands all iteratively until the end of the graph is reached.
// It then labels any cycles and identifies if errors were found.
// THIS IS THE CONSTRUCTOR YOU SHOULD USE.
func SeedAndExpandAll[D ModuleConfiguration](ctx context.Context, manifest Manifest, definitions ModuleDefinitionIndex[D], previousGraph *Graph[D]) (Graph[D], error) {
	g := Seed[D](ctx, manifest)
	for !g.IsExpanded() {
		g.Expand(ctx, definitions, previousGraph)
	}
	g.propagateCopyDependents()
	_ = g.LabelCycle()
	if g.HasErrors() {
		byNode := make(map[ResourceCoordinate]error)
		for k := range g.DepthFirstIterate(DepthFirstIteratePreOrder) {
			n := g.Nodes[k]
			if n.Error != nil {
				byNode[k] = g.Nodes[k].Error
			}
		}
		return g, &ErrGraph{ByNode: byNode}
	}
	return g, nil
}

// The Seed function is a constructor that creates the initial graph with leaf nodes and prepares for Expand.
func Seed[D ModuleConfiguration](ctx context.Context, manifest Manifest) Graph[D] {
	g := &Graph[D]{
		Workloads:       make(map[string]ResourceCoordinate),
		SharedResources: make(map[string]ResourceCoordinate),
		Nodes:           make(map[ResourceCoordinate]ResourceNode[D]),
		Edges:           make(map[ResourceCoordinate]map[string]ResourceCoordinate),
		unexpanded:      make([]ResourceCoordinate, 0),
		edgesLookup:     make(map[ResourceCoordinate]map[ResourceCoordinate]string),
	}

	for alias, res := range internal.SortedEntries(manifest.SharedResources) {
		coordinate := ResourceCoordinate{
			Type:  res.Type,
			Class: res.Class.ValueOr(DefaultClass),
			Id: res.Id.ValueOrFunc(func() string {
				return DefaultSharedResourceIdFunc(ctx, alias, res)
			}),
		}
		g.SharedResources[alias] = coordinate
		if n, ok := g.Nodes[coordinate]; !ok {
			g.Nodes[coordinate] = ResourceNode[D]{
				Params: res.Params,
			}
			if g.LeafResources == nil {
				g.LeafResources = make(map[ResourceCoordinate]bool)
			}
			g.LeafResources[coordinate] = true
			g.unexpanded = append(g.unexpanded, coordinate)
		} else if res.Params != nil && n.Params == nil {
			n.Params = res.Params
			g.Nodes[coordinate] = n
		} else if hasParamsError(n.Params, res.Params) {
			n.Error = errors.Join(n.Error, new(ErrParamsMismatch))
			g.Nodes[coordinate] = n
			g.hasErrors = true
		}
	}

	for workloadId, workload := range internal.SortedEntries(manifest.Workloads) {
		workloadCoordinate := ResourceCoordinate{
			Type:  DefaultWorkloadResourceType,
			Class: DefaultClass,
			Id:    workloadId,
		}
		params := make(map[string]interface{})
		for s, s2 := range workload.Outputs {
			params[s] = s2
		}
		g.Nodes[workloadCoordinate] = ResourceNode[D]{
			Params: params,
		}
		if g.LeafResources == nil {
			g.LeafResources = make(map[ResourceCoordinate]bool)
		}
		g.Workloads[workloadId] = workloadCoordinate
		g.LeafResources[workloadCoordinate] = true
		edges := make(map[string]ResourceCoordinate)
		g.Edges[workloadCoordinate] = edges
		edgesLookups := make(map[ResourceCoordinate]string)
		g.edgesLookup[workloadCoordinate] = edgesLookups

		for alias, res := range internal.SortedEntries(workload.Resources) {
			coordinate := ResourceCoordinate{
				Type:  res.Type,
				Class: res.Class.ValueOr(DefaultClass),
				Id: res.Id.ValueOrFunc(func() string {
					return DefaultWorkloadResourceIdFunc(ctx, workloadId, alias, res)
				}),
			}
			edges[alias] = coordinate
			edgesLookups[coordinate] = alias
			if n, ok := g.Nodes[coordinate]; !ok {
				g.Nodes[coordinate] = ResourceNode[D]{
					Params:          res.Params,
					ParamsDefinedBy: &workloadCoordinate,
				}
				g.unexpanded = append(g.unexpanded, coordinate)
			} else {
				if g.LeafResources != nil {
					// no longer a leaf node, since there's something that depends on it now
					delete(g.LeafResources, coordinate)
				}
				if res.Params != nil && n.Params == nil {
					n.Params = res.Params
				} else if hasParamsError(n.Params, res.Params) {
					n.Error = errors.Join(n.Error, new(ErrParamsMismatch))
					g.hasErrors = true
				}
				g.Nodes[coordinate] = n
			}
		}
	}
	return *g
}

func hasParamsError(a, b map[string]interface{}) bool {
	return a != nil && b != nil && !reflect.DeepEqual(a, b)
}

// HasErrors returns if any errors were found during expansion.
func (g *Graph[D]) HasErrors() bool {
	return g.hasErrors
}

// IsExpanded returns whether the graph is fully expanded. This is only useful if you are using Seed and Expand manually.
func (g *Graph[D]) IsExpanded() bool {
	return len(g.unexpanded) == 0
}

// Expand attempts to expand the next unexpanded node. This is only needed if you are using Seed and IsExpanded manually.
func (g *Graph[D]) Expand(ctx context.Context, definitions ModuleDefinitionIndex[D], previousGraph *Graph[D]) {
	if len(g.unexpanded) == 0 {
		return
	}

	var coordinate ResourceCoordinate
	coordinate, g.unexpanded = g.unexpanded[0], g.unexpanded[1:]

	node := g.Nodes[coordinate]
	// Reset the deprecated field to remove it from the graph
	node.CurrentDrift = 0

	// Now if there is no module match in the current rule set, we must set an error even if we're going to use
	// the node configuration from the previous graph.
	if m, ok := definitions.FindBest(ctx, coordinate); !ok {
		if previousGraph != nil {
			if prevNode, ok := previousGraph.Nodes[coordinate]; ok && !errors.Is(prevNode.Error, new(ErrNoRuleMatch)) {
				if prevNode.NextDriftResolution != DriftResolutionAcceptModuleChange {
					// We only accept this drop in rules if the drift resolution allows it. This is probably rare?
					node.DetectedDrift = DriftModuleChange
					node.ModuleConfiguration = prevNode.ModuleConfiguration
					g.Nodes[coordinate] = node
				}
			}
		}
		// if case 1 didn't match - then set the error
		if node.DetectedDrift == DriftNone {
			node.Error = errors.Join(node.Error, new(ErrNoRuleMatch))
			g.Nodes[coordinate] = node
			g.hasErrors = true
			return
		}
	} else {
		node.ModuleConfiguration = m.Configuration
		if previousGraph != nil {
			if prevNode, ok := previousGraph.Nodes[coordinate]; ok {
				node.DetectedDrift = node.ModuleConfiguration.CalculateDrift(prevNode.ModuleConfiguration)
				var accepted bool
				if node.EffectiveDrift, accepted = ApplyDriftResolution(node.DetectedDrift, prevNode.NextDriftResolution); !accepted {
					node.ModuleConfiguration = prevNode.ModuleConfiguration
				}
			}
		}
		g.Nodes[coordinate] = node
	}

	edges, ok := g.Edges[coordinate]
	if !ok {
		edges = make(map[string]ResourceCoordinate)
		g.Edges[coordinate] = edges
	}
	edgesLookups, ok := g.edgesLookup[coordinate]
	if !ok {
		edgesLookups = make(map[ResourceCoordinate]string)
		g.edgesLookup[coordinate] = edgesLookups
	}

	for alias, res := range internal.SortedEntries(node.ModuleConfiguration.GetDependencies()) {
		resCoordinate := ResourceCoordinate{
			Type:  res.Type,
			Class: res.Class.Map(buildOrInheritFunc(coordinate.Class)).ValueOr(DefaultClass),
			Id:    res.Id.Map(buildOrInheritFunc(coordinate.Id)).ValueOr(coordinate.Id),
		}
		if a, exist := edgesLookups[resCoordinate]; exist {
			// If the edge is already present, replace it with the new one, as only one edge is allowed between 2 nodes.
			// Most likely it's an anonymous edge added via co-provisioning.
			delete(edges, a)
		}
		edges[alias] = resCoordinate
		edgesLookups[resCoordinate] = alias
		if n, ok := g.Nodes[resCoordinate]; !ok {
			g.Nodes[resCoordinate] = ResourceNode[D]{
				Params:          res.Params,
				ParamsDefinedBy: &coordinate,
			}
			g.unexpanded = append(g.unexpanded, resCoordinate)
		} else {
			if g.LeafResources != nil {
				// no longer a leaf node, since there's something that depends on it now
				delete(g.LeafResources, resCoordinate)
			}
			if res.Params != nil && n.Params == nil {
				n.Params = res.Params
				n.ParamsDefinedBy = &coordinate
			} else if hasParamsError(n.Params, res.Params) {
				n.Error = errors.Join(n.Error, new(ErrParamsMismatch))
				g.hasErrors = true
			}
			g.Nodes[resCoordinate] = n
		}
	}

	for _, res := range node.ModuleConfiguration.GetCoProvisioned() {
		resCoordinate := ResourceCoordinate{
			Type:  res.Type,
			Class: res.Class.Map(buildOrInheritFunc(coordinate.Class)).ValueOr(DefaultClass),
			Id:    res.Id.Map(buildOrInheritFunc(coordinate.Id)).ValueOr(coordinate.Id),
		}
		if n, ok := g.Nodes[resCoordinate]; !ok {
			g.Nodes[resCoordinate] = ResourceNode[D]{
				Params:          res.Params,
				ParamsDefinedBy: &coordinate,
			}
			if g.LeafResources == nil {
				g.LeafResources = make(map[ResourceCoordinate]bool)
			}
			g.LeafResources[resCoordinate] = true
			g.unexpanded = append(g.unexpanded, resCoordinate)
		} else {
			if res.Params != nil && n.Params == nil {
				n.Params = res.Params
				n.ParamsDefinedBy = &coordinate
			} else if hasParamsError(n.Params, res.Params) {
				n.Error = errors.Join(n.Error, new(ErrParamsMismatch))
				g.hasErrors = true
			}
			g.Nodes[resCoordinate] = n
		}

		if res.IsDependentOnCurrent {
			_ = g.AddAnonymousDependency(resCoordinate, coordinate)
		}

		// If CopyDependentsOnCurrent is set, we want to copy all dependents on the current resource over to this
		// co-provisioned resource. However, due to iterative expansion, this can only happen at the end. So we
		// queue things up here.
		if res.CopyDependentsOnCurrent {
			if g.copyDependentsSet == nil {
				g.copyDependentsSet = make(map[ResourceCoordinate][]ResourceCoordinate, 1)
			}
			if e, ok := g.copyDependentsSet[resCoordinate]; ok {
				g.copyDependentsSet[resCoordinate] = append(e, coordinate)
			} else {
				g.copyDependentsSet[resCoordinate] = []ResourceCoordinate{coordinate}
			}
		}
	}
}

func (g *Graph[D]) propagateCopyDependents() {
	for target, sources := range internal.SortedEntriesFunc(g.copyDependentsSet, CompareResourceCoordinate) {
		sourceMap := make(map[ResourceCoordinate]bool, len(sources))
		for _, source := range sources {
			sourceMap[source] = true
		}
		for es, m := range g.Edges {
			if es != target {
				for _, intermediate := range m {
					if _, ok := sourceMap[intermediate]; ok {
						_ = g.AddAnonymousDependency(es, target)
					}
				}
			}
		}
	}
	g.copyDependentsSet = nil
}

func (g *Graph[D]) populateEdgesLookup() {
	g.edgesLookup = make(map[ResourceCoordinate]map[ResourceCoordinate]string, len(g.Edges))
	for from, m := range g.Edges {
		g.edgesLookup[from] = make(map[ResourceCoordinate]string, len(m))
		for alias, to := range m {
			g.edgesLookup[from][to] = alias
		}
	}
}

// DefaultAnonymousEdgeAliasFunc is the function used to determine an alias for an anonymous edge added due to
// placeholder resolution or coprovisioning rules.
var DefaultAnonymousEdgeAliasFunc = func(from, to ResourceCoordinate) string {
	h := sha256.New()
	_, _ = h.Write([]byte(to.Type))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(to.Class))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(to.Id))
	return base64.RawStdEncoding.EncodeToString(h.Sum(nil))
}

// AddAnonymousDependency can be used after the graph is expanded to add additional dependency edges to the graph. You
// should call LabelCycle after this to check if there is now a dependency cycle. The node is added with an alias that is
// a base64 encoded hash. The hash algorithm is DefaultAnonymousEdgeHash.
func (g *Graph[D]) AddAnonymousDependency(from ResourceCoordinate, to ResourceCoordinate) error {
	if _, ok := g.Nodes[from]; !ok {
		return fmt.Errorf("the 'from' node does not exist in the graph")
	} else if _, ok := g.Nodes[to]; !ok {
		return fmt.Errorf("the 'to' node does not exist in the graph")
	}
	edges := g.Edges[from]
	if edges == nil {
		edges = make(map[string]ResourceCoordinate)
		g.Edges[from] = edges
	}
	if g.edgesLookup == nil {
		// If the lookup map doesn't exist yet, it means that the dependency is added to the already expanded graph (restored from JSON).
		// Populate the lookup map from Edges.
		g.populateEdgesLookup()
	}
	edgesLookups := g.edgesLookup[from]
	if edgesLookups == nil {
		edgesLookups = make(map[ResourceCoordinate]string)
		g.edgesLookup[from] = edgesLookups
	}
	// don't add an anonymous edge if we already have a dependency link
	if internal.ContainsSeq(maps.Values(edges), to) {
		return nil
	}
	alias := DefaultAnonymousEdgeAliasFunc(from, to)
	edges[alias] = to
	edgesLookups[to] = alias
	if g.LeafResources != nil {
		delete(g.LeafResources, to)
	}
	return nil
}

// LabelCycle will attempt to find a cycle in the graph. This does not find _all_ cycles. This is called automatically
// within SeedAndExpandAll but can be called again if you have added additional edges to the graph through AddAnonymousDependency.
func (g *Graph[D]) LabelCycle() map[ResourceCoordinate]bool {
	visited := make(map[ResourceCoordinate]bool)
	recursionStack := make(map[ResourceCoordinate]bool)
	cycleNodes := make(map[ResourceCoordinate]bool)

	var dfs func(ResourceCoordinate, ResourceCoordinate) bool
	dfs = func(coordinate ResourceCoordinate, parentCoordinate ResourceCoordinate) bool {
		node := g.Nodes[coordinate]
		visited[coordinate] = true
		recursionStack[coordinate] = true

		for _, neighbour := range internal.SortedEntries(g.Edges[coordinate]) {
			if !visited[neighbour] {
				if dfs(neighbour, parentCoordinate) {
					if cycleNodes[neighbour] {
						cycleNodes[coordinate] = true
						node.Error = errors.Join(node.Error, new(ErrCycle))
						g.Nodes[coordinate] = node
						g.hasErrors = true
					}
					return true
				}
			} else if parentCoordinate != neighbour && recursionStack[neighbour] {
				cycleNodes[coordinate] = true
				cycleNodes[neighbour] = true
				node.Error = errors.Join(node.Error, new(ErrCycle))
				g.Nodes[coordinate] = node
				g.hasErrors = true
				return true
			}
		}

		recursionStack[coordinate] = false
		return false
	}

	for node := range internal.SortedKeysFunc(g.Nodes, CompareResourceCoordinate) {
		if !visited[node] {
			dfs(node, ResourceCoordinate{})
		}
	}
	return cycleNodes
}

func (g *Graph[D]) depthFirstIterateInner(nk ResourceCoordinate, visited map[ResourceCoordinate]bool, order DepthFirstIterateOrder, visitor func(coordinate ResourceCoordinate) bool) bool {
	visited[nk] = true
	if order == DepthFirstIteratePreOrder && !visitor(nk) {
		return false
	}
	for _, key := range internal.SortedEntries(g.Edges[nk]) {
		if !visited[key] && !g.depthFirstIterateInner(key, visited, order, visitor) {
			return false
		}
	}
	return order != DepthFirstIteratePostOrder || visitor(nk)
}

type DepthFirstIterateOrder int

const (
	DepthFirstIteratePreOrder DepthFirstIterateOrder = iota
	DepthFirstIteratePostOrder
)

// DepthFirstIterateFrom iterates through the subgraph rooted at the given resource coordinate.
func (g *Graph[D]) DepthFirstIterateFrom(c ResourceCoordinate, o DepthFirstIterateOrder) iter.Seq[ResourceCoordinate] {
	return func(yield func(ResourceCoordinate) bool) {
		g.depthFirstIterateInner(c, make(map[ResourceCoordinate]bool), o, yield)
	}
}

// DepthFirstIterate iterates through the entire graph in a depth first way. This is useful to make sure that all dependencies
// are processed or visited first.
func (g *Graph[D]) DepthFirstIterate(o DepthFirstIterateOrder) iter.Seq[ResourceCoordinate] {
	return func(yield func(ResourceCoordinate) bool) {
		visited := make(map[ResourceCoordinate]bool)
		// first go through the leaf resources
		for nk := range internal.SortedKeysFunc(g.LeafResources, CompareResourceCoordinate) {
			if !visited[nk] && !g.depthFirstIterateInner(nk, visited, o, yield) {
				return
			}
		}
		if len(visited) < len(g.Nodes) {
			// then anything that remains that may be part of a cycle
			for nk := range internal.SortedKeysFunc(g.Nodes, CompareResourceCoordinate) {
				if !visited[nk] && !g.depthFirstIterateInner(nk, visited, o, yield) {
					return
				}
			}
		}
	}
}

type DistanceMatrix map[ResourceCoordinate]map[ResourceCoordinate]int

// BuildAdjacencyMatrix returns a DistanceMatrix which contains only the immediate neighbours set to distance 1.
func (g *Graph[D]) BuildAdjacencyMatrix() DistanceMatrix {
	dist := make(DistanceMatrix)
	for from, m := range g.Edges {
		for _, to := range m {
			if d, ok := dist[from]; ok {
				d[to] = 1
			} else {
				dist[from] = map[ResourceCoordinate]int{to: 1}
			}
			if _, ok := dist[to]; !ok {
				dist[to] = make(map[ResourceCoordinate]int)
			}
		}
	}
	for coordinate := range g.LeafResources {
		if _, ok := dist[coordinate]; !ok {
			dist[coordinate] = make(map[ResourceCoordinate]int)
		}
	}
	return dist
}

// FillDistanceMatrix builds a full distance matrix. Looking up any node should produce all
// the nodes that are dependencies in some way by their minimum depth. Nodes that don't have any dependencies are excluded.
// This uses the Floyd-Warshall (https://en.wikipedia.org/wiki/Floyd%E2%80%93Warshall_algorithm) algorithm.
func (d DistanceMatrix) FillDistanceMatrix() DistanceMatrix {
	for k := range d {
		for i := range d {
			for j := range d {
				ik, ikOk := d[i][k]
				kj, kjOk := d[k][j]
				if ikOk && kjOk {
					if ij, ijOk := d[i][j]; !ijOk || ij > (ik+kj) {
						if dd, ok := d[i]; ok {
							dd[j] = ik + kj
						} else {
							d[i] = map[ResourceCoordinate]int{j: ik + kj}
						}
					}
				}
			}
		}
	}
	return d
}

// FindDescendentsOf returns an iteration over all the resources that the given resource c depends on, directly and
// transitively through other dependencies. The integer value is the depth. This is more efficient than the DepthFirstIterate
// function but doesn't return in any particular order.
func (d DistanceMatrix) FindDescendentsOf(c ResourceCoordinate) iter.Seq2[ResourceCoordinate, int] {
	return maps.All(d[c])
}

// applyExpansionFunction is used by ResolveSelectorPlaceholder
func applyExpansionFunction(set []ResourceCoordinate, expander func(coordinate ResourceCoordinate) iter.Seq2[ResourceCoordinate, int], filter SelectorFilter, maxDepth int) []ResourceCoordinate {
	collected := maps.Collect(func(yield func(ResourceCoordinate, bool) bool) {
		for _, setItem := range set {
			for rc, depth := range expander(setItem) {
				if filter.Matches(rc, setItem) && depth <= maxDepth {
					if !yield(rc, true) {
						return
					}
				}
			}
		}
	})
	return slices.SortedFunc(maps.Keys(collected), CompareResourceCoordinate)
}

// ResolveSelectorPlaceholder fills the MatchedCoordinates field on the selector placeholder with the correct matched nodes based on the distance matrix.
func (d DistanceMatrix) ResolveSelectorPlaceholder(context ResourceCoordinate, sp *SelectorPlaceholder) error {
	nodeSet := []ResourceCoordinate{context}
	for i, funcName := range sp.Functions {
		filter := ParseSelectorFilter(sp.Args[i])
		switch funcName {
		case "consumers":
			nodeSet = applyExpansionFunction(nodeSet, d.FindAncestorsOf, filter, 1)
		case "dependencies":
			nodeSet = applyExpansionFunction(nodeSet, d.FindDescendentsOf, filter, 1)
		default:
			return fmt.Errorf("unknown selector placeholder function '%s'", funcName)
		}
	}
	sp.MatchedCoordinates = nodeSet
	return nil
}

// FindAncestorsOf returns an iteration over all the resources that depend on c, directly and  transitively through.
// The integer value is the depth.
func (d DistanceMatrix) FindAncestorsOf(c ResourceCoordinate) iter.Seq2[ResourceCoordinate, int] {
	return func(yield func(ResourceCoordinate, int) bool) {
		for k, v := range d {
			if i, ok := v[c]; ok {
				if !yield(k, i) {
					return
				}
			}
		}
	}
}

// ResolveResourcePlaceholder fills the Coordinate field on the resource placeholder with the correct node.
func (g *Graph[D]) ResolveResourcePlaceholder(aliasContext ResourceCoordinate, rt *ResourcePlaceholder) error {
	// only workload resources can use the "shared" aliases, because those aliases only exist in the top level manifest which defines the workloads themselves.
	if rt.IsShared {
		if !internal.ContainsSeq(maps.Values(g.Workloads), aliasContext) {
			return fmt.Errorf("shared resource placeholders can only be used in a workload context")
		} else if resC, ok := g.SharedResources[rt.Alias]; ok {
			rt.Coordinate = &resC
			return nil
		} else {
			return fmt.Errorf("no shared resource with alias '%s' exists", rt.Alias)
		}
	} else if rt.IsSelf {
		// self just returns the coordinate of the context node
		rt.Coordinate = &aliasContext
		return nil
	} else if resC, ok := g.Edges[aliasContext][rt.Alias]; ok {
		rt.Coordinate = &resC
		return nil
	} else {
		return fmt.Errorf("no resource dependency with alias '%s' exists", rt.Alias)
	}
}
