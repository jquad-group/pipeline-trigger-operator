/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Recorder struct {
	conditionGauge *prometheus.GaugeVec
}

func NewRecorder() *Recorder {
	return &Recorder{
		conditionGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "pipelinetrigger_reconcile_condition",
				Help: "The status of the pipeline trigger's last started pipeline run",
			},
			[]string{"kind", "name", "namespace", "type", "status"},
		),
	}
}

func (r *Recorder) Collectors() []prometheus.Collector {
	return []prometheus.Collector{
		r.conditionGauge,
	}
}

func (r *Recorder) RecordCondition(ref corev1.ObjectReference, condition metav1.Condition) {
	for _, status := range []string{string(metav1.ConditionTrue), string(metav1.ConditionFalse), string(metav1.ConditionUnknown)} {
		var value float64

		if status == string(condition.Status) {
			value = 1
		}

		r.conditionGauge.WithLabelValues(ref.Kind, ref.Name, ref.Namespace, condition.Type, status).Set(value)
	}
}
