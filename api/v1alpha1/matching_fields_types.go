package v1alpha1

import (
	"strings"

	"k8s.io/apimachinery/pkg/fields"
)

type MatchingFields map[string]string

func (m MatchingFields) IsSubsetOf(other fields.Set) bool {
	for key, value := range m {
		if other[key] != value {
			return false
		}
	}
	return true
}

func (mf MatchingFields) ToFieldSelectorString() string {
	var parts []string
	for key, value := range mf {
		parts = append(parts, key+"="+value)
	}
	return strings.Join(parts, ",")
}
