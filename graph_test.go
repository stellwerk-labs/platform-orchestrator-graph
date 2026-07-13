package orchestrator_graph

import (
	"fmt"
	"iter"
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stellwerk-labs/platform-orchestrator-graph/internal"
)

func TestSeed_empty(t *testing.T) {
	g := Seed[*FakeModuleConfiguration](t.Context(), Manifest{})
	assert.True(t, g.IsExpanded())
	assert.False(t, g.HasErrors())
	assert.Equal(t, 0, internal.CountSeq(g.DepthFirstIterate(DepthFirstIteratePostOrder)))
	assert.Empty(t, g.LabelCycle())
	g.Expand(t.Context(), *NewModuleDefinitionIndex([]ModuleDefinition[*FakeModuleConfiguration]{}), nil)
}

func TestSeed_some(t *testing.T) {
	g := Seed[*FakeModuleConfiguration](t.Context(), Manifest{
		Workloads: map[string]ManifestWorkload{
			"wA": {
				Resources: map[string]ManifestResource{
					"a": {Type: "tA"},
					"b": {Type: "tB", Class: OptionalStringOf("cB"), Id: OptionalStringOf("iB")},
					"c": {Type: "tC", Id: OptionalStringOf("iC"), Params: map[string]interface{}{"x": "y"}},
					"d": {Type: "tD", Id: OptionalStringOf("iE"), Params: map[string]interface{}{"x": "y"}},
					"e": {Type: "tD", Id: OptionalStringOf("iE"), Params: map[string]interface{}{"x": "z"}},
				},
			},
		},
		SharedResources: map[string]ManifestResource{
			"a": {Type: "tA"},
			"b": {Type: "tB"},
			"c": {Type: "tC", Id: OptionalStringOf("iC")},
			"d": {Type: "tD", Id: OptionalStringOf("iD"), Params: map[string]interface{}{"x": "y"}},
			"e": {Type: "tD", Id: OptionalStringOf("iD"), Params: map[string]interface{}{"x": "z"}},
			"f": {Type: "tF", Id: OptionalStringOf("iE")},
			"g": {Type: "tF", Id: OptionalStringOf("iE"), Params: map[string]interface{}{"x": "y"}},
		},
	})
	assert.False(t, g.IsExpanded())
	assert.True(t, g.HasErrors())
	assert.Equal(t, []ResourceCoordinate{
		{Type: "tA", Class: DefaultClass, Id: "shared.a"},
		{Type: "tA", Class: DefaultClass, Id: "workloads.wA.a"},
		{Type: "tB", Class: "cB", Id: "iB"},
		{Type: "tB", Class: DefaultClass, Id: "shared.b"},
		{Type: "tC", Class: DefaultClass, Id: "iC"},
		{Type: "tD", Class: DefaultClass, Id: "iD"},
		{Type: "tD", Class: DefaultClass, Id: "iE"},
		{Type: "tF", Class: DefaultClass, Id: "iE"},
		{Type: "workload", Class: DefaultClass, Id: "wA"},
	}, slices.SortedFunc(g.DepthFirstIterate(DepthFirstIteratePostOrder), CompareResourceCoordinate))
	assert.Equal(t, map[string]ResourceCoordinate{
		"wA": {Type: "workload", Class: DefaultClass, Id: "wA"},
	}, g.Workloads)
	assert.Equal(t, map[string]ResourceCoordinate{
		"a": {Type: "tA", Class: DefaultClass, Id: "shared.a"},
		"b": {Type: "tB", Class: DefaultClass, Id: "shared.b"},
		"c": {Type: "tC", Class: DefaultClass, Id: "iC"},
		"d": {Type: "tD", Class: DefaultClass, Id: "iD"},
		"e": {Type: "tD", Class: DefaultClass, Id: "iD"},
		"f": {Type: "tF", Class: DefaultClass, Id: "iE"},
		"g": {Type: "tF", Class: DefaultClass, Id: "iE"},
	}, g.SharedResources)
	assert.Equal(t, map[ResourceCoordinate]bool{
		ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "shared.a"}: true,
		ResourceCoordinate{Type: "tB", Class: DefaultClass, Id: "shared.b"}: true,
		ResourceCoordinate{Type: "tD", Class: DefaultClass, Id: "iD"}:       true,
		ResourceCoordinate{Type: "tF", Class: DefaultClass, Id: "iE"}:       true,
		ResourceCoordinate{Type: "workload", Class: DefaultClass, Id: "wA"}: true,
	}, g.LeafResources)
	assert.Equal(t, []ResourceCoordinate{
		{Type: "tA", Class: DefaultClass, Id: "shared.a"},
		{Type: "tA", Class: DefaultClass, Id: "workloads.wA.a"},
		{Type: "tB", Class: "cB", Id: "iB"},
		{Type: "tB", Class: DefaultClass, Id: "shared.b"},
		{Type: "tC", Class: DefaultClass, Id: "iC"},
		{Type: "tD", Class: DefaultClass, Id: "iD"},
		{Type: "tD", Class: DefaultClass, Id: "iE"},
		{Type: "tF", Class: DefaultClass, Id: "iE"},
		{Type: "workload", Class: DefaultClass, Id: "wA"},
	}, slices.SortedFunc(maps.Keys(g.Nodes), CompareResourceCoordinate))
	assert.Equal(t, []string{
		"tD,default,iD: referenced multiple times in the graph with different resource parameters",
		"tD,default,iE: referenced multiple times in the graph with different resource parameters",
	}, slices.Collect(func(yield func(string) bool) {
		for k := range g.DepthFirstIterate(DepthFirstIteratePostOrder) {
			n := g.Nodes[k]
			if n.Error != nil {
				yield(fmt.Sprintf("%s,%s,%s: %v", k.Type, k.Class, k.Id, n.Error))
			}
		}
	}))
	assert.Equal(t, DistanceMatrix{
		ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "shared.a"}:       map[ResourceCoordinate]int{},
		ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "workloads.wA.a"}: map[ResourceCoordinate]int{},
		ResourceCoordinate{Type: "tB", Class: "cB", Id: "iB"}:                     map[ResourceCoordinate]int{},
		ResourceCoordinate{Type: "tB", Class: DefaultClass, Id: "shared.b"}:       map[ResourceCoordinate]int{},
		ResourceCoordinate{Type: "tC", Class: DefaultClass, Id: "iC"}:             map[ResourceCoordinate]int{},
		ResourceCoordinate{Type: "tD", Class: DefaultClass, Id: "iD"}:             map[ResourceCoordinate]int{},
		ResourceCoordinate{Type: "tD", Class: DefaultClass, Id: "iE"}:             map[ResourceCoordinate]int{},
		ResourceCoordinate{Type: "tF", Class: DefaultClass, Id: "iE"}:             map[ResourceCoordinate]int{},
		ResourceCoordinate{Type: "workload", Class: DefaultClass, Id: "wA"}: map[ResourceCoordinate]int{
			ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "workloads.wA.a"}: 1,
			ResourceCoordinate{Type: "tB", Class: "cB", Id: "iB"}:                     1,
			ResourceCoordinate{Type: "tC", Class: DefaultClass, Id: "iC"}:             1,
			ResourceCoordinate{Type: "tD", Class: DefaultClass, Id: "iE"}:             1,
		},
	}, g.BuildAdjacencyMatrix().FillDistanceMatrix())
}

func TestExpand_simple_match(t *testing.T) {
	g := Seed[*FakeModuleConfiguration](t.Context(), Manifest{
		SharedResources: map[string]ManifestResource{
			"a": {Type: "tA"},
		},
	})
	assert.False(t, g.IsExpanded())
	g.Expand(t.Context(), *NewModuleDefinitionIndex[*FakeModuleConfiguration]([]ModuleDefinition[*FakeModuleConfiguration]{
		{ResourceType: "tA", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{Id: 1}},
	}), nil)
	assert.True(t, g.IsExpanded())
	assert.Empty(t, g.LabelCycle())
	node := g.Nodes[ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "shared.a"}]
	assert.Equal(t, DriftNone, node.DetectedDrift)
	assert.Equal(t, &FakeModuleConfiguration{Id: 1}, node.ModuleConfiguration)
	assert.Equal(t, DistanceMatrix{
		ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "shared.a"}: map[ResourceCoordinate]int{},
	}, g.BuildAdjacencyMatrix().FillDistanceMatrix())
}

func TestExpand_simple_no_match(t *testing.T) {
	g := Seed[*FakeModuleConfiguration](t.Context(), Manifest{
		SharedResources: map[string]ManifestResource{
			"a": {Type: "tA"},
		},
	})
	assert.False(t, g.IsExpanded())
	g.Expand(t.Context(), *NewModuleDefinitionIndex[*FakeModuleConfiguration]([]ModuleDefinition[*FakeModuleConfiguration]{}), nil)
	assert.True(t, g.IsExpanded())
	assert.Empty(t, g.LabelCycle())
	assert.True(t, g.HasErrors())
	node := g.Nodes[ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "shared.a"}]
	assert.Equal(t, DriftNone, node.DetectedDrift)
	assert.EqualError(t, node.Error, "no module definition matches this resource")
}

func TestExpand_simple_dependencies(t *testing.T) {
	index := NewModuleDefinitionIndex[*FakeModuleConfiguration]([]ModuleDefinition[*FakeModuleConfiguration]{
		{ResourceType: "tA", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Id:           1,
			Dependencies: map[string]ManifestResource{"child": {Type: "tB"}},
			Dependents:   []ManifestCoProvision{{ManifestResource: ManifestResource{Type: "tB", Id: OptionalStringOf("parent")}}},
		}},
		{ResourceType: "tB", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{Id: 2}},
	})
	g, err := SeedAndExpandAll[*FakeModuleConfiguration](t.Context(), Manifest{
		Workloads: map[string]ManifestWorkload{
			"thing": {Resources: map[string]ManifestResource{
				"a": {Type: "tA", Id: OptionalStringOf("shared.a")},
			}},
		},
		SharedResources: map[string]ManifestResource{
			"a": {Type: "tA"},
		},
	}, *index, nil)
	require.NoError(t, err)
	assert.Equal(t, map[ResourceCoordinate]map[string]ResourceCoordinate{
		{Type: "tA", Class: DefaultClass, Id: "shared.a"}: {
			"child": {Type: "tB", Class: DefaultClass, Id: "shared.a"},
		},
		{Type: "tB", Class: DefaultClass, Id: "parent"}:   {},
		{Type: "tB", Class: DefaultClass, Id: "shared.a"}: {},
		{Type: "workload", Class: DefaultClass, Id: "thing"}: {
			"a": {Type: "tA", Class: DefaultClass, Id: "shared.a"},
		},
	}, g.Edges)
	assert.Equal(t, map[ResourceCoordinate]bool{
		ResourceCoordinate{Type: "tB", Class: DefaultClass, Id: "parent"}:      true,
		ResourceCoordinate{Type: "workload", Class: DefaultClass, Id: "thing"}: true,
	}, g.LeafResources)
	assert.Equal(t, DistanceMatrix{
		ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "shared.a"}: map[ResourceCoordinate]int{
			ResourceCoordinate{Type: "tB", Class: DefaultClass, Id: "shared.a"}: 1,
		},
		ResourceCoordinate{Type: "tB", Class: DefaultClass, Id: "shared.a"}: map[ResourceCoordinate]int{},
		ResourceCoordinate{Type: "tB", Class: DefaultClass, Id: "parent"}:   map[ResourceCoordinate]int{},
		ResourceCoordinate{Type: "workload", Class: DefaultClass, Id: "thing"}: map[ResourceCoordinate]int{
			ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "shared.a"}: 1,
			ResourceCoordinate{Type: "tB", Class: DefaultClass, Id: "shared.a"}: 2,
		},
	}, g.BuildAdjacencyMatrix().FillDistanceMatrix())
}

func TestExpand_simple_edges_from_params(t *testing.T) {
	index := NewModuleDefinitionIndex[*FakeModuleConfiguration]([]ModuleDefinition[*FakeModuleConfiguration]{
		{ResourceType: "tA", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{Id: 1}},
		{ResourceType: "tB", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{Id: 2}},
	})
	g, err := SeedAndExpandAll[*FakeModuleConfiguration](t.Context(), Manifest{
		Workloads: map[string]ManifestWorkload{
			"thing": {Resources: map[string]ManifestResource{
				"a": {Type: "tA"},
				"b": {
					Type: "tB",
					Params: map[string]interface{}{
						"data": map[string]interface{}{
							"test": "${resources.a.outputs.test}",
						},
					},
				},
			}},
		},
	}, *index, nil)
	require.NoError(t, err)
	rp := &ResourcePlaceholder{Alias: "a", Output: []string{"test"}}
	require.NoError(t, g.ResolveResourcePlaceholder(ResourceCoordinate{Type: DefaultWorkloadResourceType, Class: DefaultClass, Id: "thing"}, rp))
	assert.Equal(t, ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "workloads.thing.a"}, *rp.Coordinate)
	require.NoError(t, g.AddAnonymousDependency(ResourceCoordinate{Type: "tB", Class: DefaultClass, Id: "workloads.thing.b"}, *rp.Coordinate))
	assert.Equal(t, map[ResourceCoordinate]map[string]ResourceCoordinate{
		{Type: "tA", Class: DefaultClass, Id: "workloads.thing.a"}: {},
		{Type: "tB", Class: DefaultClass, Id: "workloads.thing.b"}: {
			"YT+9yho8vboHHzSyuWyrtbEJc9D1WsVuI2D4AHlgulE": {Type: "tA", Class: DefaultClass, Id: "workloads.thing.a"},
		},
		{Type: "workload", Class: DefaultClass, Id: "thing"}: {
			"a": {Type: "tA", Class: DefaultClass, Id: "workloads.thing.a"},
			"b": {Type: "tB", Class: DefaultClass, Id: "workloads.thing.b"},
		},
	}, g.Edges)
	assert.Equal(t, map[ResourceCoordinate]bool{
		ResourceCoordinate{Type: "workload", Class: DefaultClass, Id: "thing"}: true,
	}, g.LeafResources)
	assert.Equal(t, DistanceMatrix{
		ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "workloads.thing.a"}: map[ResourceCoordinate]int{},
		ResourceCoordinate{Type: "tB", Class: DefaultClass, Id: "workloads.thing.b"}: map[ResourceCoordinate]int{
			ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "workloads.thing.a"}: 1,
		},
		ResourceCoordinate{Type: "workload", Class: DefaultClass, Id: "thing"}: map[ResourceCoordinate]int{
			ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "workloads.thing.a"}: 1,
			ResourceCoordinate{Type: "tB", Class: DefaultClass, Id: "workloads.thing.b"}: 1,
		},
	}, g.BuildAdjacencyMatrix().FillDistanceMatrix())
}

func TestBuildFullDistanceMatrix(t *testing.T) {
	g, err := SeedAndExpandAll[*FakeModuleConfiguration](
		t.Context(),
		Manifest{SharedResources: map[string]ManifestResource{
			"a": {Type: "tA"},
		}},
		*NewModuleDefinitionIndex[*FakeModuleConfiguration]([]ModuleDefinition[*FakeModuleConfiguration]{
			{ResourceType: "tA", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
				Dependencies: map[string]ManifestResource{"x": {Type: "tC"}, "y": {Type: "tD"}},
				Dependents:   []ManifestCoProvision{{ManifestResource: ManifestResource{Type: "tB"}, IsDependentOnCurrent: true}},
			}},
			{ResourceType: "tB", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{}},
			{ResourceType: "tC", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
				Dependencies: map[string]ManifestResource{"x": {Type: "tE"}},
			}},
			{ResourceType: "tD", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
				Dependencies: map[string]ManifestResource{"x": {Type: "tE"}},
			}},
			{ResourceType: "tE", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
				Dependencies: map[string]ManifestResource{"x": {Type: "tF"}},
			}},
			{ResourceType: "tF", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{}},
		}),
		nil,
	)
	require.NoError(t, err)
	assert.Equal(t, map[ResourceCoordinate]bool{
		ResourceCoordinate{Type: "tB", Class: DefaultClass, Id: "shared.a"}: true,
	}, g.LeafResources)
	assert.Equal(t, DistanceMatrix{
		ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "shared.a"}: map[ResourceCoordinate]int{
			ResourceCoordinate{Type: "tC", Class: DefaultClass, Id: "shared.a"}: 1,
			ResourceCoordinate{Type: "tD", Class: DefaultClass, Id: "shared.a"}: 1,
		},
		ResourceCoordinate{Type: "tB", Class: DefaultClass, Id: "shared.a"}: map[ResourceCoordinate]int{
			ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "shared.a"}: 1,
		},
		ResourceCoordinate{Type: "tC", Class: DefaultClass, Id: "shared.a"}: map[ResourceCoordinate]int{
			ResourceCoordinate{Type: "tE", Class: DefaultClass, Id: "shared.a"}: 1,
		},
		ResourceCoordinate{Type: "tD", Class: DefaultClass, Id: "shared.a"}: map[ResourceCoordinate]int{
			ResourceCoordinate{Type: "tE", Class: DefaultClass, Id: "shared.a"}: 1,
		},
		ResourceCoordinate{Type: "tE", Class: DefaultClass, Id: "shared.a"}: map[ResourceCoordinate]int{
			ResourceCoordinate{Type: "tF", Class: DefaultClass, Id: "shared.a"}: 1,
		},
		ResourceCoordinate{Type: "tF", Class: DefaultClass, Id: "shared.a"}: map[ResourceCoordinate]int{},
	}, g.BuildAdjacencyMatrix())
	assert.Equal(t, DistanceMatrix{
		ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "shared.a"}: map[ResourceCoordinate]int{
			ResourceCoordinate{Type: "tC", Class: DefaultClass, Id: "shared.a"}: 1,
			ResourceCoordinate{Type: "tD", Class: DefaultClass, Id: "shared.a"}: 1,
			ResourceCoordinate{Type: "tE", Class: DefaultClass, Id: "shared.a"}: 2,
			ResourceCoordinate{Type: "tF", Class: DefaultClass, Id: "shared.a"}: 3,
		},
		ResourceCoordinate{Type: "tB", Class: DefaultClass, Id: "shared.a"}: map[ResourceCoordinate]int{
			ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "shared.a"}: 1,
			ResourceCoordinate{Type: "tC", Class: DefaultClass, Id: "shared.a"}: 2,
			ResourceCoordinate{Type: "tD", Class: DefaultClass, Id: "shared.a"}: 2,
			ResourceCoordinate{Type: "tE", Class: DefaultClass, Id: "shared.a"}: 3,
			ResourceCoordinate{Type: "tF", Class: DefaultClass, Id: "shared.a"}: 4,
		},
		ResourceCoordinate{Type: "tC", Class: DefaultClass, Id: "shared.a"}: map[ResourceCoordinate]int{
			ResourceCoordinate{Type: "tE", Class: DefaultClass, Id: "shared.a"}: 1,
			ResourceCoordinate{Type: "tF", Class: DefaultClass, Id: "shared.a"}: 2,
		},
		ResourceCoordinate{Type: "tD", Class: DefaultClass, Id: "shared.a"}: map[ResourceCoordinate]int{
			ResourceCoordinate{Type: "tE", Class: DefaultClass, Id: "shared.a"}: 1,
			ResourceCoordinate{Type: "tF", Class: DefaultClass, Id: "shared.a"}: 2,
		},
		ResourceCoordinate{Type: "tE", Class: DefaultClass, Id: "shared.a"}: map[ResourceCoordinate]int{
			ResourceCoordinate{Type: "tF", Class: DefaultClass, Id: "shared.a"}: 1,
		},
		ResourceCoordinate{Type: "tF", Class: DefaultClass, Id: "shared.a"}: map[ResourceCoordinate]int{},
	}, g.BuildAdjacencyMatrix().FillDistanceMatrix())
}

func TestExpand_simple_cycle_1(t *testing.T) {
	g := Seed[*FakeModuleConfiguration](t.Context(), Manifest{
		SharedResources: map[string]ManifestResource{
			"a": {Type: "tA"},
		},
	})
	assert.False(t, g.IsExpanded())
	index := NewModuleDefinitionIndex[*FakeModuleConfiguration]([]ModuleDefinition[*FakeModuleConfiguration]{
		{ResourceType: "tA", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Id:           1,
			Dependencies: map[string]ManifestResource{"child": {Type: "tA"}},
		}},
	})
	g.Expand(t.Context(), *index, nil)
	assert.True(t, g.IsExpanded())
	assert.Equal(t, map[ResourceCoordinate]bool{
		ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "shared.a"}: true,
	}, g.LabelCycle())
	assert.True(t, g.HasErrors())
	assert.EqualError(t, g.Nodes[ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "shared.a"}].Error, "cycle detected")
}

func TestExpand_simple_cycle_2(t *testing.T) {
	index := NewModuleDefinitionIndex[*FakeModuleConfiguration]([]ModuleDefinition[*FakeModuleConfiguration]{
		{ResourceType: "tA", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Id:           1,
			Dependencies: map[string]ManifestResource{"child": {Type: "tB"}},
		}},
		{ResourceType: "tB", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Id:           1,
			Dependencies: map[string]ManifestResource{"child": {Type: "tA"}},
		}},
	})
	g, err := SeedAndExpandAll[*FakeModuleConfiguration](t.Context(), Manifest{
		SharedResources: map[string]ManifestResource{
			"a": {Type: "tA"},
		},
	}, *index, nil)
	require.EqualError(t, err, `graph contains 2 errors:
	type=tA,class=default,id=shared.a: cycle detected
	type=tB,class=default,id=shared.a: cycle detected
`,
	)
	assert.NotNil(t, g)
}

func TestDrift_no_drift(t *testing.T) {
	index := NewModuleDefinitionIndex[*FakeModuleConfiguration]([]ModuleDefinition[*FakeModuleConfiguration]{
		{ResourceType: "tA", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Id:           1,
			Dependencies: map[string]ManifestResource{"child": {Type: "tB"}},
		}},
		{ResourceType: "tB", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Id:           1,
			Dependencies: map[string]ManifestResource{},
		}},
	})
	g, err := SeedAndExpandAll[*FakeModuleConfiguration](t.Context(), Manifest{
		Workloads: map[string]ManifestWorkload{
			"child": {
				Resources: map[string]ManifestResource{
					"a": {Type: "tA"},
				},
			},
		},
	}, *index, nil)
	require.NoError(t, err)

	g2, err := SeedAndExpandAll[*FakeModuleConfiguration](t.Context(), Manifest{
		SharedResources: map[string]ManifestResource{
			"a": {Type: "tA"},
		},
	}, *index, &g)
	require.NoError(t, err)
	for k := range g2.DepthFirstIterate(DepthFirstIteratePostOrder) {
		assert.Equal(t, DriftNone, g2.Nodes[k].DetectedDrift, "should not drift", k)
	}
}

func TestDrift_drift_version(t *testing.T) {
	index := NewModuleDefinitionIndex[*FakeModuleConfiguration]([]ModuleDefinition[*FakeModuleConfiguration]{
		{ResourceType: "tA", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Id:           1,
			Dependencies: map[string]ManifestResource{"child": {Type: "tB"}},
		}},
		{ResourceType: "tB", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Id:           1,
			Dependencies: map[string]ManifestResource{},
		}},
	})
	g, err := SeedAndExpandAll[*FakeModuleConfiguration](t.Context(), Manifest{
		SharedResources: map[string]ManifestResource{
			"a": {Type: "tA"},
		},
	}, *index, nil)
	require.NoError(t, err)

	index = NewModuleDefinitionIndex[*FakeModuleConfiguration]([]ModuleDefinition[*FakeModuleConfiguration]{
		{ResourceType: "tA", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Id: 1, Version: 2,
			Dependencies: map[string]ManifestResource{"child": {Type: "tB"}},
		}},
		{ResourceType: "tB", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Id:           1,
			Dependencies: map[string]ManifestResource{},
		}},
	})

	g2, err := SeedAndExpandAll[*FakeModuleConfiguration](t.Context(), Manifest{
		SharedResources: map[string]ManifestResource{
			"a": {Type: "tA"},
		},
	}, *index, &g)
	require.NoError(t, err)
	assert.Equal(t, DriftConfigurationChange, g2.Nodes[ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "shared.a"}].DetectedDrift)
	assert.Equal(t, DriftConfigurationChangeSkipped, g2.Nodes[ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "shared.a"}].EffectiveDrift)
	assert.Equal(t, DriftNone, g2.Nodes[ResourceCoordinate{Type: "tB", Class: DefaultClass, Id: "shared.a"}].DetectedDrift)

	{
		coord := ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "shared.a"}
		n := g2.Nodes[coord]
		n.NextDriftResolution = DriftResolutionAcceptConfigChange
		g2.Nodes[coord] = n
	}

	g3, err := SeedAndExpandAll[*FakeModuleConfiguration](t.Context(), Manifest{
		SharedResources: map[string]ManifestResource{
			"a": {Type: "tA"},
		},
	}, *index, &g2)
	require.NoError(t, err)
	assert.Equal(t, DriftNone, g3.Nodes[ResourceCoordinate{Type: "tA", Id: "shared.a"}].DetectedDrift)
	assert.Equal(t, DriftNone, g3.Nodes[ResourceCoordinate{Type: "tB", Id: "shared.a"}].DetectedDrift)
}

func TestDrift_drift_module_change(t *testing.T) {
	index := NewModuleDefinitionIndex[*FakeModuleConfiguration]([]ModuleDefinition[*FakeModuleConfiguration]{
		{ResourceType: "tA", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Id:           1,
			Dependencies: map[string]ManifestResource{"child": {Type: "tB"}},
		}},
		{ResourceType: "tB", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Id:           1,
			Dependencies: map[string]ManifestResource{},
		}},
	})
	g, err := SeedAndExpandAll[*FakeModuleConfiguration](t.Context(), Manifest{
		SharedResources: map[string]ManifestResource{
			"a": {Type: "tA"},
		},
	}, *index, nil)
	require.NoError(t, err)

	index = NewModuleDefinitionIndex[*FakeModuleConfiguration]([]ModuleDefinition[*FakeModuleConfiguration]{
		{ResourceType: "tA", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Id:           1,
			Dependencies: map[string]ManifestResource{"child": {Type: "tB"}},
		}},
		{ResourceType: "tB", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Id:           2,
			Dependencies: map[string]ManifestResource{},
		}},
	})
	g2, err := SeedAndExpandAll[*FakeModuleConfiguration](t.Context(), Manifest{
		SharedResources: map[string]ManifestResource{
			"a": {Type: "tA"},
		},
	}, *index, &g)
	require.NoError(t, err)
	assert.Equal(t, DriftNone, g2.Nodes[ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "shared.a"}].DetectedDrift)
	assert.Equal(t, DriftModuleChange, g2.Nodes[ResourceCoordinate{Type: "tB", Class: DefaultClass, Id: "shared.a"}].DetectedDrift)
	assert.Equal(t, DriftModuleChangeSkipped, g2.Nodes[ResourceCoordinate{Type: "tB", Class: DefaultClass, Id: "shared.a"}].EffectiveDrift)

	{
		coord := ResourceCoordinate{Type: "tB", Class: DefaultClass, Id: "shared.a"}
		n := g2.Nodes[coord]
		n.NextDriftResolution = DriftResolutionAcceptModuleChange
		g2.Nodes[coord] = n
	}

	g3, err := SeedAndExpandAll[*FakeModuleConfiguration](t.Context(), Manifest{
		SharedResources: map[string]ManifestResource{
			"a": {Type: "tA"},
		},
	}, *index, &g2)
	require.NoError(t, err)
	assert.Equal(t, DriftNone, g3.Nodes[ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "shared.a"}].DetectedDrift)
	assert.Equal(t, DriftModuleChange, g3.Nodes[ResourceCoordinate{Type: "tB", Class: DefaultClass, Id: "shared.a"}].DetectedDrift)
	assert.Equal(t, DriftModuleChange, g3.Nodes[ResourceCoordinate{Type: "tB", Class: DefaultClass, Id: "shared.a"}].EffectiveDrift)
}

func TestDrift_drift_missing_rule(t *testing.T) {
	index := NewModuleDefinitionIndex[*FakeModuleConfiguration]([]ModuleDefinition[*FakeModuleConfiguration]{
		{ResourceType: "tA", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Id:           1,
			Dependencies: map[string]ManifestResource{},
		}},
	})
	g, err := SeedAndExpandAll[*FakeModuleConfiguration](t.Context(), Manifest{
		SharedResources: map[string]ManifestResource{
			"a": {Type: "tA"},
		},
	}, *index, nil)
	require.NoError(t, err)

	index = NewModuleDefinitionIndex[*FakeModuleConfiguration]([]ModuleDefinition[*FakeModuleConfiguration]{})
	g2, err := SeedAndExpandAll[*FakeModuleConfiguration](t.Context(), Manifest{
		SharedResources: map[string]ManifestResource{
			"a": {Type: "tA"},
		},
	}, *index, &g)
	require.NoError(t, err)
	assert.Equal(t, DriftModuleChange, g2.Nodes[ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "shared.a"}].DetectedDrift)

	{
		coord := ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "shared.a"}
		n := g2.Nodes[coord]
		n.NextDriftResolution = DriftResolutionAcceptModuleChange
		g2.Nodes[coord] = n
	}

	g3, err := SeedAndExpandAll[*FakeModuleConfiguration](t.Context(), Manifest{
		SharedResources: map[string]ManifestResource{
			"a": {Type: "tA"},
		},
	}, *index, &g2)
	require.EqualError(t, err, `graph contains 1 errors:
	type=tA,class=default,id=shared.a: no module definition matches this resource
`)
	assert.NotNil(t, g3)
}

// TestSelectorHypothesis demonstrates a hypothetical selector
func TestSelectorHypothesis(t *testing.T) {
	index := NewModuleDefinitionIndex[*FakeModuleConfiguration]([]ModuleDefinition[*FakeModuleConfiguration]{
		{ResourceType: "bucket", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{}},
		{ResourceType: "bucket-policy", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{}},
		{ResourceType: "identity", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{}},
	})
	g, err := SeedAndExpandAll[*FakeModuleConfiguration](t.Context(), Manifest{
		Workloads: map[string]ManifestWorkload{
			"one": {Resources: map[string]ManifestResource{
				"i": {Type: "identity"},
				"b": {Type: "bucket", Id: OptionalStringOf("store")},
				"p": {Type: "bucket-policy", Id: OptionalStringOf("store")},
			}},
			"two": {Resources: map[string]ManifestResource{
				"i": {Type: "identity"},
				"b": {Type: "bucket", Id: OptionalStringOf("store")},
				"p": {Type: "bucket-policy", Id: OptionalStringOf("store")},
			}},
		},
	}, *index, nil)
	require.NoError(t, err)

	matrix := g.BuildAdjacencyMatrix().FillDistanceMatrix()

	FilterResourceCoordinates := func(in iter.Seq2[ResourceCoordinate, int], filter func(coordinate ResourceCoordinate) bool) iter.Seq[ResourceCoordinate] {
		return func(yield func(ResourceCoordinate) bool) {
			for rc := range in {
				if filter(rc) {
					if !yield(rc) {
						return
					}
				}
			}
		}
	}

	FilterResourceCoordinatesForTarget := func(in iter.Seq2[ResourceCoordinate, int], resType string, resClass string, resId OptionalString) iter.Seq[ResourceCoordinate] {
		return FilterResourceCoordinates(in, func(coordinate ResourceCoordinate) bool {
			return coordinate.Type == resType && resClass == coordinate.Class &&
				(!resId.IsSet() || coordinate.Id == resId.MustValue())
		})
	}

	// start with the policy
	policy := ResourceCoordinate{Type: "bucket-policy", Class: DefaultClass, Id: "store"}
	// find all the workloads that depend on it
	for w := range FilterResourceCoordinatesForTarget(
		matrix.FindAncestorsOf(policy),
		"workload", DefaultClass, OptionalString{},
	) {
		// find all the identities that depend on the workload
		for i := range FilterResourceCoordinatesForTarget(
			matrix.FindDescendentsOf(w),
			"identity", DefaultClass, OptionalString{},
		) {
			// add a dependent edge
			require.NoError(t, g.AddAnonymousDependency(policy, i))
		}
	}

	assert.Equal(t, map[string]ResourceCoordinate{
		"zBStrHuiqpgprxguRPOGwEqJFBR8rJojEmCXTkEkejM": {Type: "identity", Class: DefaultClass, Id: "workloads.one.i"},
		"1gWWHBqa/DFzRherJX4Sb69ZRwZSx1zMHUVaVuLlu8w": {Type: "identity", Class: DefaultClass, Id: "workloads.two.i"},
	}, g.Edges[policy])

	// there should be no cycle
	assert.Empty(t, g.LabelCycle())
}

func TestCoProvisionCopyDependentsOnCurrent(t *testing.T) {
	index := NewModuleDefinitionIndex[*FakeModuleConfiguration]([]ModuleDefinition[*FakeModuleConfiguration]{
		{ResourceType: "y", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Dependents: []ManifestCoProvision{
				{
					ManifestResource:        ManifestResource{Type: "c", Id: OptionalStringOf("common")},
					IsDependentOnCurrent:    true,
					CopyDependentsOnCurrent: true,
				},
			},
		}},
		{ResourceType: "x", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Dependencies: map[string]ManifestResource{
				"y": {Type: "y"},
			},
		}},
		{ResourceType: "c", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{}},
	})
	g, err := SeedAndExpandAll[*FakeModuleConfiguration](t.Context(), Manifest{
		Workloads: map[string]ManifestWorkload{
			"one": {Resources: map[string]ManifestResource{
				"a": {Type: "y"},
			}},
			"two": {Resources: map[string]ManifestResource{
				"a": {Type: "y"},
			}},
		},
	}, *index, nil)
	require.NoError(t, err)
	assert.Equal(t, map[ResourceCoordinate]map[string]ResourceCoordinate{
		ResourceCoordinate{Type: "workload", Class: DefaultClass, Id: "one"}: {
			"a": {Type: "y", Class: DefaultClass, Id: "workloads.one.a"},
			"rOwvUd8Ysad3LS+Jrlidxyi/VSQa11kBnBQ1+aZEnxw": {Type: "c", Class: DefaultClass, Id: "common"},
		},
		ResourceCoordinate{Type: "workload", Class: DefaultClass, Id: "two"}: {
			"a": {Type: "y", Class: DefaultClass, Id: "workloads.two.a"},
			"rOwvUd8Ysad3LS+Jrlidxyi/VSQa11kBnBQ1+aZEnxw": {Type: "c", Class: DefaultClass, Id: "common"},
		},
		ResourceCoordinate{Type: "c", Class: DefaultClass, Id: "common"}: {
			"Ct5HPH03rB5dHWLFetjO3RZfmXx1c9NxdYYb/dNwl98": {Type: "y", Class: DefaultClass, Id: "workloads.one.a"},
			"5uLh7MH4TDMkprVafJsK79Y1Vrr1fhl6fUdkqKi0fpw": ResourceCoordinate{Type: "y", Class: DefaultClass, Id: "workloads.two.a"},
		},
		ResourceCoordinate{Type: "y", Class: DefaultClass, Id: "workloads.one.a"}: {},
		ResourceCoordinate{Type: "y", Class: DefaultClass, Id: "workloads.two.a"}: {},
	}, g.Edges)
}

func TestCoProvisionIsDependentOnCurrent_WithReverseDependency(t *testing.T) {
	index := NewModuleDefinitionIndex[*FakeModuleConfiguration]([]ModuleDefinition[*FakeModuleConfiguration]{
		{ResourceType: "x", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Dependencies: map[string]ManifestResource{
				"y": {Type: "y"},
			},
		}},
		{ResourceType: "y", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Dependents: []ManifestCoProvision{
				{
					ManifestResource:        ManifestResource{Type: "c"},
					IsDependentOnCurrent:    true,
					CopyDependentsOnCurrent: false,
				},
			},
		}},
		{ResourceType: "c", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Dependencies: map[string]ManifestResource{
				"y": {Type: "y"},
			},
		}},
	})
	g, err := SeedAndExpandAll[*FakeModuleConfiguration](t.Context(), Manifest{
		Workloads: map[string]ManifestWorkload{
			"one": {Resources: map[string]ManifestResource{
				"a": {Type: "y"},
			}},
		},
	}, *index, nil)
	require.NoError(t, err)
	assert.Equal(t, map[ResourceCoordinate]map[string]ResourceCoordinate{
		ResourceCoordinate{Type: "workload", Class: DefaultClass, Id: "one"}: {
			"a": {Type: "y", Class: DefaultClass, Id: "workloads.one.a"},
		},
		ResourceCoordinate{Type: "c", Class: DefaultClass, Id: "workloads.one.a"}: {
			// Despite "y" co-provisions "c" and "c" have "y" as a direct dependency, we have only one edge between
			// them, the one related to the direct dependency.
			"y": {Type: "y", Class: DefaultClass, Id: "workloads.one.a"},
		},
		ResourceCoordinate{Type: "y", Class: DefaultClass, Id: "workloads.one.a"}: {},
	}, g.Edges)
}

func TestInheritClassId(t *testing.T) {
	index := NewModuleDefinitionIndex[*FakeModuleConfiguration]([]ModuleDefinition[*FakeModuleConfiguration]{
		{ResourceType: "y", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{
			Dependencies: map[string]ManifestResource{
				"a": {
					Type:  "x",
					Class: OptionalStringOf("@"),
					Id:    OptionalStringOf("@-a"),
				},
			},
			Dependents: []ManifestCoProvision{
				{
					ManifestResource: ManifestResource{
						Type:  "x",
						Class: OptionalStringOf("@-sub"),
						Id:    OptionalStringOf("@"),
					},
				},
			},
		}},
		{ResourceType: "x", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{}},
	})
	g, err := SeedAndExpandAll[*FakeModuleConfiguration](t.Context(), Manifest{
		Workloads: map[string]ManifestWorkload{
			"one": {Resources: map[string]ManifestResource{
				"a": {Type: "y"},
			}},
		},
	}, *index, nil)
	require.NoError(t, err)
	assert.Equal(t, map[ResourceCoordinate]map[string]ResourceCoordinate{
		ResourceCoordinate{Type: "workload", Class: "default", Id: "one"}:          {"a": ResourceCoordinate{Type: "y", Class: "default", Id: "workloads.one.a"}},
		ResourceCoordinate{Type: "x", Class: "default", Id: "workloads.one.a-a"}:   {},
		ResourceCoordinate{Type: "x", Class: "default-sub", Id: "workloads.one.a"}: {},
		ResourceCoordinate{Type: "y", Class: "default", Id: "workloads.one.a"}:     {"a": {Type: "x", Class: "default", Id: "workloads.one.a-a"}},
	}, g.Edges)
}

func TestIterPlaceholders(t *testing.T) {
	tests := []struct {
		name           string
		data           interface{}
		expectedResult map[string]PlaceholderSub
		err            string
	}{
		{
			name: "resolve placeholders successfully",
			data: map[string]interface{}{
				"one": "${resources.a.outputs.x}",
				"two": ":/${resources.a.outputs.y.a}/${shared.b.outputs.y}",
				"three": map[string]interface{}{
					"four": "${resources.a.outputs.z}",
				},
				"five": []interface{}{
					"five: ${shared.b.outputs.z}",
				},
			},
			expectedResult: map[string]PlaceholderSub{
				"${resources.a.outputs.x}": &ResourcePlaceholder{
					Output:   []string{"x"},
					Alias:    "a",
					IsShared: false,
				},
				"${resources.a.outputs.y.a}": &ResourcePlaceholder{
					Output:   []string{"y", "a"},
					Alias:    "a",
					IsShared: false,
				},
				"${resources.a.outputs.z}": &ResourcePlaceholder{
					Output:   []string{"z"},
					Alias:    "a",
					IsShared: false,
				},
				"${shared.b.outputs.y}": &ResourcePlaceholder{
					Output:   []string{"y"},
					Alias:    "b",
					IsShared: true,
				},
				"${shared.b.outputs.z}": &ResourcePlaceholder{
					Output:   []string{"z"},
					Alias:    "b",
					IsShared: true,
				},
			},
		},
		{
			name: "resolve when no placeholders found",
			data: map[string]interface{}{
				"one": "value",
			},
			expectedResult: map[string]PlaceholderSub{},
		},
		{
			name: "fail with invalid placeholder",
			data: map[string]interface{}{
				"one": "${invalid.placeholder}",
			},
			err: "one: invalid placeholder 'invalid.placeholder': @0: expected one of context., self.outputs., var., resources., shared., or select.; got \"invalid\"",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var err error
			res := maps.Collect(IterPlaceholders(test.data, &err))
			if test.err == "" {
				require.NoError(t, err, test.err)
				assert.Equal(t, test.expectedResult, res)
			} else {
				assert.EqualError(t, err, test.err)
			}
		})
	}
}

func TestResolvePlaceholders(t *testing.T) {
	rc := func(t, c, id string) ResourceCoordinate {
		return ResourceCoordinate{Type: t, Class: c, Id: id}
	}
	workloadCoordinate := rc("workload", DefaultClass, "my-workload")
	explicitResourceCoordinates := rc("foo", DefaultClass, "workloads.my-workload.a")
	sharedResourceCoordinates := rc("bar", DefaultClass, "shared.b")
	g := Graph[*FakeModuleConfiguration]{
		Workloads: map[string]ResourceCoordinate{
			"my-workload": {Type: "workload", Class: DefaultClass, Id: "my-workload"},
		},
		SharedResources: map[string]ResourceCoordinate{
			"b": sharedResourceCoordinates,
		},
		Nodes: map[ResourceCoordinate]ResourceNode[*FakeModuleConfiguration]{
			workloadCoordinate: {},
			explicitResourceCoordinates: {
				ParamsDefinedBy: &workloadCoordinate,
			},
			sharedResourceCoordinates: {
				ParamsDefinedBy: &workloadCoordinate,
			},
		},
		Edges: map[ResourceCoordinate]map[string]ResourceCoordinate{
			workloadCoordinate: {
				"a": explicitResourceCoordinates,
			},
			explicitResourceCoordinates: {},
			sharedResourceCoordinates:   {},
		},
	}
	tests := []struct {
		name     string
		context  ResourceCoordinate
		rp       ResourcePlaceholder
		expected *ResourceCoordinate
		err      string
	}{
		{"resolve-shared", workloadCoordinate, ResourcePlaceholder{
			Alias:    "b",
			IsShared: true,
		}, &sharedResourceCoordinates, ""},
		{"resolve-direct", workloadCoordinate, ResourcePlaceholder{
			Alias: "a",
		}, &explicitResourceCoordinates, ""},
		{"unknown-direct", workloadCoordinate, ResourcePlaceholder{
			Alias: "u",
		}, nil, "no resource dependency with alias 'u' exists"},
		{"unknown-shared", workloadCoordinate, ResourcePlaceholder{
			Alias:    "u",
			IsShared: true,
		}, nil, "no shared resource with alias 'u' exists"},
		{"cant-use-shared", explicitResourceCoordinates, ResourcePlaceholder{
			Alias:    "b",
			IsShared: true,
		}, nil, "shared resource placeholders can only be used in a workload context"},
		{"self", explicitResourceCoordinates, ResourcePlaceholder{
			Alias:  "self",
			IsSelf: true,
		}, &explicitResourceCoordinates, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rp := test.rp
			err := g.ResolveResourcePlaceholder(test.context, &rp)
			if test.err == "" {
				require.NoError(t, err, test.err)
				assert.Equal(t, test.expected, rp.Coordinate)
			} else {
				assert.EqualError(t, err, test.err)
			}
		})
	}
}

func TestApplyExpansionFunction(t *testing.T) {
	rc := func(t, i string) ResourceCoordinate {
		return ResourceCoordinate{Type: t, Class: DefaultClass, Id: i}
	}
	m := DistanceMatrix{
		rc("a", "1"): map[ResourceCoordinate]int{
			rc("b", "1"): 1,
			rc("b", "2"): 1,
		},
		rc("b", "1"): map[ResourceCoordinate]int{
			rc("b", "3"): 1,
		},
		rc("b", "2"): map[ResourceCoordinate]int{
			rc("b", "3"): 1,
		},
		rc("b", "3"): map[ResourceCoordinate]int{},
	}
	m = m.FillDistanceMatrix()
	assert.Empty(t, applyExpansionFunction([]ResourceCoordinate{}, m.FindDescendentsOf, SelectorFilter{}, 1))
	assert.Empty(t, applyExpansionFunction([]ResourceCoordinate{}, m.FindAncestorsOf, SelectorFilter{}, 1))
	assert.Equal(t, []ResourceCoordinate{rc("b", "1"), rc("b", "2")}, applyExpansionFunction([]ResourceCoordinate{rc("a", "1")}, m.FindDescendentsOf, SelectorFilter{Type: "b"}, 1))
	assert.Equal(t, []ResourceCoordinate{rc("b", "1"), rc("b", "2"), rc("b", "3")}, applyExpansionFunction([]ResourceCoordinate{rc("a", "1")}, m.FindDescendentsOf, SelectorFilter{Type: "b"}, 2))
	assert.Equal(t, []ResourceCoordinate{rc("b", "1"), rc("b", "2")}, applyExpansionFunction([]ResourceCoordinate{rc("b", "3")}, m.FindAncestorsOf, SelectorFilter{Type: "b"}, 1))
	assert.Equal(t, []ResourceCoordinate{rc("a", "1")}, applyExpansionFunction([]ResourceCoordinate{rc("b", "3")}, m.FindAncestorsOf, SelectorFilter{Type: "a"}, 10))
}

func TestAddAnonymousDependency_PopulateEdgesLookup(t *testing.T) {
	// Create a graph with nodes and edges but without edgesLookup (simulating a restored graph)
	from := ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "a"}
	to := ResourceCoordinate{Type: "tB", Class: DefaultClass, Id: "b"}
	existing := ResourceCoordinate{Type: "tC", Class: DefaultClass, Id: "c"}

	g := &Graph[*FakeModuleConfiguration]{
		Nodes: map[ResourceCoordinate]ResourceNode[*FakeModuleConfiguration]{
			from:     {},
			to:       {},
			existing: {},
		},
		Edges: map[ResourceCoordinate]map[string]ResourceCoordinate{
			from: {
				"existing_alias": existing,
			},
		},
		// edgesLookup is intentionally nil
	}

	err := g.AddAnonymousDependency(from, to)
	require.NoError(t, err)

	// Verify the edge was added
	edges := g.Edges[from]
	assert.Contains(t, edges, DefaultAnonymousEdgeAliasFunc(from, to))
	assert.Equal(t, to, edges[DefaultAnonymousEdgeAliasFunc(from, to)])

	// Verify edgesLookup was populated and contains the existing and the new edge
	require.NotNil(t, g.edgesLookup)
	assert.Equal(t, "existing_alias", g.edgesLookup[from][existing])
	assert.Equal(t, DefaultAnonymousEdgeAliasFunc(from, to), g.edgesLookup[from][to])
}
