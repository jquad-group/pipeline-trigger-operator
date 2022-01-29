# Pipeline Trigger Operator

The Pipeline Trigger Operator listens for events from the Flux v2 `ImagePolicy` or `GitRepository` resources and creates a Tekton `PipelineRun` for a given `Pipeline` resouce.

**Automated creation of Tekton PipelineRuns on events from Flux resources**

Using the automated pipeline trigger operator is based on the following resources:
1. `GitRepository` - [**Flux** resource](https://fluxcd.io/docs/components/source/gitrepositories/), configure as required
2. `ImageRepository` - [**Flux** resource](https://fluxcd.io/docs/components/image/imagerepositories/), configure as required
3. `ImagePolicy` - [**Flux** resource](https://fluxcd.io/docs/components/image/imagepolicies/), configure as required
4. `Pipeline` - [**Tekton** resource](https://tekton.dev/docs/pipelines/pipelines/), configure as required
5. `PipelineTrigger` - **PipelineTrigger* resource, configuration description in this readme

![Workflow](https://github.com/jquad-group/pipeline-trigger-operator/blob/main/img/pipeline-trigger-operator.svg)

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
        value: "main" # or "$(branch)" - branch name taken from the Flux GitRepository resource
      - name: "commit-id"
        value: "03da4fdbf8f3e027fb56dd0d96244c951a24f2b4" # or "$(commit)" - commit id taken from the Flux Gitrepository resource
```

# Installation

Run the following command: 

`kubectl apply -f https://github.com/jquad-group/pipeline-trigger-operator/releases/download/v0.1.4/release.yaml`

The operator is installed in the `pipeline-trigger-operator-system` namespace. 

After the installation of the operator, the `PipelineTrigger` resource is added to the kubernetes cluster.

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

1. Create the example namespace `jq-example-namespace`. In this namespace are deployed all the examples: 

 `kubectl create ns jq-example-namespace`

2. Deploy the `gitrepository.yaml`. The `GitRepository` resource clones a branch for given git repository: 

```
$ kubectl apply -f examples/create-pipeline-on-git-push/gitrepository.yaml
$ kubectl describe gitrepository microservice-code-repo -n jq-example-namespace
Name:         microservice-code-repo
Namespace:    jq-example-namespace
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
 $ kubectl describe pipelinetrigger pipelinetrigger-for-git-project -n jq-example-namespace
Name:         pipelinetrigger-for-git-project
Namespace:    jq-example-namespace
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
  Normal  Info    65s   PipelineTrigger  Source microservice-code-repo in namespace jq-example-namespace got new event main/03da4fdbf8f3e027fb56dd0d96244c951a24f2b4
 ```

5. The `PipelineRun` was successfully created: 

```
$ kubectl get pipelineruns -n jq-example-namespace
NAME                                   SUCCEEDED   REASON      STARTTIME   COMPLETIONTIME
pipelinetrigger-for-git-project-mkfx   True        Succeeded   2m41s       2m29s
```

## Example 2: Build a microservice and create the microservice image when a new base image version is released (`FROM` instruction in the `Dockerfile`)

1. Create the example namespace `jq-example-namespace`. In this namespace are deployed all the examples: 

 `kubectl create ns jq-example-namespace`

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

If the API source files are changed, the command `make manifests` needs to be run.

Run the project locally by `make deploy`.

## Build the container image 

Build the container image using `docker build . --tag pipeline-trigger-operator` (implicitly builds the operator)

Build and deploy the changed version by running the following command from the `config/default` directory:

`kustomize build . | kubectl apply -f -` 
