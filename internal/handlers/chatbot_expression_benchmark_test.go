package handlers

import "testing"

var expressionEvalSink bool

var benchmarkExprData = map[string]interface{}{
	"status": "vip",
	"amount": "150",
	"name":   "alice",
	"a":      "1",
	"b":      "2",
	"c":      "3",
	"d":      "4",
	"e":      "5",
	"f":      "6",
}

const benchmarkExprSimple = "status == 'vip'"
const benchmarkExprComplex = "(status == 'vip' OR amount > 100) AND (name != '' OR a == '1') AND (b == '2' OR c == '0') AND (d >= '4' AND e <= '5')"
const benchmarkExprNested = "(((a == '1' AND b == '2') OR (c == '9' AND d == '4')) AND ((e == '5' OR f == '0') AND (status == 'vip' OR name == 'bob')))"

func BenchmarkEvaluateExpressionSimple(b *testing.B) {
	b.ReportAllocs()
	result := false
	for i := 0; i < b.N; i++ {
		result = evaluateExpression(benchmarkExprSimple, benchmarkExprData)
	}
	expressionEvalSink = result
}

func BenchmarkEvaluateExpressionComplex(b *testing.B) {
	b.ReportAllocs()
	result := false
	for i := 0; i < b.N; i++ {
		result = evaluateExpression(benchmarkExprComplex, benchmarkExprData)
	}
	expressionEvalSink = result
}

func BenchmarkEvaluateExpressionNested(b *testing.B) {
	b.ReportAllocs()
	result := false
	for i := 0; i < b.N; i++ {
		result = evaluateExpression(benchmarkExprNested, benchmarkExprData)
	}
	expressionEvalSink = result
}
