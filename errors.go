package orchestrator_graph

import (
	"fmt"
	"strings"

	"github.com/stellwerk-labs/platform-orchestrator-graph/internal"
)

type ErrParamsMismatch struct{}

func (e *ErrParamsMismatch) Error() string {
	return "referenced multiple times in the graph with different resource parameters"
}

type ErrNoRuleMatch struct{}

func (e *ErrNoRuleMatch) Error() string {
	return "no module definition matches this resource"
}

type ErrCycle struct{}

func (e *ErrCycle) Error() string {
	return "cycle detected"
}

type ErrGraph struct {
	ByNode map[ResourceCoordinate]error
}

func (e *ErrGraph) Error() string {
	sb := new(strings.Builder)
	_, _ = fmt.Fprintf(sb, "graph contains %d errors:\n", len(e.ByNode))
	for coordinate, err := range internal.SortedEntriesFunc(e.ByNode, CompareResourceCoordinate) {
		_, _ = fmt.Fprintf(sb, "\t%s: %s\n", coordinate, err)
	}
	return sb.String()
}

type ErrInvalidPlaceholder struct {
	message string
}

func (e *ErrInvalidPlaceholder) Error() string {
	return e.message
}

func NewErrInvalidPlaceholder(message string) *ErrInvalidPlaceholder {
	return &ErrInvalidPlaceholder{message: message}
}
