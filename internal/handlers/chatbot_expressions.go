package handlers

import (
	"fmt"
	"strconv"
	"strings"
)

type expressionTokenType uint8

const (
	expressionTokenCondition expressionTokenType = iota
	expressionTokenAnd
	expressionTokenOr
	expressionTokenLeftParen
	expressionTokenRightParen
)

type expressionToken struct {
	kind  expressionTokenType
	value string
}

func needsExpressionParsing(expr string) bool {
	inQuote := byte(0)
	for i := 0; i < len(expr); i++ {
		ch := expr[i]
		if inQuote != 0 {
			if ch == inQuote {
				inQuote = 0
			}
			continue
		}

		switch ch {
		case '\'', '"':
			inQuote = ch
		case '(', ')':
			return true
		default:
			if _, isLogical := parseLogicalOperatorAt(expr, i); isLogical {
				return true
			}
		}
	}

	return false
}

func parseExpressionToRPN(expr string) ([]expressionToken, bool) {
	tokens := tokenizeExpression(expr)
	if len(tokens) == 0 {
		return nil, false
	}

	output := make([]expressionToken, 0, len(tokens))
	operators := make([]expressionToken, 0, len(tokens))

	for _, token := range tokens {
		switch token.kind {
		case expressionTokenCondition:
			output = append(output, token)
		case expressionTokenAnd, expressionTokenOr:
			for len(operators) > 0 {
				top := operators[len(operators)-1]
				if top.kind == expressionTokenLeftParen {
					break
				}
				if expressionOperatorPrecedence(top.kind) < expressionOperatorPrecedence(token.kind) {
					break
				}
				output = append(output, top)
				operators = operators[:len(operators)-1]
			}
			operators = append(operators, token)
		case expressionTokenLeftParen:
			operators = append(operators, token)
		case expressionTokenRightParen:
			matchedLeftParen := false
			for len(operators) > 0 {
				top := operators[len(operators)-1]
				operators = operators[:len(operators)-1]
				if top.kind == expressionTokenLeftParen {
					matchedLeftParen = true
					break
				}
				output = append(output, top)
			}
			if !matchedLeftParen {
				return nil, false
			}
		default:
			return nil, false
		}
	}

	for len(operators) > 0 {
		top := operators[len(operators)-1]
		operators = operators[:len(operators)-1]
		if top.kind == expressionTokenLeftParen || top.kind == expressionTokenRightParen {
			return nil, false
		}
		output = append(output, top)
	}

	return output, true
}

func tokenizeExpression(expr string) []expressionToken {
	tokens := make([]expressionToken, 0, len(expr)/2)
	segmentStart := 0
	inQuote := byte(0)

	appendCondition := func(start, end int) {
		if end <= start {
			return
		}
		condition := strings.TrimSpace(expr[start:end])
		if condition != "" {
			tokens = append(tokens, expressionToken{
				kind:  expressionTokenCondition,
				value: condition,
			})
		}
	}

	for i := 0; i < len(expr); i++ {
		ch := expr[i]
		if inQuote != 0 {
			if ch == inQuote {
				inQuote = 0
			}
			continue
		}

		switch ch {
		case '\'', '"':
			inQuote = ch
		case '(':
			appendCondition(segmentStart, i)
			tokens = append(tokens, expressionToken{kind: expressionTokenLeftParen})
			segmentStart = i + 1
		case ')':
			appendCondition(segmentStart, i)
			tokens = append(tokens, expressionToken{kind: expressionTokenRightParen})
			segmentStart = i + 1
		default:
			if op, isLogical := parseLogicalOperatorAt(expr, i); isLogical {
				appendCondition(segmentStart, i)
				tokens = append(tokens, expressionToken{kind: op})
				if op == expressionTokenAnd {
					i += 2
				} else {
					i += 1
				}
				segmentStart = i + 1
			}
		}
	}

	appendCondition(segmentStart, len(expr))
	return tokens
}

func parseLogicalOperatorAt(expr string, idx int) (expressionTokenType, bool) {
	if idx+3 <= len(expr) &&
		strings.EqualFold(expr[idx:idx+3], "AND") &&
		isLogicalTokenBoundary(expr, idx-1) &&
		isLogicalTokenBoundary(expr, idx+3) {
		return expressionTokenAnd, true
	}
	if idx+2 <= len(expr) &&
		strings.EqualFold(expr[idx:idx+2], "OR") &&
		isLogicalTokenBoundary(expr, idx-1) &&
		isLogicalTokenBoundary(expr, idx+2) {
		return expressionTokenOr, true
	}
	return 0, false
}

func isLogicalTokenBoundary(expr string, idx int) bool {
	if idx < 0 || idx >= len(expr) {
		return true
	}

	switch expr[idx] {
	case ' ', '\t', '\n', '\r', '(', ')':
		return true
	default:
		return false
	}
}

func expressionOperatorPrecedence(kind expressionTokenType) int {
	switch kind {
	case expressionTokenOr:
		return 1
	case expressionTokenAnd:
		return 2
	default:
		return 0
	}
}

func evaluateExpressionRPN(tokens []expressionToken, data map[string]any) bool {
	stack := make([]bool, 0, len(tokens))

	for _, token := range tokens {
		switch token.kind {
		case expressionTokenCondition:
			stack = append(stack, evaluateSingleCondition(token.value, data))
		case expressionTokenAnd, expressionTokenOr:
			if len(stack) < 2 {
				return false
			}
			right := stack[len(stack)-1]
			left := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			if token.kind == expressionTokenAnd {
				stack = append(stack, left && right)
			} else {
				stack = append(stack, left || right)
			}
		default:
			return false
		}
	}

	if len(stack) != 1 {
		return false
	}

	return stack[0]
}

func evaluateSingleCondition(expr string, data map[string]any) bool {
	expr = strings.TrimSpace(expr)

	if expr == "true" {
		return true
	}
	if expr == "false" {
		return false
	}

	operators := []string{"!=", "==", ">=", "<=", ">", "<"}

	for _, op := range operators {
		opIdx := strings.Index(expr, op)
		if opIdx == -1 {
			continue
		}

		varName := strings.TrimSpace(expr[:opIdx])
		expectedValue := strings.TrimSpace(expr[opIdx+len(op):])
		expectedValue = strings.Trim(expectedValue, "'\"")

		actualValue := ""
		if val, exists := data[varName]; exists && val != nil {
			actualValue = fmt.Sprintf("%v", val)
		}

		return compareValues(actualValue, op, expectedValue)
	}
	return false
}

func compareValues(actual, operator, expected string) bool {
	switch operator {
	case "==":
		return actual == expected
	case "!=":
		return actual != expected
	case ">", "<", ">=", "<=":
		actualNum, err1 := parseNumber(actual)
		expectedNum, err2 := parseNumber(expected)
		if err1 == nil && err2 == nil {
			switch operator {
			case ">":
				return actualNum > expectedNum
			case "<":
				return actualNum < expectedNum
			case ">=":
				return actualNum >= expectedNum
			case "<=":
				return actualNum <= expectedNum
			}
		}
		switch operator {
		case ">":
			return actual > expected
		case "<":
			return actual < expected
		case ">=":
			return actual >= expected
		case "<=":
			return actual <= expected
		}
	}
	return false
}

func parseNumber(s string) (float64, error) {
	trimmed := strings.TrimSpace(s)

	if n, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return n, nil
	}

	var n float64
	_, err := fmt.Sscanf(trimmed, "%f", &n)
	return n, err
}
