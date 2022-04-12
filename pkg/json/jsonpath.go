package json

import (
	"errors"
	"strings"

	"github.com/spyzhov/ajson"
)

func EvalExpr(json string, expr string) (string, error) {
	root := ajson.Must(ajson.Unmarshal([]byte(json)))
	_, err := ajson.Eval(root, expr)
	if err != nil {
		return "", err
	}
	result := ajson.Must(ajson.Eval(root, expr))
	return result.String(), nil
}

func Exists(json string, expr string) (bool, error) {
	root := ajson.Must(ajson.Unmarshal([]byte(json)))
	result, err := ajson.Eval(root, expr)
	if err != nil {
		return false, err
	}
	if strings.Contains(result.String(), "null") {
		return false, errors.New("Expression " + expr + " evaluates to null.")
	}
	return true, nil
}
