# Setup

The Eventing Manager follows the [Kubernetes operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) and is scaffolded using [Kubebuilder](https://book.kubebuilder.io/).

Projects created by Kubebuilder contain a Makefile with tooling that must be locally executed before pushing (i.e. `git push`) to the repository. Refer to [Governance](./governance.md) for further details.

## Prerequisites

- [Go](https://go.dev/)
- [Docker](https://www.docker.com/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [Kubebuilder](https://book.kubebuilder.io/)
- [kustomize](https://kustomize.io/)
- [golangci-lint](https://golangci-lint.run/)

## Available Commands
Commands are available for easier development and installation of the Eventing Manager.
To find out which commands are available and for some more details about each command, run:

```bash
make help
```
