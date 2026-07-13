package orchestrator_graph

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptionalString(t *testing.T) {
	var o OptionalString
	assert.False(t, o.IsSet())
	assert.Panics(t, func() {
		_ = o.MustValue()
	})
	assert.Nil(t, o.Ref())
	assert.Equal(t, "<not set>", o.String())
	assert.False(t, o.Map(func(s string) string { return s }).IsSet())
	assert.Equal(t, "x", o.ValueOrFunc(func() string {
		return "x"
	}))
}

func TestOptionalStringOf(t *testing.T) {
	o := OptionalStringOf("x")
	assert.True(t, o.IsSet())
	assert.Equal(t, "x", o.MustValue())
	assert.NotNil(t, o.Ref())
	assert.Equal(t, OptionalStringOf("x"), o)
	assert.Equal(t, "x", o.String())
	assert.Equal(t, "X", o.Map(func(s string) string { return strings.ToUpper(s) }).MustValue())
	assert.Equal(t, "x", o.ValueOrFunc(func() string {
		return "y"
	}))
}

func TestOptionalStringOfRef(t *testing.T) {
	o := OptionalStringOfRef(nil)
	assert.False(t, o.IsSet())
	x := "x"
	o = OptionalStringOfRef(&x)
	assert.True(t, o.IsSet())
	x = "y"
	assert.Equal(t, "x", o.MustValue())
}

func TestOptionalStringOfNonEmpty(t *testing.T) {
	o := OptionalStringOfNonEmpty("")
	assert.False(t, o.IsSet())
	o = OptionalStringOfNonEmpty("x")
	assert.True(t, o.IsSet())
	assert.Equal(t, "x", o.MustValue())
}

func TestCompareOptionalString(t *testing.T) {
	assert.Equal(t, 0, CompareOptionalString(OptionalStringOf("x"), OptionalStringOf("x")))
	assert.Equal(t, 0, CompareOptionalString(OptionalString{}, OptionalString{}))

	assert.Equal(t, -1, CompareOptionalString(OptionalStringOf("x"), OptionalStringOf("y")))
	assert.Equal(t, -1, CompareOptionalString(OptionalString{}, OptionalStringOf("y")))

	assert.Equal(t, 1, CompareOptionalString(OptionalStringOf("y"), OptionalStringOf("x")))
	assert.Equal(t, 1, CompareOptionalString(OptionalStringOf("y"), OptionalString{}))
}
