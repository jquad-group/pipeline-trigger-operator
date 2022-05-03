package controllers

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type StatusChangePredicate struct {
	predicate.Funcs
}

func (StatusChangePredicate) Update(e event.UpdateEvent) bool {
	/*
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
	*/
	return true
}

func (StatusChangePredicate) Create(e event.CreateEvent) bool {
	return false
}

func (StatusChangePredicate) Delete(e event.DeleteEvent) bool {
	return false
}
