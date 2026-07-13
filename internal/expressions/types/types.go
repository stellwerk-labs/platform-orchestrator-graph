package types

import (
	"strings"

	"github.com/stellwerk-labs/platform-orchestrator-graph/internal/expressions/generated/token"
)

type ContextExpr struct {
	Key string
}

type VarExpr struct {
	Key string
}

type ResourceExpression struct {
	IsShared   bool
	Alias      string
	OutputKeys []string
}

type FuncCallExpression struct {
	Func string
	Args []interface{}
}

type SelectExpression struct {
	Functions  []FuncCallExpression
	OutputKeys []string
}

type SelfExpression struct {
	OutputKeys []string
}

func mustTokenString(raw interface{}) string {
	return string(raw.(*token.Token).Lit)
}

func NewContextExpr(key interface{}) (*ContextExpr, error) {
	return &ContextExpr{Key: mustTokenString(key)}, nil
}

func NewVarExpr(key interface{}) (*VarExpr, error) {
	return &VarExpr{Key: mustTokenString(key)}, nil
}

func NewResourceExpr(alias interface{}, key interface{}, traversal interface{}) (*ResourceExpression, error) {
	concat := mustTokenString(key)
	if traversal != nil {
		concat += mustTokenString(traversal)
	}
	return &ResourceExpression{
		Alias:      mustTokenString(alias),
		OutputKeys: strings.Split(concat, "."),
	}, nil
}

func NewSharedResourceExpr(alias interface{}, key interface{}, traversal interface{}) (*ResourceExpression, error) {
	r, err := NewResourceExpr(alias, key, traversal)
	if err != nil {
		return nil, err
	}
	r.IsShared = true
	return r, nil
}

func NewFuncCall(name interface{}, arg interface{}) (*FuncCallExpression, error) {
	funcName := strings.TrimSuffix(mustTokenString(name), "(")
	argStr := mustTokenString(arg)
	argStr = argStr[1 : len(argStr)-1]
	return &FuncCallExpression{Func: funcName, Args: []interface{}{argStr}}, nil
}

func NewSelectExpression(funcChain interface{}, key interface{}, traversal interface{}) (*SelectExpression, error) {
	concat := mustTokenString(key)
	if traversal != nil {
		concat += mustTokenString(traversal)
	}
	functions := make([]FuncCallExpression, 0, 1)
	switch typed := funcChain.(type) {
	case *FuncCallExpression:
		functions = append(functions, *typed)
	default:
		functions = append(functions, typed.([]FuncCallExpression)...)
	}
	return &SelectExpression{
		Functions:  functions,
		OutputKeys: strings.Split(concat, "."),
	}, nil
}

func CollectFuncCalls(more interface{}, f interface{}) ([]FuncCallExpression, error) {
	if more == nil {
		return []FuncCallExpression{*f.(*FuncCallExpression)}, nil
	}
	return append(more.([]FuncCallExpression), *f.(*FuncCallExpression)), nil
}

func NewSelfExpr(key interface{}, traversal interface{}) (*SelfExpression, error) {
	concat := mustTokenString(key)
	if traversal != nil {
		concat += mustTokenString(traversal)
	}
	return &SelfExpression{
		OutputKeys: strings.Split(concat, "."),
	}, nil
}
