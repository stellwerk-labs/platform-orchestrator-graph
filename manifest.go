package orchestrator_graph

type Manifest struct {
	Workloads       map[string]ManifestWorkload
	SharedResources map[string]ManifestResource
}

type ManifestWorkload struct {
	Resources map[string]ManifestResource
	Outputs   map[string]string
}

type ManifestResource struct {
	Type   string
	Class  OptionalString
	Id     OptionalString
	Params map[string]interface{}
}
