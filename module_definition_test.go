package orchestrator_graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type FakeModuleConfiguration struct {
	Id              uint64                      `json:"id"`
	Version         uint64                      `json:"version"`
	Dependencies    map[string]ManifestResource `json:"dependencies"`
	Dependents      []ManifestCoProvision       `json:"dependents"`
	DriftResolution DriftResolution             `json:"drift_resolution"`
}

func (f *FakeModuleConfiguration) CalculateDrift(mc ModuleConfiguration) DriftType {
	if f2 := mc.(*FakeModuleConfiguration); f.Id != f2.Id {
		return DriftModuleChange
	} else if f.Version != f2.Version {
		return DriftConfigurationChange
	}
	return DriftNone
}

func (f *FakeModuleConfiguration) GetDependencies() map[string]ManifestResource {
	return f.Dependencies
}

func (f *FakeModuleConfiguration) GetCoProvisioned() []ManifestCoProvision {
	return f.Dependents
}

var _ ModuleConfiguration = (*FakeModuleConfiguration)(nil)

func TestModuleDefinitionIndex(t *testing.T) {
	index := NewModuleDefinitionIndex[*FakeModuleConfiguration]([]ModuleDefinition[*FakeModuleConfiguration]{
		{"tA", []Rule{{}}, &FakeModuleConfiguration{Id: 1, Version: 0, Dependencies: nil, Dependents: nil}},
		{"tA", []Rule{{ResourceClass: OptionalStringOf("cA")}}, &FakeModuleConfiguration{Id: 2, Version: 0, Dependencies: nil, Dependents: nil}},
		{"tA", []Rule{{ResourceId: OptionalStringOf("iA")}}, &FakeModuleConfiguration{Id: 3, Version: 0, Dependencies: nil, Dependents: nil}},
	})
	if m, ok := index.FindBest(t.Context(), ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "some-id"}); assert.True(t, ok) {
		assert.Equal(t, uint64(1), m.Configuration.Id)
	}
	if m, ok := index.FindBest(t.Context(), ResourceCoordinate{Type: "tA", Class: DefaultClass, Id: "iA"}); assert.True(t, ok) {
		assert.Equal(t, uint64(3), m.Configuration.Id)
	}
	if m, ok := index.FindBest(t.Context(), ResourceCoordinate{Type: "tA", Class: "cB"}); assert.True(t, ok) {
		assert.Equal(t, uint64(1), m.Configuration.Id)
	}
	if m, ok := index.FindBest(t.Context(), ResourceCoordinate{Type: "tA", Class: "cA"}); assert.True(t, ok) {
		assert.Equal(t, uint64(2), m.Configuration.Id)
	}
	if m, ok := index.FindBest(t.Context(), ResourceCoordinate{Type: "tB", Class: DefaultClass}); assert.False(t, ok) {
		assert.Nil(t, m.Configuration)
	}
}
