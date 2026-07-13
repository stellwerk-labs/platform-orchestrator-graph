package orchestrator_graph

import "strings"

// OptionalString is a static non-pointer implementation of a nullable string. Use IsSet to check the value, and MustValue
// to retrieve tha value. Various utility constructors and methods exist for making things easy
type OptionalString struct {
	value string
	isSet bool
}

// OptionalStringOf returns an OptionalString with the value set.
func OptionalStringOf(s string) OptionalString {
	return OptionalString{value: s, isSet: true}
}

// OptionalStringOfRef returns an OptionalString with the value set if it is not nil.
func OptionalStringOfRef(s *string) OptionalString {
	if s == nil {
		return OptionalString{}
	}
	return OptionalString{value: *s, isSet: true}
}

// OptionalStringOfNonEmpty is similar to OptionalStringOf but treats an empty string as not set.
func OptionalStringOfNonEmpty(s string) OptionalString {
	if s == "" {
		return OptionalString{}
	}
	return OptionalStringOf(s)
}

func (o OptionalString) MustValue() string {
	if !o.isSet {
		panic("optional string value is not set")
	}
	return o.value
}

func (o OptionalString) IsSet() bool {
	return o.isSet
}

func (o OptionalString) ValueOr(defaultValue string) string {
	if !o.isSet {
		return defaultValue
	}
	return o.value
}

func (o OptionalString) ValueOrFunc(defaultValue func() string) string {
	if !o.isSet {
		return defaultValue()
	}
	return o.value
}

func (o OptionalString) Ref() *string {
	if !o.isSet {
		return nil
	}
	return &o.value
}

func (o OptionalString) String() string {
	return o.ValueOr("<not set>")
}

func (o OptionalString) Map(f func(s string) string) OptionalString {
	if !o.isSet {
		return o
	}
	return OptionalStringOf(f(o.value))
}

func CompareOptionalString(a, b OptionalString) int {
	if a.isSet {
		if b.isSet {
			return strings.Compare(a.value, b.value)
		}
		return 1
	} else if b.IsSet() {
		return -1
	}
	return 0
}
