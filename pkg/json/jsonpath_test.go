package json

import (
	"testing"
)

func TestEvalExpr(t *testing.T) {
	jsonData := `{
		"name": "John",
		"age": 30,
		"city": "New York"
	}`
	expr := "$.name" // Use '$' to start from the root

	result, err := EvalExpr(jsonData, expr)

	if err != nil {
		t.Errorf("EvalExpr returned an error: %v", err)
	}

	expectedResult := "John"
	if result != "\""+expectedResult+"\"" {
		t.Errorf("EvalExpr result is not as expected. Got: %s, Expected: %s", result, expectedResult)
	}
}

func TestEvalExpr_InvalidExpression(t *testing.T) {
	jsonData := `{
		"name": "John",
		"age": 30,
		"city": "New York"
	}`
	expr := ".nonexistentfield"

	_, err := EvalExpr(jsonData, expr)

	if err == nil {
		t.Error("EvalExpr should return an error for an invalid expression, but it didn't.")
	}
}

func TestExists(t *testing.T) {
	jsonData := `{
		"name": "John",
		"age": 30,
		"city": "New York"
	}`
	expr := "$.name" // Use '$' to start from the root

	exists, err := Exists(jsonData, expr)

	if err != nil {
		t.Errorf("Exists returned an error: %v", err)
	}

	if !exists {
		t.Error("Exists should return true, but it returned false.")
	}
}

func TestExists_NonexistentField(t *testing.T) {
	jsonData := `{
		"name": "John",
		"age": 30,
		"city": "New York"
	}`
	expr := "$.nonexistentfield" // Use '$' to start from the root

	exists, err := Exists(jsonData, expr)

	if err.Error() != "Expression $.nonexistentfield evaluates to null." {
		t.Errorf("Exists returned an error: %v", err)
	}

	if exists {
		t.Error("Exists should return false, but it returned true.")
	}
}

func TestExists_NullValue(t *testing.T) {
	jsonData := `{
		"name": null,
		"age": 30,
		"city": "New York"
	}`
	expr := ".name"

	_, err := Exists(jsonData, expr)

	if err == nil {
		t.Error("Exists should return an error for null value, but it didn't.")
		if err.Error() != "Expression .name evaluates to null." {
			t.Errorf("Exists returned an unexpected error message: %v", err.Error())
		}
	}
}

func TestExists_EmptyJSON(t *testing.T) {
	jsonData := `{}` // Empty JSON object
	expr := "$.name" // Use '$' to start from the root

	_, err := Exists(jsonData, expr)

	if err.Error() != "Expression $.name evaluates to null." {
		t.Errorf("Exists returned an error: %v", err)
	}

	if err == nil {
		t.Errorf("Exists should return an error for empty JSON, but it didn't.")
	}
}
