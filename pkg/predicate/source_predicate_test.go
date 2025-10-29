package predicate

import (
	"testing"

	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestSourceRevisionChangePredicateUpdate(t *testing.T) {
	// Create a sourcev1.Source object with an artifact revision
	oldSource := &sourcev1.GitRepository{
		// Create a sample artifact with a revision
		Status: sourcev1.GitRepositoryStatus{
			Artifact: &meta.Artifact{
				Revision: "revision1",
			},
		},
	}

	// Create a sourcev1.Source object with a different artifact revision
	newSource := &sourcev1.GitRepository{
		// Create a sample artifact with a different revision
		Status: sourcev1.GitRepositoryStatus{
			Artifact: &meta.Artifact{
				Revision: "revision2",
			},
		},
	}

	// Test an update event
	updateEvent := event.UpdateEvent{
		ObjectOld: oldSource,
		ObjectNew: newSource,
	}

	// Create a SourceRevisionChangePredicate
	predicate := SourceRevisionChangePredicate{}

	// Test the Update method
	result := predicate.Update(updateEvent)

	// Expect the result to be true since the artifact revision has changed
	if !result {
		t.Errorf("Expected true, but got false")
	}
}

func TestSourceRevisionChangePredicateNoChange(t *testing.T) {
	// Create a sourcev1.Source object with an artifact revision
	oldSource := &sourcev1.GitRepository{
		// Create a sample artifact with a revision
		Status: sourcev1.GitRepositoryStatus{
			Artifact: &meta.Artifact{
				Revision: "revision1",
			},
		},
	}

	// Create a sourcev1.Source object with the same artifact revision
	newSource := &sourcev1.GitRepository{
		// Create a sample artifact with the same revision
		Status: sourcev1.GitRepositoryStatus{
			Artifact: &meta.Artifact{
				Revision: "revision1",
			},
		},
	}

	// Test an update event
	updateEvent := event.UpdateEvent{
		ObjectOld: oldSource,
		ObjectNew: newSource,
	}

	// Create a SourceRevisionChangePredicate
	predicate := SourceRevisionChangePredicate{}

	// Test the Update method
	result := predicate.Update(updateEvent)

	// Expect the result to be false since the artifact revision hasn't changed
	if result {
		t.Errorf("Expected false, but got true")
	}
}

func TestSourceRevisionChangePredicateCreate(t *testing.T) {
	// Create a sourcev1.Source object with an artifact revision
	newSource := &sourcev1.GitRepository{
		// Create a sample artifact with a revision
		Status: sourcev1.GitRepositoryStatus{
			Artifact: &meta.Artifact{
				Revision: "revision1",
			},
		},
	}

	// Test a create event
	createEvent := event.CreateEvent{
		Object: newSource,
	}

	// Create a SourceRevisionChangePredicate
	predicate := SourceRevisionChangePredicate{}

	// Test the Create method
	result := predicate.Create(createEvent)

	// Expect the result to be false since the Create method always returns false
	if result {
		t.Errorf("Expected false, but got true")
	}
}

func TestSourceRevisionChangePredicateDelete(t *testing.T) {
	// Create a sourcev1.Source object with an artifact revision
	oldSource := &sourcev1.GitRepository{
		// Create a sample artifact with a revision
		Status: sourcev1.GitRepositoryStatus{
			Artifact: &meta.Artifact{
				Revision: "revision1",
			},
		},
	}

	// Test a delete event
	deleteEvent := event.DeleteEvent{
		Object: oldSource,
	}

	// Create a SourceRevisionChangePredicate
	predicate := SourceRevisionChangePredicate{}

	// Test the Delete method
	result := predicate.Delete(deleteEvent)

	// Expect the result to be false since the Delete method always returns false
	if result {
		t.Errorf("Expected false, but got true")
	}
}
