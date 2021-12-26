# Pipeline Trigger Operator

The Pipeline Trigger Operator listens for events from the Flux v2 `ImagePolicy` resource and creates a Tekton `PipelineRun` from a given `Pipeline` resouce.

The `PipelineRun` is automatically deleted if successful. 

# Deployment

`git clone https://github.com/jquad-group/pipeline-trigger-operator.git`

`cd config/default`

`kustomize build . | kubectl apply -f -`

# Examples

`config/samples`

# Development

`git clone https://github.com/jquad-group/pipeline-trigger-operator.git`

`make`

`make manifests` 

`docker build . --tag harbor.jquad.rocks/library/pipeline-trigger-operator:v0.0.1`

`docker push harbor.jquad.rocks/library/pipeline-trigger-operator:v0.0.1`



