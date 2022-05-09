package predicate

import (
	"reflect"

	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type StatusChangePredicate struct {
	predicate.Funcs
}

func (StatusChangePredicate) Update(e event.UpdateEvent) bool {

	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	oldPipelineRun, ok := e.ObjectOld.(*tektondevv1.PipelineRun)
	if !ok {
		return false
	}

	newPipelineRun, ok := e.ObjectNew.(*tektondevv1.PipelineRun)
	if !ok {
		return false
	}

	result := !reflect.DeepEqual(oldPipelineRun.Status.Status, newPipelineRun.Status.Status)
	return result
}

func (StatusChangePredicate) Create(e event.CreateEvent) bool {
	return false
}

func (StatusChangePredicate) Delete(e event.DeleteEvent) bool {
	return false
}
