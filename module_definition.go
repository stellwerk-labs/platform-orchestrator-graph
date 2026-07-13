package orchestrator_graph

import "context"

type DriftType string

const (
	// DriftNone means the nodes are using exactly the same module configuration
	DriftNone DriftType = ""
	// DriftConfigurationChange means the nodes are using the same module, but they differ in version, params,
	// or configuration. An upgrade is likely to succeed.
	DriftConfigurationChange DriftType = "config_change"
	// DriftModuleChange means the nodes are using different modules entirely and a replacement will likely be needed.
	DriftModuleChange DriftType = "module_change"
	// DriftConfigurationChangeSkipped occurs when a configuration change is skipped due to a drift resolution strategy.
	DriftConfigurationChangeSkipped DriftType = "config_change_skipped"
	// DriftModuleChangeSkipped occurs when a module change is skipped due to a drift resolution strategy.
	DriftModuleChangeSkipped DriftType = "module_change_skipped"
)

type DriftResolution string

const (
	// DriftResolutionAcceptModuleChange allows the node to accept module and configuration changes
	DriftResolutionAcceptModuleChange DriftResolution = "accept_module_change"
	// DriftResolutionAcceptConfigChange allows the node to accept configuration changes
	DriftResolutionAcceptConfigChange DriftResolution = "accept_config_change"
)

type ModuleConfiguration interface {
	CalculateDrift(mc ModuleConfiguration) DriftType
	GetDependencies() map[string]ManifestResource
	GetCoProvisioned() []ManifestCoProvision
}

type ManifestCoProvision struct {
	ManifestResource
	// If IsDependentOnCurrent is true, then we create an edge from the co provisioned resource to the current resource
	// so that this co provisioned resource "depends" on the current resource and is only provisioned after the
	// current resource is completely provisioned. This may be needed if the co provisioned resource uses selectors
	// to pull values out of the current resource or relies on side effects from it.
	IsDependentOnCurrent bool

	// If CopyDependentsOnCurrent is true, then we also add edges from all resources that depend on the current resource
	// to the co provisioned resource. So that we enforce a sibling relationships between the current resource and the
	// co provisioned one. This ensures that the co provisioned resource is co provisioned before the parents.
	CopyDependentsOnCurrent bool
}

type ModuleDefinition[E ModuleConfiguration] struct {
	// The resource matching bits
	ResourceType string
	Rules        []Rule

	// The actual configuration of the module
	Configuration E
}

type Rule struct {
	ResourceClass OptionalString
	ResourceId    OptionalString
}

type ModuleDefinitionIndex[D ModuleConfiguration] struct {
	index map[string]map[OptionalString]map[OptionalString]ModuleDefinition[D]
}

func NewModuleDefinitionIndex[D ModuleConfiguration](defs []ModuleDefinition[D]) *ModuleDefinitionIndex[D] {
	index := make(map[string]map[OptionalString]map[OptionalString]ModuleDefinition[D])
	for _, def := range defs {
		byClass, ok := index[def.ResourceType]
		if !ok {
			byClass = make(map[OptionalString]map[OptionalString]ModuleDefinition[D])
			index[def.ResourceType] = byClass
		}

		for _, rule := range def.Rules {
			byId, ok := byClass[rule.ResourceClass]
			if !ok {
				byId = make(map[OptionalString]ModuleDefinition[D])
				byClass[rule.ResourceClass] = byId
			}
			byId[rule.ResourceId] = def
		}
	}
	return &ModuleDefinitionIndex[D]{index: index}
}

func (d *ModuleDefinitionIndex[D]) FindBest(_ context.Context, coordinate ResourceCoordinate) (ModuleDefinition[D], bool) {
	if byType, ok := d.index[coordinate.Type]; ok {
		byClass, ok := byType[OptionalStringOf(coordinate.Class)]
		if !ok {
			byClass, ok = byType[OptionalString{}]
		}
		if ok {
			byId, ok := byClass[OptionalStringOf(coordinate.Id)]
			if !ok {
				byId, ok = byClass[OptionalString{}]
			}
			if ok {
				return byId, true
			}
		}
	}
	return ModuleDefinition[D]{}, false
}

func ApplyDriftResolution(current DriftType, resolution DriftResolution) (DriftType, bool) {
	switch current {
	case DriftConfigurationChange:
		if resolution != DriftResolutionAcceptConfigChange && resolution != DriftResolutionAcceptModuleChange {
			return DriftConfigurationChangeSkipped, false
		}
	case DriftModuleChange:
		if resolution != DriftResolutionAcceptModuleChange {
			return DriftModuleChangeSkipped, false
		}
	}
	return current, true
}
