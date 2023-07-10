# Pipeline Trigger Operator

[![Go Report Card](https://goreportcard.com/badge/jquad-group/pipeline-trigger-operator)](https://goreportcard.com/report/jquad-group/pipeline-trigger-operator)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/jquad-group/pipeline-trigger-operator/blob/main/LICENSE)

The `Pipeline Trigger Operator` increases the level of automation of the build pipeline of a software component by enhancing it with reactive capabilities to react to events in the build environment.

The `Pipeline Trigger Operator` runs on [Kubernetes](https://kubernetes.io/docs/home/) and integrates itself in the [GitOps operational framework](https://about.gitlab.com/topics/gitops/) to allow for automated creation of build pipelines. It does that by listening for events from the [Flux v2](https://fluxcd.io/flux/) `ImagePolicy` and `GitRepository` resources as well as from the [pullrequest-operator](https://github.com/jquad-group/pullrequest-operator/) `PullRequest` resource and reacts by creating a [Tekton](https://tekton.dev/docs/) `PipelineRun` for a given `Pipeline` resouce.

**Automated creation of Tekton PipelineRuns on events from Flux resources**

The use of the automated pipeline trigger operator is based on the following resources:
1. `GitRepository` - [**Flux** resource](https://fluxcd.io/docs/components/source/gitrepositories/), configure as required
2. `ImageRepository` - [**Flux** resource](https://fluxcd.io/docs/components/image/imagerepositories/), configure as required
3. `ImagePolicy` - [**Flux** resource](https://fluxcd.io/docs/components/image/imagepolicies/), configure as required
4. `PullRequest` - [**PullRequest** resource](https://github.com/jquad-group/pullrequest-operator/), configure as required
5. `Pipeline` - [**Tekton** resource](https://tekton.dev/docs/pipelines/pipelines/), configure as required
6. `PipelineTrigger` - **PipelineTrigger* resource, configuration description in this readme

[Short video with the components overview](https://www.youtube.com/watch?v=3TmczsYnDNc)

![Workflow](https://github.com/jquad-group/pipeline-trigger-operator/blob/main/img/pipeline-trigger-operator.svg)

# PipelineTrigger Specification

```
apiVersion: pipeline.jquad.rocks/v1alpha1
kind: PipelineTrigger
metadata:
  name: pipelinetrigger-sample-image
  namespace: jq-example-namespace
spec:
  # Source can be ImagePolicy, GitRepository, or PullRequest
  # The operator subscribes for events from these resources
  source: 
    kind: ImagePolicy
    name: latest-image-notifier
  # The PipelineRunSpec of Tekton
  # See the Tekton API: https://tekton.dev/docs/pipelines/pipeline-api/#tekton.dev/v1beta1.PipelineRun 
  pipelineRunSpec: 
    # The name of Pipeline that is used for the creation of the PipelineRun resources
    pipelineRef:
      name: build-and-push-pipeline
    # Your kubernetes service account name
    serviceAccountName: build-bot
    # The workspace for the tekton pipeline
    workspaces:
    - name: workspace
      volumeClaimTemplate:
        spec:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 16Mi
          volumeMode: Filesystem
    # The specific input parameters for the pipeline that is used for the creation of the PipelineRun 
    params:
    - name: "repo-url"
      value: "https://github.com/my-project.git"
    - name: "branch-name"
      value: "main" # or JSON path expression, e.g. $.imageName
    - name: "commit-id"
      value: "03da4fdbf8f3e027fb56dd0d96244c951a24f2b4" # or JSON path expression - commit id taken from the Flux Gitrepository resource
```

## Using dynamic input parameters for the pipeline

The pipeline-trigger-operator accepts json path expressions. 

E.g. given the following status:

```
Status:
  Branches:
  Git Repository:
    Branch Name:  main
    Commit Id:    03da4fdbf8f3e027fb56dd0d96244c951a24f2b4
    Details:                 {"branchName":"main","commitId":"03da4fdbf8f3e027fb56dd0d96244c951a24f2b4","repositoryName":"microservice-code-repo"}
    Repository Name:         microservice-code-repo
```
the branch name can be extracted from the `Details` using the expression `$.branchName` as a input parameter for the pipeline:

```
    params:
    - name: "branch-name"
      value: $.branchName
```

# Prerequisites 

- Flux is already installed on the cluster: https://fluxcd.io/docs/installation/#install-the-flux-cli

- The PullRequest Operator is already installed on the cluster: https://github.com/jquad-group/pullrequest-operator/

# Installation

Run the following command: 
```
kubectl apply -f https://github.com/jquad-group/pipeline-trigger-operator/releases/latest/download/release.yaml
```

or with RBAC Proxy:
```
kubectl apply -f https://github.com/jquad-group/pipeline-trigger-operator/releases/latest/download/release-with-rbac-proxy.yaml
```

The operator is installed in the `pipeline-trigger-operator-system` namespace. 

After the installation of the operator, the `PipelineTrigger` resource is added to the kubernetes cluster.

# Install with Monitoring Support

The Pipeline Trigger Operator utilizes the following components:
- Prometheus Operator
- Prometheus
- Grafana 

Run the following command: 
```
kubectl apply -f https://github.com/jquad-group/pipeline-trigger-operator/releases/latest/download/release-with-monitoring.yaml
```

Then you can import the dashboard in Grafana from `config/dashboard/dashboard.json`.

![Dashboard](https://github.com/jquad-group/pipeline-trigger-operator/blob/main/img/dashboard.png)


# Cross Data Center Distribution Lock

If you deploy the operator in a multi cluster setup, but only of them must be working at a time, the argument `--cross-ds-leader-elect` should be added in the `Deployment`:

```
    spec:
      containers:
      - args:
        - --leader-elect
        - --cross-dc-leader-elect
```

In order for the cross datacenter leader election to work a MySQL backend is needed. The configuration for the backend can be set via enviroment variables or via `Secrets` or `ConfigMaps`:

```
        env:
          - name: username
            value: "root" 
          - name: password
            value: "mypassword"            
          - name: address
            value: "mysql.cluster:3306"
          - name: plaintext_connection_allowed
            value: "true"
          - name: ha_enabled
            value: "true"            
```

# Usage

Examples can be found in the `examples` directory of the project. 

## Example 1: Build a microservice and create the microservice image on git push on the master branch 

1. Create the example namespace `jq-example-git`. In this namespace are deployed all the examples: 

 `kubectl create ns jq-example-git`

2. Deploy the `gitrepository.yaml`. The `GitRepository` resource clones a branch for given git repository: 

```
$ kubectl apply -f examples/create-pipeline-on-git-push/gitrepository.yaml
$ kubectl describe gitrepository microservice-code-repo -n jq-example-git
Name:         microservice-code-repo
Namespace:    jq-example-git
...
Events:
  Type    Reason  Age   From               Message
  ----    ------  ----  ----               -------
  Normal  info    17s   source-controller  Fetched revision: main/03da4fdbf8f3e027fb56dd0d96244c951a24f2b4
```

3. Deploy the `tekton-pipelines.yaml`. This a dummy tekton pipeline, which is started when the `GitRepository` detects new commits: 

 `kubectl apply -f examples/create-pipeline-on-git-push/tekton-pipelines.yaml`

4. Deploy the `pipelinetrigger.yaml`:

 ```
 $ kubectl apply -f examples/create-pipeline-on-git-push/pipelinetrigger.yaml
 pipelinetrigger.pipeline.jquad.rocks/pipelinetrigger-for-git-project created
 $ kubectl describe pipelinetrigger pipelinetrigger-for-git-project -n jq-example-git
Name:         pipelinetrigger-for-git-project
Namespace:    jq-example-git
...
Status:
  Conditions:
    Last Transition Time:  2022-01-12T15:40:26Z
    Message:
    Observed Generation:   1
    Reason:                Succeded
    Status:                True
    Type:                  Success
  Latest Event:            main/03da4fdbf8f3e027fb56dd0d96244c951a24f2b4
  Latest Pipeline Run:     pipelinetrigger-for-git-project-mkfx
Events:
  Type    Reason  Age   From             Message
  ----    ------  ----  ----             -------
  Normal  Info    65s   PipelineTrigger  Source microservice-code-repo in namespace jq-example-git got new event main/03da4fdbf8f3e027fb56dd0d96244c951a24f2b4
 ```

5. The `PipelineRun` was successfully created: 

```
$ kubectl get pipelineruns -n jq-example-git
NAME                                   SUCCEEDED   REASON      STARTTIME   COMPLETIONTIME
pipelinetrigger-for-git-project-mkfx   True        Succeeded   2m41s       2m29s
```

## Example 2: Build a microservice and create the microservice image when a new base image version is released (`FROM` instruction in the `Dockerfile`)

1. Create the example namespace `jq-example-image`. In this namespace are deployed all the examples: 

 `kubectl create ns jq-example-image`

2. Deploy the `imagerepository.yaml`. The `ImageRepository` resource fetches the tags for a given container registry: 

 `kubectl apply -f examples/create-pipeline-on-image-update/imagerepository.yaml`

3. Deploy the `imagepolicy.yaml`. The `ImagePolicy` resource selects the latest tag for the list of images acquired by the `ImageRepository` resource according to defined criteria: 

 `kubectl apply -f examples/create-pipeline-on-image-update/imagepolicy.yaml`

4. Deploy the `tekton-pipelines.yaml`. This a dummy tekton pipeline, which is started when the `ImagePolicy` resource creates a new event: 

 `kubectl apply -f examples/create-pipeline-on-image-update/tekton-pipelines.yaml`

5. Deploy the `pipelinetrigger.yaml`:

 `kubectl apply -f examples/create-pipeline-on-image-update/pipelinetrigger.yaml`

# Development / Contribution

## Build and run locally 

In order to build the project, run `make`.

Run the ginkgo tests with `make test`.

If the API source files are changed, the command `make manifests` needs to be run.

Run the project locally by `make deploy`.

## Build the container image 

Build the container image using `docker build . --tag pipeline-trigger-operator` (implicitly builds the operator)

Build and deploy the changed version by running the following command from the `config/default` directory:

`kustomize build . | kubectl apply -f -` 
