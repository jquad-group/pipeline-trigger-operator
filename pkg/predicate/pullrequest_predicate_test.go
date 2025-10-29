package predicate

import (
	"testing"

	pullrequestv1alpha1 "github.com/jquad-group/pullrequest-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestPullRequestStatusChangePredicateUpdate(t *testing.T) {
	predicate := PullRequestStatusChangePredicate{}

	// Create old and new PullRequest objects
	oldPR := &pullrequestv1alpha1.PullRequest{
		Status: pullrequestv1alpha1.PullRequestStatus{
			Conditions: []metav1.Condition{
				{
					Type:   "Ready",
					Status: "True",
				},
			},
		},
	}
	newPR := &pullrequestv1alpha1.PullRequest{
		Status: pullrequestv1alpha1.PullRequestStatus{
			Conditions: []metav1.Condition{
				{
					Type:   "Ready",
					Status: "False",
				},
			},
		},
	}

	// Create an update event with old and new objects
	updateEvent := event.UpdateEvent{
		ObjectOld: oldPR,
		ObjectNew: newPR,
	}

	// Test the Update function
	result := predicate.Update(updateEvent)
	if !result {
		t.Errorf("Update function should return true for status change, but it returned false")
	}

	// Modify the new PullRequest object to simulate a status change
	newPR.Status = pullrequestv1alpha1.PullRequestStatus{
		Conditions: []metav1.Condition{
			{
				Type:   "Ready",
				Status: "False",
			},
		},
	}

	// Test the Update function again
	result = predicate.Update(updateEvent)
	if !result {
		t.Errorf("Update function should return true for status change, but it returned false")
	}
}

func TestPullRequestStatusChangePredicateCreate(t *testing.T) {
	predicate := PullRequestStatusChangePredicate{}

	// Create a create event
	createEvent := event.CreateEvent{}

	// Test the Create function
	result := predicate.Create(createEvent)
	if result {
		t.Errorf("Create function should always return false, but it returned true")
	}
}

func TestPullRequestStatusChangePredicateDelete(t *testing.T) {
	predicate := PullRequestStatusChangePredicate{}

	// Create a delete event
	deleteEvent := event.DeleteEvent{}

	// Test the Delete function
	result := predicate.Delete(deleteEvent)
	if result {
		t.Errorf("Delete function should always return false, but it returned true")
	}
}
