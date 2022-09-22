package predicate

import (
	"reflect"

	pullrequestv1alpha1 "github.com/jquad-group/pullrequest-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type PullRequestStatusChangePredicate struct {
	predicate.Funcs
}

func (PullRequestStatusChangePredicate) Update(e event.UpdateEvent) bool {

	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	oldPullRequest, ok := e.ObjectOld.(*pullrequestv1alpha1.PullRequest)
	if !ok {
		return false
	}

	newPullRequest, ok := e.ObjectNew.(*pullrequestv1alpha1.PullRequest)
	if !ok {
		return false
	}

	result := !reflect.DeepEqual(oldPullRequest.Status, newPullRequest.Status)
	return result
}

func (PullRequestStatusChangePredicate) Create(e event.CreateEvent) bool {
	return false
}

func (PullRequestStatusChangePredicate) Delete(e event.DeleteEvent) bool {
	return false
}
