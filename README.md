# Pipeline Trigger Operator

The Pipeline Trigger Operator listens for events from the Flux v2 `ImagePolicy` or `GitRepository` resources and creates a Tekton `PipelineRun` for a given `Pipeline` resouce.

**Automated creation of Tekton PipelineRuns on events from Flux resources**

Using the automated pipeline trigger operator is based on the following resources:
1. `GitRepository` - [**Flux** resource](https://fluxcd.io/docs/components/source/gitrepositories/), configure as required
2. `ImageRepository` - [**Flux** resource](https://fluxcd.io/docs/components/image/imagerepositories/), configure as required
3. `ImagePolicy` - [**Flux** resource](https://fluxcd.io/docs/components/image/imagepolicies/), configure as required
4. `Pipeline` - [**Tekton** resource](https://tekton.dev/docs/pipelines/pipelines/), configure as required
```diff
5. `PipelineTrigger` - **JQuad** resource, configuration description in this readme
```

![Workflow](https://raw.githubusercontent.com/jquad-group/pipeline-trigger-operator/main/img/pipeline-trigger-operator.svg)

# PipelineTrigger Specification

```
apiVersion: pipeline.jquad.rocks/v1alpha1
kind: PipelineTrigger
metadata:
  name: pipelinetrigger-sample-image
  namespace: jq-example-namespace
spec:
  # Source can be both ImagePolicy as well as GitRepository
  # The operator subscribes for events from these resources
  source: 
    kind: ImagePolicy
    name: latest-image-notifier
  # The configuration of the Tekton Pipeline 
  pipeline: 
    # The name of Pipeline that is used for the creation of the PipelineRun resources
    name: build-and-push-pipeline
    # Your kubernetes service account name
    serviceAccountName: build-bot
    # Max number of executed pipelines that should remain on the cluster
    maxHistory: 5
    # Number of retries to execute a pipeline
    retries: 3
    # The workspace for the tekton pipeline
    workspace:
      name: workspace
      size: 1Gi
      accessMode: ReadWriteOnce
    # The specific input parameters for the pipeline that is used for the creation of the PipelineRun 
    inputParams:
      - name: "repo-url"
        value: "https://github.com/my-project.git"
      - name: "branch-name"
        value: "main"
```

# Installation

Clone the project and from directory `config/default` run:

`kustomize build . | kubectl apply -f -`

The operator is installed in the `pipeline-trigger-operator-system` namespace. 

After the installation of the operator, the `PipelineTrigger` resource is added to the kubernetes cluster.

# Usage

Examples can be found in the `examples` directory of the project. 

# Development

The project is built with: `make`

If API source files are changed, the command `make manifests` needs to be run. 

Build the container image using `docker build . --tag pipeline-trigger-operator`

# Example 1: Listen to updates from a Flux v2 image policy

1. Deploy an `ImageRepository` (Flux) resource

```
apiVersion: image.toolkit.fluxcd.io/v1beta1
kind: ImageRepository
metadata:
  name: microservice-container-repo 
  namespace: jq-example-namespace
spec:
  interval: 5m
  image: <URL to the image>
```

2. Deploy an `ImagePolicy` (Flux) resource in order to select the latest image version

```
apiVersion: image.toolkit.fluxcd.io/v1beta1
kind: ImagePolicy
metadata:
  name: latest-image-notifier
  namespace: jq-example-namespace
spec:
  imageRepositoryRef:
    name: microservice-container-repo
    namespace: jq-example-namespace
  policy:
    semver:
      range: 0.0.x

```

3. Deploy your tekton build pipeline

4. Deploy the `PipelineTrigger` resource which creates a new pipeline when the `ImagePolicy` triggers an event
The pipelintrigger resource has the following specific elements:

```
apiVersion: pipeline.jquad.rocks/v1alpha1
kind: PipelineTrigger
metadata:
  name: pipelinetrigger-sample-image
  namespace: jq-example-namespace
spec:
  source: 
    kind: ImagePolicy
    name: latest-image-notifier
  pipeline: 
    name: <your pipeline name>
    serviceAccountName: <your kubernetes service account name>
    maxHistory: <max number of executed pipelines that should remain on the cluster>
    retries: <number of retries to execute a pipeline>
    workspace:
      name: <your workspace name>
      size: <size of the workspace>
      accessMode: ReadWriteOnce
    inputParams:
      - name: "repo-url"
        value: "git@github.com:jquad-group/www-jquad.git"
      - name: "branch-name"
        value: "main"
```



