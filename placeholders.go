package orchestrator_graph

import (
	"fmt"
	"iter"
	"regexp"
	"strings"

	"github.com/stellwerk-labs/platform-orchestrator-graph/internal/expressions"
	"github.com/stellwerk-labs/platform-orchestrator-graph/internal/expressions/types"
)

var RePlaceholder = regexp.MustCompile(`\${[^{}]+}`)

type PlaceholderType string

const (
	PlaceholderTypeResource PlaceholderType = "resource"
	PlaceholderTypeContext  PlaceholderType = "context"
	PlaceholderTypeTfVar    PlaceholderType = "var"
	PlaceholderTypeSelector PlaceholderType = "select"
	PlaceholderTypeSelf     PlaceholderType = "self"
)

type ResourcePlaceholder struct {
	// Coordinate is the matched coordinate once we've resolved the placeholder
	Coordinate *ResourceCoordinate
	// Alias is the local alias within the context of the current node
	Alias string
	// IsShared holds whether this is from the shared aliases rather than the dependency aliases
	IsShared bool
	// IsSelf holds whether this is the self placeholder
	IsSelf bool
	// Output captures the output key traversal they want to pull out of the target resource
	Output []string
}

func (r ResourcePlaceholder) Type() PlaceholderType {
	return PlaceholderTypeResource
}

type ContextPlaceholder struct {
	// Key is the context key from the context map
	Key string
}

func (c ContextPlaceholder) Type() PlaceholderType {
	return PlaceholderTypeContext
}

type TfVarPlaceholder struct {
	Key string
}

func (e TfVarPlaceholder) Type() PlaceholderType {
	return PlaceholderTypeTfVar
}

type SelectorFunction struct {
	Func string
	Args []interface{}
}

type SelectorPlaceholder struct {
	// MatchedCoordinates is the set of matches nodes once we've resolved the placeholder
	MatchedCoordinates []ResourceCoordinate
	// Functions is the list of dependency / consumer function calls
	Functions []string
	// Args is the 1:1 map of args to each function call
	Args []string
	// Output is the output key traversal to pull out of the target resources
	Output []string
}

func (e SelectorPlaceholder) Type() PlaceholderType {
	return PlaceholderTypeSelector
}

type PlaceholderSub interface {
	Type() PlaceholderType
}

func ParsePlaceholder(placeholder string) (PlaceholderSub, error) {
	if !strings.HasPrefix(placeholder, "${") || !strings.HasSuffix(placeholder, "}") {
		return nil, NewErrInvalidPlaceholder("the string is not a placeholder: " + placeholder)
	}
	unwrapped := placeholder[2 : len(placeholder)-1]
	out, err := expressions.Parse(unwrapped)
	if err != nil {
		return nil, NewErrInvalidPlaceholder(fmt.Sprintf("invalid placeholder '%s': %s", unwrapped, err))
	}
	switch out := out.(type) {
	case *types.ContextExpr:
		return &ContextPlaceholder{Key: out.Key}, nil
	case *types.VarExpr:
		return &TfVarPlaceholder{Key: out.Key}, nil
	case *types.ResourceExpression:
		return &ResourcePlaceholder{Alias: out.Alias, Output: out.OutputKeys, IsShared: out.IsShared}, nil
	case *types.SelfExpression:
		return &ResourcePlaceholder{Alias: string(PlaceholderTypeSelf), Output: out.OutputKeys, IsSelf: true}, nil
	case *types.SelectExpression:
		functions, args := make([]string, 0, len(out.Functions)), make([]string, 0, len(out.Functions))
		for _, f := range out.Functions {
			functions = append(functions, f.Func)
			args = append(args, f.Args[0].(string))
		}
		return &SelectorPlaceholder{Functions: functions, Args: args, Output: out.OutputKeys}, nil
	default:
		panic("unreachable")
	}
}

// IterPlaceholders streams over the data type and returns any observed placeholders and a parsed object for them.
// It will automatically deduplicate placeholders so they only need to be evaluated once.
func IterPlaceholders(data interface{}, finalErr *error) iter.Seq2[string, PlaceholderSub] {
	return func(yield func(string, PlaceholderSub) bool) {
		seen := make(map[string]bool)
		switch typedData := data.(type) {
		case map[string]interface{}:
			for k, v := range typedData {
				for placeholder, sub := range IterPlaceholders(v, finalErr) {
					if !yield(placeholder, sub) {
						return
					}
				}
				if *finalErr != nil {
					*finalErr = fmt.Errorf("%v: %w", k, *finalErr)
					return
				}
			}
		case []interface{}:
			for i, v := range typedData {
				for placeholder, sub := range IterPlaceholders(v, finalErr) {
					if !yield(placeholder, sub) {
						return
					}
				}
				if *finalErr != nil {
					*finalErr = fmt.Errorf("[%d]: %w", i, *finalErr)
					return
				}
			}
		case string:
			matches := RePlaceholder.FindAllStringSubmatch(typedData, -1)
			for _, match := range matches {
				placeholder := match[0]
				if seen[placeholder] {
					continue
				}
				seen[placeholder] = true
				sub, err := ParsePlaceholder(placeholder)
				if err != nil {
					*finalErr = err
					return
				}
				if !yield(placeholder, sub) {
					return
				}
			}
		}
	}
}

type SelectorFilter struct {
	Type  string
	Class OptionalString
	Id    OptionalString
}

func ParseSelectorFilter(raw string) SelectorFilter {
	var rc SelectorFilter
	parts := strings.SplitN(raw, "#", 2)
	if len(parts) == 2 {
		rc.Id = OptionalStringOf(parts[1])
	}
	parts = strings.SplitN(parts[0], ".", 2)
	if len(parts) == 2 {
		rc.Class = OptionalStringOf(parts[1])
	}
	rc.Type = parts[0]
	return rc
}

func (f SelectorFilter) Matches(rc ResourceCoordinate, context ResourceCoordinate) bool {
	if f.Type != rc.Type {
		return false
	}
	if f.Class.IsSet() {
		compareClass := strings.ReplaceAll(f.Class.MustValue(), InheritClassOrIdSymbol, context.Class)
		if rc.Class != compareClass {
			return false
		}
	}
	if f.Id.IsSet() {
		compareId := strings.ReplaceAll(f.Id.MustValue(), InheritClassOrIdSymbol, context.Id)
		if rc.Id != compareId {
			return false
		}
	}
	return true
}
