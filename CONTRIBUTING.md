# Contributing to Spanner Omni Autoscaler

Thank you for your interest in contributing to the **Google Cloud Spanner Omni Autoscaler** project! We welcome bug fixes, documentation improvements, Helm chart updates, and multi-cloud platform support.

---

## 🛠️ Development Setup & Prerequisites

1. **Clone the repository**:
   ```bash
   git clone https://github.com/cloud-gtm/spanner-omni-autoscaler.git
   cd spanner-omni-autoscaler
   ```

2. **Required Tooling**:
   - [Go](https://go.dev/doc/install) (version >= 1.22)
   - [Kubernetes CLI (`kubectl`)](https://kubernetes.io/docs/tasks/tools/)
   - [Helm CLI](https://helm.sh/docs/intro/install/) (version >= 3.12.0)
   - [Terraform CLI](https://developer.hashicorp.com/terraform/downloads) (version >= 1.5.0)

3. **Run Unit Tests**:
   ```bash
   go test -v ./pkg/scaler/... ./pkg/prometheus/... ./pkg/spanner/... ./pkg/poller/...
   ```

4. **Lint Helm Chart**:
   ```bash
   helm lint helm/spanner-omni-autoscaler/
   ```

5. **Format & Validate Terraform**:
   ```bash
   cd terraform && terraform fmt -check -recursive
   ```

---

## 📝 Code Style & Guidelines

- **Go**: Follow official Go guidelines (`gofmt`, `go vet`). All exported functions and structs must have descriptive comments.
- **Helm**: Ensure templates pass `helm lint` and avoid unescaped Go template tokens when embedding Prometheus alert rules.
- **Terraform**: Follow standard HashiCorp HCL style guidelines. Run `terraform fmt -recursive` before committing.
- **Documentation**: Keep `README.md`, `DESIGN.md`, and `DEMO_GUIDE.md` updated in synchronization whenever architecture or CLI workflows change.
- **Security & Safety**: Always protect Spanner Paxos quorums (root servers) and TrueTime health in scaling calculations.

---

## 🧪 Submitting a Pull Request

1. Create a new topic branch:
   ```bash
   git checkout -b feature/my-enhancement
   ```
2. Make your modifications, format code, and ensure all tests pass.
3. Commit your changes using clear, conventional commit messages (e.g. `feat: add stepwise scaling algorithm`, `docs: update DEMO_GUIDE with EKS steps`).
4. Push to your branch and open a Pull Request against `main`.

---

## 📄 License

By contributing to this repository, you agree that your contributions will be licensed under the **Apache License, Version 2.0**.
