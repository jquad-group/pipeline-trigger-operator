module github.com/jquad-group/pipeline-trigger-operator

go 1.16

require (
	github.com/Azure/go-autorest/autorest v0.11.18 // indirect
	github.com/fluxcd/image-reflector-controller/api v0.13.2
	github.com/fluxcd/source-controller/api v0.15.0 
	github.com/form3tech-oss/jwt-go v3.2.3+incompatible // indirect
	github.com/go-logr/logr v0.4.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/tektoncd/pipeline v0.31.0
	k8s.io/api v0.21.4 // indirect
	k8s.io/apimachinery v0.21.4
	k8s.io/client-go v0.21.4
	k8s.io/utils v0.0.0-20210819203725-bdf08cb9a70a // indirect
	sigs.k8s.io/controller-runtime v0.9.7
)
