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

package main

import (
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"
	controllers "github.com/jquad-group/pipeline-trigger-operator/internal/controller"
	metricsApi "github.com/jquad-group/pipeline-trigger-operator/pkg/metrics"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	crtlruntimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	imagereflectorv1 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	pullrequestv1alpha1 "github.com/jquad-group/pullrequest-operator/api/v1alpha1"
	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(imagereflectorv1.AddToScheme(scheme))
	utilruntime.Must(tektondevv1.AddToScheme(scheme))
	utilruntime.Must(sourcev1.AddToScheme(scheme))
	utilruntime.Must(pipelinev1alpha1.AddToScheme(scheme))
	utilruntime.Must(pullrequestv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var enableSecondCluster bool
	var secondClusterAddr string
	var secondClusterBearerToken string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&enableSecondCluster, "second-cluster", false,
		"Enable second cluster. "+
			"Enabling this will ensure there is only one pipelinerun in a 2 cluster setup.")
	flag.StringVar(&secondClusterAddr, "second-cluster-address", "", "The address of the second server API server.")
	flag.StringVar(&secondClusterBearerToken, "second-cluster-bearer-token", "", "The bearer token for the communication with the second API server.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	metricsRecorder := metricsApi.NewRecorder()
	crtlruntimemetrics.Registry.MustRegister(metricsRecorder.Collectors()...)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "bb9e0b30.jquad.rocks",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if enableSecondCluster {
		if err = (&controllers.PipelineTriggerReconciler{
			Client:                   mgr.GetClient(),
			Scheme:                   mgr.GetScheme(),
			MetricsRecorder:          metricsRecorder,
			SecondClusterEnabled:     enableSecondCluster,
			SecondClusterAddr:        secondClusterAddr,
			SecondClusterBearerToken: secondClusterBearerToken,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "PipelineTrigger")
			os.Exit(1)
		}
	} else {
		if err = (&controllers.PipelineTriggerReconciler{
			Client:               mgr.GetClient(),
			Scheme:               mgr.GetScheme(),
			MetricsRecorder:      metricsRecorder,
			SecondClusterEnabled: false,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "PipelineTrigger")
			os.Exit(1)
		}
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}

}

/*
func getEnv(envVar string) (string, error) {

		ns, found := os.LookupEnv(envVar)
		if !found {
			return "", fmt.Errorf("%s must be set", envVar)
		}
		return ns, nil
	}
*/
