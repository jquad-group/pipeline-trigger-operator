package meta

import (
	"testing"
)

func TestTypeMeta(t *testing.T) {
	kind := "TestKind"
	apiVersion := "v1"

	tm := TypeMeta(kind, apiVersion)

	if tm.Kind != kind {
		t.Errorf("TypeMeta() Kind is incorrect. Got: %s, Expected: %s", tm.Kind, kind)
	}
	if tm.APIVersion != apiVersion {
		t.Errorf("TypeMeta() APIVersion is incorrect. Got: %s, Expected: %s", tm.APIVersion, apiVersion)
	}
}

func TestObjectMeta(t *testing.T) {
	namespace := "test-namespace"
	name := "test-name"
	expectedLabels := map[string]string{"key1": "value1", "key2": "value2"}

	opts := AddLabels(expectedLabels)
	om := ObjectMeta(NamespacedName(namespace, name), opts)

	if om.Namespace != namespace {
		t.Errorf("ObjectMeta() Namespace is incorrect. Got: %s, Expected: %s", om.Namespace, namespace)
	}
	if om.Name != name {
		t.Errorf("ObjectMeta() Name is incorrect. Got: %s, Expected: %s", om.Name, name)
	}
	if !labelsEqual(om.Labels, expectedLabels) {
		t.Errorf("ObjectMeta() Labels are incorrect. Got: %v, Expected: %v", om.Labels, expectedLabels)
	}
}

func labelsEqual(labels, expectedLabels map[string]string) bool {
	if len(labels) != len(expectedLabels) {
		return false
	}
	for key, value := range labels {
		if expectedValue, ok := expectedLabels[key]; !ok || value != expectedValue {
			return false
		}
	}
	return true
}

func TestNamespacedName(t *testing.T) {
	namespace := "test-namespace"
	name := "test-name"

	nsName := NamespacedName(namespace, name)

	if nsName.Namespace != namespace {
		t.Errorf("NamespacedName() Namespace is incorrect. Got: %s, Expected: %s", nsName.Namespace, namespace)
	}
	if nsName.Name != name {
		t.Errorf("NamespacedName() Name is incorrect. Got: %s, Expected: %s", nsName.Name, name)
	}
}
