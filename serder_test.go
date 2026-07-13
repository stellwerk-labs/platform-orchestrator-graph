package orchestrator_graph

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSerDer(t *testing.T) {
	g := Seed[*FakeModuleConfiguration](t.Context(), Manifest{
		SharedResources: map[string]ManifestResource{
			"a": {Type: "tA"},
		},
	})
	g.Expand(t.Context(), *NewModuleDefinitionIndex[*FakeModuleConfiguration]([]ModuleDefinition[*FakeModuleConfiguration]{
		{ResourceType: "tA", Rules: []Rule{{}}, Configuration: &FakeModuleConfiguration{Id: 1}},
	}), nil)
	if assert.True(t, g.IsExpanded()) && assert.Empty(t, g.LabelCycle()) && assert.False(t, g.HasErrors()) {
		raw, err := json.Marshal(g)
		if assert.NoError(t, err) {
			var g2 *Graph[*FakeModuleConfiguration]
			dec := json.NewDecoder(bytes.NewReader(raw))
			dec.DisallowUnknownFields()
			if assert.NoError(t, dec.Decode(&g2)) {
				assert.Equal(t, g.Nodes, g2.Nodes)
				assert.Equal(t, g.Edges, g2.Edges)
				assert.Equal(t, g.Workloads, g2.Workloads)
				assert.Equal(t, g.SharedResources, g2.SharedResources)
				assert.Equal(t, g.LeafResources, g2.LeafResources)
				assert.Equal(t, g.HasErrors(), g2.HasErrors())
			}
		}
	}
}
