package orchestrator_graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompareResourceCoordinate(t *testing.T) {
	assert.Equal(t, 0, CompareResourceCoordinate(ResourceCoordinate{}, ResourceCoordinate{}))

	assert.Equal(t, -1, CompareResourceCoordinate(ResourceCoordinate{Type: "t", Class: "c", Id: "i"}, ResourceCoordinate{Type: "u", Class: "c", Id: "i"}))
	assert.Equal(t, 1, CompareResourceCoordinate(ResourceCoordinate{Type: "u", Class: "c", Id: "i"}, ResourceCoordinate{Type: "t", Class: "c", Id: "i"}))

	assert.Equal(t, -1, CompareResourceCoordinate(ResourceCoordinate{Type: "t", Class: "c", Id: "i"}, ResourceCoordinate{Type: "t", Class: "d", Id: "i"}))
	assert.Equal(t, 1, CompareResourceCoordinate(ResourceCoordinate{Type: "t", Class: "d", Id: "i"}, ResourceCoordinate{Type: "t", Class: "c", Id: "i"}))

	assert.Equal(t, -1, CompareResourceCoordinate(ResourceCoordinate{Type: "t", Class: "c", Id: "i"}, ResourceCoordinate{Type: "t", Class: "c", Id: "j"}))
	assert.Equal(t, 1, CompareResourceCoordinate(ResourceCoordinate{Type: "t", Class: "c", Id: "j"}, ResourceCoordinate{Type: "t", Class: "c", Id: "i"}))

	assert.Equal(t, 0, CompareResourceCoordinate(ResourceCoordinate{Type: "t", Class: "c", Id: "i"}, ResourceCoordinate{Type: "t", Class: "c", Id: "i"}))
}
