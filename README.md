# Pipeline Trigger Operator

The Pipeline Trigger Operator listens for events from the Flux v2 `ImagePolicy` and `GitRepository` resources and creates a Tekton `PipelineRun` for a given `Pipeline` resouce.

# Installation

Clone the project and from directory `config/default` run:

`kustomize build . | kubectl apply -f -`

The operator is installed in the `pipeline-trigger-operator-system` namespace. 

# Usage

Examples can be found in the `config/samples` directory of the project. 

# Development

The project is built with: `make`

If API source files are changed, the command `make manifests` needs to be run. 

Build the container image using `docker build . --tag harbor.jquad.rocks/library/pipeline-trigger-operator:v0.0.1`



