package expressions

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	parsererror "github.com/stellwerk-labs/platform-orchestrator-graph/internal/expressions/generated/errors"
	"github.com/stellwerk-labs/platform-orchestrator-graph/internal/expressions/generated/lexer"
	"github.com/stellwerk-labs/platform-orchestrator-graph/internal/expressions/generated/parser"
)

//go:generate go tool gocc -o ./generated -p github.com/stellwerk-labs/platform-orchestrator-graph/internal/expressions/generated -a ./expressions.bnf

func Parse(raw string) (interface{}, error) {
	lex := lexer.NewLexer([]byte(raw))
	parse := parser.NewParser()
	out, err := parse.Parse(lex)
	if err != nil {
		if e := new(parsererror.Error); errors.As(err, &e) {
			tokens := make([]string, len(e.ExpectedTokens))
			for idx, token := range e.ExpectedTokens {
				if !unicode.IsLetter(rune(token[0])) {
					token = strconv.Quote(token)
				}
				tokens[idx] = token
			}
			got := strings.ReplaceAll(parsererror.DescribeToken(e.ErrorToken), "end-of-file", "end of line")
			expected := strings.ReplaceAll(parsererror.DescribeExpected(tokens), "␚", "end of line")
			return nil, fmt.Errorf("@%d: %s; got %s", e.ErrorToken.Column-1, expected, got)
		}
		return nil, err
	}
	return out, nil
}
