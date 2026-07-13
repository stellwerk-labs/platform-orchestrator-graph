package orchestrator_graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePlaceholder(t *testing.T) {
	for pc, tc := range map[string]interface{}{
		"banana":                             "the string is not a placeholder: banana",
		"${context}":                         "invalid placeholder 'context': @0: expected one of context., self.outputs., var., resources., shared., or select.; got \"context\"",
		"${context.foo}":                     &ContextPlaceholder{Key: "foo"},
		"${context.fizz.bar}":                "invalid placeholder 'context.fizz.bar': @12: expected end of line; got \".bar\"",
		"${resources.foo}":                   "invalid placeholder 'resources.foo': @13: expected \".outputs.\"; got end of line",
		"${resources.foo.bar}":               "invalid placeholder 'resources.foo.bar': @13: expected \".outputs.\"; got \".bar\"",
		"${resources.foo.outputs}":           "invalid placeholder 'resources.foo.outputs': @13: expected \".outputs.\"; got \".outputs\"",
		"${resources.foo.outputs.fizz.buzz}": &ResourcePlaceholder{Alias: "foo", Output: []string{"fizz", "buzz"}},
		"${shared.foo}":                      "invalid placeholder 'shared.foo': @10: expected \".outputs.\"; got end of line",
		"${shared.foo.bar}":                  "invalid placeholder 'shared.foo.bar': @10: expected \".outputs.\"; got \".bar\"",
		"${shared.foo.outputs}":              "invalid placeholder 'shared.foo.outputs': @10: expected \".outputs.\"; got \".outputs\"",
		"${shared.foo.outputs.fizz.buzz}":    &ResourcePlaceholder{Alias: "foo", Output: []string{"fizz", "buzz"}, IsShared: true},
		"${var.foo}":                         &TfVarPlaceholder{Key: "foo"},
		"${self.outputs.fizz}":               &ResourcePlaceholder{Alias: "self", IsSelf: true, Output: []string{"fizz"}},
		"${self.outputs.fizz.buzz}":          &ResourcePlaceholder{Alias: "self", IsSelf: true, Output: []string{"fizz", "buzz"}},
	} {
		t.Run(pc, func(t *testing.T) {
			p, err := ParsePlaceholder(pc)
			switch typed := tc.(type) {
			case string:
				require.EqualError(t, err, typed)
			case *ResourcePlaceholder:
				require.NoError(t, err)
				assert.Equal(t, tc, p)
			case *ContextPlaceholder:
				require.NoError(t, err)
				assert.Equal(t, tc, p)
			}
		})
	}
}

func TestSelectorFilter_parse(t *testing.T) {
	assert.Equal(t, SelectorFilter{}, ParseSelectorFilter(""))
	assert.Equal(t, SelectorFilter{Type: "dns"}, ParseSelectorFilter("dns"))
	assert.Equal(t, SelectorFilter{Type: "dns", Class: OptionalStringOf("default")}, ParseSelectorFilter("dns.default"))
	assert.Equal(t, SelectorFilter{Type: "dns", Id: OptionalStringOf("thing")}, ParseSelectorFilter("dns#thing"))
	assert.Equal(t, SelectorFilter{Type: "dns", Class: OptionalStringOf("@"), Id: OptionalStringOf("@")}, ParseSelectorFilter("dns.@#@"))
	assert.Equal(t, SelectorFilter{Type: "pathological", Class: OptionalStringOf("many.classes..what"), Id: OptionalStringOf("is.this#an.id")}, ParseSelectorFilter("pathological.many.classes..what#is.this#an.id"))
}

func TestSelectorFilter_matches(t *testing.T) {
	rca := ResourceCoordinate{Type: "dns", Class: "something", Id: "an-id"}
	rcb := ResourceCoordinate{Type: "thing", Class: "default", Id: "another-id"}

	assert.True(t, ParseSelectorFilter("dns").Matches(rca, rcb))
	assert.False(t, ParseSelectorFilter("thing").Matches(rca, rcb))
	assert.True(t, ParseSelectorFilter("dns.something").Matches(rca, rcb))
	assert.False(t, ParseSelectorFilter("dns.default").Matches(rca, rcb))
	assert.False(t, ParseSelectorFilter("dns.@").Matches(rca, rcb))
	assert.True(t, ParseSelectorFilter("dns.@").Matches(rca, rca))
	assert.True(t, ParseSelectorFilter("dns#an-id").Matches(rca, rcb))
	assert.False(t, ParseSelectorFilter("dns#another-id").Matches(rca, rcb))
	assert.False(t, ParseSelectorFilter("dns#@").Matches(rca, rcb))
	assert.True(t, ParseSelectorFilter("dns#@").Matches(rca, rca))
	assert.True(t, ParseSelectorFilter("dns.@#@").Matches(rca, rca))
}
