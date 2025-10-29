# Changelog

All notable changes to the Pipeline Trigger Operator will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- **BREAKING**: Upgraded Flux Image Automation APIs from `v1beta2` to `v1` (GA release)
- Updated `github.com/fluxcd/image-reflector-controller/api` from v0.29.1 to v1.0.3
- Updated `github.com/fluxcd/source-controller/api` from v1.0.1 to v1.7.3
- Updated `github.com/fluxcd/pkg/apis/meta` from v1.1.1 to v1.22.0
- Upgraded Go toolchain from 1.24.0 to 1.25.0
- Updated all example YAML files to use `image.toolkit.fluxcd.io/v1` API version
- Updated Flux CRDs to latest versions from Flux v2.7

### Fixed
- **CRITICAL**: Fixed panic when using Tekton resolver-based pipeline references (bundles, git, cluster)
- Updated ImagePolicy field access to use new `latestRef` structure instead of deprecated `latestImage`
- Updated Artifact type references to use `meta.Artifact` with required `digest` field
- Fixed all unit tests to work with new API structures
- Added proper nil checks in `existsPipelineResource` to prevent interface conversion panics

### Bug Fixes Detail
**Tekton Resolver Panic Fix:**
The operator previously assumed all pipeline references used the direct `pipelineRef.name` format and would panic with "interface conversion: interface {} is nil, not string" when encountering resolver-based references like:
```yaml
pipelineRef:
  resolver: bundles
  params:
  - name: bundle
    value: ghcr.io/example/pipeline:v1.0.0
```

The fix adds proper nil checking and skips validation for resolver-based references, allowing them to be validated by Tekton instead.

### Migration Notes
- ImagePolicy resources must be updated from `apiVersion: image.toolkit.fluxcd.io/v1beta2` to `v1`
- ImageRepository resources must be updated from `apiVersion: image.toolkit.fluxcd.io/v1beta2` to `v1`
- The ImagePolicy status now uses `latestRef.name` and `latestRef.tag` instead of `latestImage`
- **No action required** for the resolver panic fix - it's automatically resolved

### Compatibility
- Requires Flux v2.7 or later
- Compatible with Kubernetes 1.27+
- Tekton Pipelines v1+

---

