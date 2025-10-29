# Pipeline Trigger Operator 

[![Go Report Card](https://goreportcard.com/badge/jquad-group/pipeline-trigger-operator)](https://goreportcard.com/report/jquad-group/pipeline-trigger-operator)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/jquad-group/pipeline-trigger-operator/blob/main/LICENSE)

- [Pipeline Trigger Operator](#pipeline-trigger-operator)
  - [Overview](#overview)
  - [AI Engineer](#ai-engineer)
  - [Features](#features)
  - [Prerequisites](#prerequisites)
    - [Installation](#installation)
    - [Installation with Monitoring Support](#installation-with-monitoring-support)
  - [Specification](#specification)
  - [Usage](#usage)
    - [Example 1: Build a Microservice Image on Git Push](#example-1-build-a-microservice-image-on-git-push)
    - [Example 2: Build a Microservice Image on New Base Image Release](#example-2-build-a-microservice-image-on-new-base-image-release)
    - [Example 3: Build a Microservice Image on New Pull Request](#example-3-build-a-microservice-image-on-new-pull-request)
  - [Advanced Usage](#advanced-usage)
    - [Managed Credentials](#managed-credentials)
      - [Overview](#managed-credentials-overview)
      - [Example: Using ManagedCredential with PipelineTrigger](#example-using-managedcredential-with-pipelinetrigger)
    - [Dynamic Input Parameters](#dynamic-input-parameters)
      - [Example: Starting a PipelineRun on Events from PullRequest](#example-starting-a-pipelinerun-on-events-from-pullrequest)
      - [Example: Starting a PipelineRun on Events from GitRepository](#example-starting-a-pipelinerun-on-events-from-gitrepository)
      - [Example: Starting a PipelineRun on Events from ImagePolicy](#example-starting-a-pipelinerun-on-events-from-imagepolicy)
    - [Dual Cluster Deployment](#dual-cluster-deployment)
  - [Development and Contribution](#development-and-contribution)
    - [Build and run locally](#build-and-run-locally)
    - [Build the container image](#build-the-container-image)

## Overview

The **Pipeline Trigger Operator** is a tool designed to enhance the automation of software build pipelines. It brings reactive capabilities to your build pipeline by allowing it to react to various events within the kubernetes build environment.

The operator runs on [Kubernetes](https://kubernetes.io/docs/home/) and seamlessly integrates into the GitOps operational framework to enable automated build pipeline creation. It accomplishes this by monitoring events from various resources, including [Flux v2](https://fluxcd.io/flux/) `ImagePolicy` and `GitRepository` resources, as well as the [pullrequest-operator](https://github.com/jquad-group/pullrequest-operator/) `PullRequest` resource. When triggered, it creates a [Tekton](https://tekton.dev/docs/) `PipelineRun` for a specified `Pipeline` resource.

[Short video with the components overview](https://www.youtube.com/watch?v=3TmczsYnDNc)

![Workflow](https://github.com/jquad-group/pipeline-trigger-operator/blob/main/img/pipeline-trigger-operator.svg)

## AI Engineer

![Logo](https://github.com/jquad-group/pipeline-trigger-operator/blob/main/img/logo.png)

Still wondering what the pipeline-trigger-operator does or what a reactive pipeline actually is? Our AI Engineer got you covered! **It's mission is simple and it's purpose is clear**:  guide you to the highest level of build pipeline efficiency. Our **'Reactive CI/CD Pipeline Engineer'**, is here to offer expert advice and insights on setting up and managing these pipelines, with a focus on integrating the *pipeline-trigger-operator* and optionally - the *pullrequest-operator*. Whether you're a seasoned developer or new to this domain, our AI Engineer is ready to help you navigate and excel in reactive CI/CD. So, feel free to [reach out to it with your queries](https://chat.openai.com/g/g-2dst34wup-reactive-ci-cd-pipeline-engineer) related to our project!

## Features

- Automated creation of Tekton `PipelineRuns` in response to events from Flux and Pull Request Operator resources.
- Works with `GitRepository`, `ImagePolicy`, and `PullRequest` resources.
- Dynamically configurable `PipelineRun` input parameters using JSON path expressions.
- **Managed Credentials** - Simplified credential management for Tekton pipelines with automatic Secret and ServiceAccount creation.
- Monitoring support with Prometheus, Grafana, and a dedicated dashboard.

## Prerequisites

Before using the **Pipeline Trigger Operator**, ensure you have the following prerequisites in place:

- [Flux](https://fluxcd.io/docs/installation/#install-the-flux-cli) is installed on your Kubernetes cluster.
- The [PullRequest Operator](https://github.com/jquad-group/pullrequest-operator/) is also installed on your cluster.

### Installation

You can install the **Pipeline Trigger Operator** using the following commands in the `pipeline-trigger-operator-system` namespace:

- Without RBAC Proxy:

  ```shell
  kubectl apply -f https://github.com/jquad-group/pipeline-trigger-operator/releases/latest/download/release.yaml
  ```

- Or with RBAC Proxy:

  ```shell
  kubectl apply -f https://github.com/jquad-group/pipeline-trigger-operator/releases/latest/download/release-with-rbac-proxy.yaml
  ```

### Installation with Monitoring Support

The Pipeline Trigger Operator utilizes the following components:
- Prometheus Operator
- Prometheus
- Grafana 

Install using the following command: 

  ```shell
  kubectl apply -f https://github.com/jquad-group/pipeline-trigger-operator/releases/latest/download/release-with-monitoring.yaml
  ```

You can import the Grafana dashboard from `config/dashboard/dashboard.json` to monitor the operator's performance.

![Dashboard](https://github.com/jquad-group/pipeline-trigger-operator/blob/main/img/dashboard.png)

## Specification

```yaml
apiVersion: pipeline.jquad.rocks/v1alpha1
kind: PipelineTrigger
metadata:
  name: pipelinetrigger-sample-image
  namespace: jq-example-namespace
spec:
  # Source can be ImagePolicy, GitRepository, or PullRequest
  # The operator subscribes for events from these resources
  source: 
    apiVersion: image.toolkit.fluxcd.io/v1
    kind: ImagePolicy
    name: latest-image-notifier
  # The Tekton PipelineRun
  # See the Tekton API: https://tekton.dev/docs/pipelines/pipeline-api/#tekton.dev/v1beta1.PipelineRun or https://tekton.dev/docs/pipelines/pipeline-api/#tekton.dev/v1.PipelineRun
  pipelineRun: 
    apiVersion: tekton.dev/v1
    kind: PipelineRun
    metadata:
      # Can be either name or generateName
      generateName: microservice-pipeline-
      # This must be the same with the PipelineTriggers namespace
      namespace: jq-example-namespace
      labels:
        app: microservice
    spec:
      pipelineRef:
        name: release-pipeline-go
      serviceAccountName: build-robot
      podTemplate:
        securityContext:
          fsGroup: 0
          runAsGroup: 0
          runAsUser: 0        
      workspaces:
      - name: workspace
        volumeClaimTemplate:
          spec:
            accessModes:
              - ReadWriteOnce
            resources:
              requests:
                storage: 128Mi
            volumeMode: Filesystem              
    # The specific input parameters for the pipeline used for the creation of the PipelineRun 
      params:
      - name: "repo-url"
        value: "https://github.com/my-project.git"
      - name: "image-name"
        value: "main" # or JSON path expression, e.g. $.imageName
  # Optional: Reference to a ManagedCredential for authentication
  managedCredentialRef:
    apiVersion: v1
    kind: ManagedCredential
    name: github-credential
    namespace: jq-example-namespace
```

## Usage

### Example 1: Build a Microservice Image on Git Push

1. Create an example namespace:

    ```shell
     kubectl create ns jq-example-git
    ```

2. Deploy the `GitRepository` resource to clone a branch from a git repository:

    ```shell
    kubectl apply -f examples/create-pipeline-on-git-push/gitrepository.yaml
    ```

3. Deploy the `tekton-pipelines.yaml` to create a dummy Tekton pipeline, which will be started when the GitRepository detects new commits:

    ```shell
    kubectl apply -f examples/create-pipeline-on-git-push/tekton-pipelines.yaml
    ```

4. Deploy the `pipelinetrigger.yaml` to trigger the pipeline:

    ```shell
    kubectl apply -f examples/create-pipeline-on-git-push/pipelinetrigger.yaml
    ```

The PipelineRun will be created automatically, when new git commit is detected. 

### Example 2: Build a Microservice Image on New Base Image Release

1. Create an example namespace:

    ```shell
    kubectl create ns jq-example-image
    ```

2. Deploy the `ImageRepository` resource to fetch tags for a repository in a container registry:

    ```shell
    kubectl apply -f examples/create-pipeline-on-image-update/imagerepository.yaml
    ```

3. Deploy the `ImagePolicy` resource to select the latest tag from the `ImageRepository`:

    ```shell
    kubectl apply -f examples/create-pipeline-on-image-update/imagepolicy.yaml
    ```

4. Deploy the `tekton-pipelines.yaml` to create a dummy Tekton pipeline, which will be started when the `ImagePolicy` detects new image versions:

    ```shell
    kubectl apply -f examples/create-pipeline-on-image-update/tekton-pipelines.yaml
    ``` 

5. Deploy the `pipelinetrigger.yaml` to trigger the pipeline:

    ```shell
    kubectl apply -f examples/create-pipeline-on-image-update/pipelinetrigger.yaml
    ```

The PipelineRun will be created automatically, when new base image is released. 

### Example 3: Build a Microservice Image on New Pull Request

1. Create an example namespace:

    ```shell
    kubectl create ns jq-example-pr
    ```

2. Deploy the `PullRequest` resource to check for new pull requests for a git repository:

    ```shell
    kubectl apply -f examples/create-pipeline-on-pr/pullrequest.yaml
    ```

3. Deploy the `tekton-pipelines.yaml` to create a dummy Tekton pipeline, which will be started when the GitRepository detects new commits:

    ```shell
    kubectl apply -f examples/create-pipeline-on-pr/tekton-pipelines.yaml
    ```

4. Deploy the `pipelinetrigger.yaml` to trigger the pipeline:

    ```shell
    kubectl apply -f examples/create-pipeline-on-pr/pipelinetrigger.yaml
    ```

The PipelineRun will be created automatically, when new pull requests are detected.

## Advanced Usage

### Managed Credentials

#### Overview

The **ManagedCredential** feature simplifies credential management for Tekton pipelines by automatically creating and managing Kubernetes Secrets and ServiceAccounts with proper Tekton annotations.

**Benefits:**
- Automatic conversion of raw tokens to Tekton-compatible Secrets
- Proper annotation handling for different credential types (Git, Docker Registry, Cloud Provider)
- Automatic ServiceAccount creation and linking
- Simplified credential rotation and management

**ManagedCredential Specification:**

```yaml
apiVersion: v1
kind: ManagedCredential
metadata:
  name: github-credential
  namespace: jq-example-namespace
spec:
  # Type of credential: git, docker-registry, cloud-provider, or generic
  credentialType: git
  # Provider for Tekton annotations (e.g., github.com, gitlab.com, docker.io)
  provider: github.com
  # Reference to the Secret containing the raw token
  tokenSecretRef:
    name: github-token-secret
    namespace: jq-example-namespace
  # Optional: Human-readable description
  description: "GitHub credentials for CI/CD pipelines"
  # Optional: Customize the generated Secret
  secretTemplate:
    labels:
      app: ci-cd
    annotations:
      custom: annotation
  # Optional: Customize the generated ServiceAccount
  serviceAccountTemplate:
    labels:
      app: ci-cd
```

#### Example: Using ManagedCredential with PipelineTrigger

1. Create a Secret containing your raw token:

```shell
kubectl create secret generic github-token-secret \
  --from-literal=token=ghp_your_github_token_here \
  -n jq-example-namespace
```

2. Create a ManagedCredential:

```yaml
apiVersion: v1
kind: ManagedCredential
metadata:
  name: github-credential
  namespace: jq-example-namespace
spec:
  credentialType: git
  provider: github.com
  tokenSecretRef:
    name: github-token-secret
    namespace: jq-example-namespace
  description: "GitHub credentials for pipeline authentication"
```

3. Reference the ManagedCredential in your PipelineTrigger:

```yaml
apiVersion: pipeline.jquad.rocks/v1alpha1
kind: PipelineTrigger
metadata:
  name: pipelinetrigger-with-credentials
  namespace: jq-example-namespace
spec:
  source:
    apiVersion: source.toolkit.fluxcd.io/v1
    kind: GitRepository
    name: my-repo
  managedCredentialRef:
    apiVersion: v1
    kind: ManagedCredential
    name: github-credential
    namespace: jq-example-namespace
  pipelineRun:
    apiVersion: tekton.dev/v1
    kind: PipelineRun
    metadata:
      generateName: authenticated-pipeline-
    spec:
      pipelineRef:
        name: build-pipeline
      params:
      - name: repo-url
        value: "https://github.com/my-org/my-repo.git"
```

The operator will automatically:
- Apply the ServiceAccount from the ManagedCredential to the PipelineRun
- Add the credential Secret as a workspace to the PipelineRun
- Handle proper Tekton annotations for authentication

### Dynamic Input Parameters

The operator accepts JSON path expressions for dynamic input parameters, enabling you to extract and use specific data from events.

#### Example: Starting a PipelineRun on Events from PullRequest

Given the following Github response:

```json
{"id":1431803478,"number":8,"state":"open","locked":false,"title":"simple pr","created_at":"2023-07-12T18:37:32Z"}
```

You can use the expression `$.title` in a `PipelineTrigger` to extract the title like this:

```yaml
params:
- name: "pullrequest-title"
  value: $.title
```

The whole Github/Bitbucket response can be shown, by using `kubectl describe pullrequest`.

#### Example: Starting a PipelineRun on Events from GitRepository

You can use the following JSON expressions in the arguments of a `PipelineRun`:

`$.branchName` - The current branch name

`$.branchId` - The current commit ID

`$.repositoryName` - The current repository name

#### Example: Starting a PipelineRun on Events from ImagePolicy

You can use the following JSON expressions in the arguments of a `PipelineRun`:

`$.imageName` - The current image name

`$.imageVersion` - The current image version

`$.repositoryName` - The current repository name

### Dual Cluster Deployment

If you plan to deploy the operator in a dual-cluster setup and wish to evenly distribute `PipelineRuns` between the clusters, you can achieve this by configuring the Controller `Deployment` with the following arguments:

```yaml
      containers:
      - name: manager
        command:
        - /manager
        args:
        - --leader-elect
        - --second-cluster
        - --second-cluster-address=xxx
        - --second-cluster-bearer-token=xxx     
```

In this dual-cluster configuration, the operator will efficiently manage and distribute `PipelineRuns` across the specified clusters.

## Development and Contribution

### Build and run locally 

In order to build the project, run `make`.

Run the tests with `make test`.

If the API source files change, run `make manifests`.

Run the project locally by `make deploy`.

### Build the container image 

Build the container image using `docker build . --tag pipeline-trigger-operator` (implicitly builds the operator)

Build and deploy the changed version by running the following command from the `config/default` directory:

`kustomize build . | kubectl apply -f -` 