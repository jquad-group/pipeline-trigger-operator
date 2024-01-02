package predicate

import (
	"testing"

	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"knative.dev/pkg/apis"
	v1 "knative.dev/pkg/apis/duck/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestStatusChangePredicateUpdate(t *testing.T) {
	// Create sample PipelineRun objects for testing
	oldPipelineRun := &tektondevv1.PipelineRun{
		Status: tektondevv1.PipelineRunStatus{
			Status: v1.Status{
				Conditions: v1.Conditions{
					apis.Condition{
						Type:   "Ready",
						Status: "True",
					},
				},
			},
		},
	}
	newPipelineRun := &tektondevv1.PipelineRun{
		Status: tektondevv1.PipelineRunStatus{
			Status: v1.Status{
				Conditions: v1.Conditions{
					apis.Condition{
						Type:   "Ready",
						Status: "False",
					},
				},
			},
		},
	}

	// Test a status change event
	updateEvent := event.UpdateEvent{
		ObjectOld: oldPipelineRun,
		ObjectNew: newPipelineRun,
	}

	// Create a StatusChangePredicate
	predicate := StatusChangePredicate{}

	// Test the Update method
	result := predicate.Update(updateEvent)

	// Expect the result to be true since the status has changed
	if !result {
		t.Errorf("Expected true, but got false")
	}
}

func TestStatusChangePredicateUpdateNoChange(t *testing.T) {
	// Create sample PipelineRun objects for testing
	oldPipelineRun := &tektondevv1.PipelineRun{
		Status: tektondevv1.PipelineRunStatus{
			Status: v1.Status{
				Conditions: v1.Conditions{
					apis.Condition{
						Type:   "Ready",
						Status: "True",
					},
				},
			},
		},
	}
	newPipelineRun := &tektondevv1.PipelineRun{
		Status: tektondevv1.PipelineRunStatus{
			Status: v1.Status{
				Conditions: v1.Conditions{
					apis.Condition{
						Type:   "Ready",
						Status: "True",
					},
				},
			},
		},
	}

	// Test an event where the status remains the same
	updateEvent := event.UpdateEvent{
		ObjectOld: oldPipelineRun,
		ObjectNew: newPipelineRun,
	}

	// Create a StatusChangePredicate
	predicate := StatusChangePredicate{}

	// Test the Update method
	result := predicate.Update(updateEvent)

	// Expect the result to be false since the status hasn't changed
	if result {
		t.Errorf("Expected false, but got true")
	}
}

func TestStatusChangePredicateUpdateInvalidObjects(t *testing.T) {
	// Test an event with invalid object types
	invalidOldObject := &tektondevv1.TaskRun{}
	invalidNewObject := &tektondevv1.TaskRun{}

	// Test an update event with invalid objects
	updateEvent := event.UpdateEvent{
		ObjectOld: invalidOldObject,
		ObjectNew: invalidNewObject,
	}

	// Create a StatusChangePredicate
	predicate := StatusChangePredicate{}

	// Test the Update method
	result := predicate.Update(updateEvent)

	// Expect the result to be false since the object types are invalid
	if result {
		t.Errorf("Expected false, but got true")
	}
}

func TestStatusChangePredicateCreate(t *testing.T) {
	// Test a create event
	createEvent := event.CreateEvent{}

	// Create a StatusChangePredicate
	predicate := StatusChangePredicate{}

	// Test the Create method
	result := predicate.Create(createEvent)

	// Expect the result to be false since it's always false for create events
	if result {
		t.Errorf("Expected false, but got true")
	}
}

func TestStatusChangePredicateDelete(t *testing.T) {
	// Test a delete event
	deleteEvent := event.DeleteEvent{}

	// Create a StatusChangePredicate
	predicate := StatusChangePredicate{}

	// Test the Delete method
	result := predicate.Delete(deleteEvent)

	// Expect the result to be false since it's always false for delete events
	if result {
		t.Errorf("Expected false, but got true")
	}
}
