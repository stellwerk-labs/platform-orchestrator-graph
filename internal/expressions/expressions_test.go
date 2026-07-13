package expressions

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stellwerk-labs/platform-orchestrator-graph/internal/expressions/types"
)

func Test_ok(t *testing.T) {
	for _, tt := range []struct {
		raw      string
		expected interface{}
	}{
		{raw: "var.foo", expected: &types.VarExpr{Key: "foo"}},
		{raw: "var.var", expected: &types.VarExpr{Key: "var"}},
		{raw: "context.foo", expected: &types.ContextExpr{Key: "foo"}},
		{raw: "context.context", expected: &types.ContextExpr{Key: "context"}},
		{raw: "resources.foo.outputs.resources", expected: &types.ResourceExpression{Alias: "foo", OutputKeys: []string{"resources"}}},
		{raw: "resources.foo.outputs.x.y", expected: &types.ResourceExpression{Alias: "foo", OutputKeys: []string{"x", "y"}}},
		{raw: "shared.foo.outputs.x", expected: &types.ResourceExpression{Alias: "foo", OutputKeys: []string{"x"}, IsShared: true}},
		{raw: "shared.foo.outputs.x.y", expected: &types.ResourceExpression{Alias: "foo", OutputKeys: []string{"x", "y"}, IsShared: true}},
		{raw: "select.dependencies('filter').outputs.x.y", expected: &types.SelectExpression{
			Functions:  []types.FuncCallExpression{{Func: "dependencies", Args: []interface{}{"filter"}}},
			OutputKeys: []string{"x", "y"},
		}},
		{raw: "select.dependencies('filter').consumers('a').outputs.x.y", expected: &types.SelectExpression{
			Functions:  []types.FuncCallExpression{{Func: "dependencies", Args: []interface{}{"filter"}}, {Func: "consumers", Args: []interface{}{"a"}}},
			OutputKeys: []string{"x", "y"},
		}},
		{raw: "select.dependencies('access-key#@').consumers('workload').dependencies('s3.read-only').outputs.bucket", expected: &types.SelectExpression{
			Functions:  []types.FuncCallExpression{{Func: "dependencies", Args: []interface{}{"access-key#@"}}, {Func: "consumers", Args: []interface{}{"workload"}}, {Func: "dependencies", Args: []interface{}{"s3.read-only"}}},
			OutputKeys: []string{"bucket"},
		}},
	} {
		t.Run(tt.raw, func(t *testing.T) {
			o, err := Parse(tt.raw)
			require.NoError(t, err)
			require.Equal(t, tt.expected, o)
		})
	}
}

func Test_fail(t *testing.T) {
	for _, tt := range []struct {
		raw    string
		errMsg string
	}{
		{raw: "", errMsg: "@0: expected one of context., self.outputs., var., resources., shared., or select."},
		{raw: "things", errMsg: "@0: expected one of context., self.outputs., var., resources., shared., or select."},
		{raw: "var.", errMsg: "@4: expected key"},
		{raw: "context.", errMsg: "@8: expected key"},
		{raw: "resources..outputs", errMsg: "@10: expected key"},
		{raw: "shared.fii.outputs", errMsg: "@10: expected \".outputs.\"; got \".outputs\""},
		{raw: "select.unknownFunction('arg').outputs", errMsg: "@7: expected either dependencies( or consumers(; got \"unknownFunction\""},
		{raw: "select.dependencies().outputs", errMsg: "@20: expected filter; got \").\""},
		{raw: "select..outputs", errMsg: "@7: expected either dependencies( or consumers(; got \".outputs\""},
		{raw: "dependencies('filter')", errMsg: "@0: expected one of context., self.outputs., var., resources., shared., or select.; got \"dependencies(\""},
	} {
		t.Run(tt.raw, func(t *testing.T) {
			_, err := Parse(tt.raw)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.errMsg)
		})
	}
}
