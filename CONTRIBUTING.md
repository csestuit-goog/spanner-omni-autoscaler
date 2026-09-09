# Contributing to Spanner Omni on Regional GKE

Thank you for your interest in contributing to the **Spanner Omni on Regional GKE** project! We welcome bug fixes, documentation improvements, Terraform module enhancements, and operational runbook scripts.

---

## 🛠️ Development Setup & Prerequisites

1. **Clone the repository**:
   ```bash
   git clone https://github.com/csestuit-goog/spanner-omni-demo-regional.git
   cd spanner-omni-demo-regional
   ```

2. **Required Tooling**:
   - [Google Cloud SDK (`gcloud`)](https://cloud.google.com/sdk/docs/install) (authenticated with `gcloud auth login`)
   - [Terraform CLI](https://developer.hashicorp.com/terraform/downloads) (version >= 1.5.0)
   - [Kubernetes CLI (`kubectl`)](https://kubernetes.io/docs/tasks/tools/)
   - [Helm CLI](https://helm.sh/docs/intro/install/) (version >= 3.12.0)

3. **Validate Terraform Configuration**:
   ```bash
   cd terraform
   terraform fmt -check
   terraform validate
   ```

4. **Validate Shell Scripts**:
   ```bash
   shellcheck scripts/*.sh quickstart.sh
   ```

---

## 📝 Code Style & Guidelines

- **Terraform**: Follow standard HashiCorp HCL style guidelines. Run `terraform fmt -recursive` before committing.
- **Documentation**: Keep `README.md`, `DESIGN.md`, and `DEMO_GUIDE.md` updated in synchronization whenever architecture or CLI workflows change.
- **Resource Attribution**: Ensure all Cloud SDK commands include appropriate metric tags (`CLOUDSDK_METRICS_ENVIRONMENT`).
- **Data Protection**: Never commit credentials, license keys, or sensitive `.tfvars` files. Use `.gitignore` to protect state files.

---

## 🧪 Submitting a Pull Request

1. Create a new topic branch:
   ```bash
   git checkout -b feature/my-enhancement
   ```
2. Make your modifications, format code, and test locally against a development GKE cluster.
3. Commit your changes using clear, conventional commit messages (e.g. `docs: update scaling guide`, `feat: add automated backup script`).
4. Push to your branch and open a Pull Request against `main`.

---

## 📄 License

By contributing to this repository, you agree that your contributions will be licensed under the **Apache License, Version 2.0**.
