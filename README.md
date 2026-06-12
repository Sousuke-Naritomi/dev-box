# dev-box 🚀

A fast, interactive, and highly flexible CLI tool built in Go that instantly generates secure, production-ready local development environments and deployment manifests (Docker Compose, Kubernetes, Terraform).

## ✨ Features

- **Interactive Scaffolding**: Instantly set up a full-stack project (Frontend + Backend + Database) through an interactive CLI or command-line flags.
- **Supported Frameworks**:
  - **Frontend**: React (Vite), Vue (Vite)
  - **Backend**: Node.js (Express), Python (Flask), Go, PHP (Apache), Java (HttpServer)
  - **Database**: PostgreSQL, MySQL
  - **Cache**: Redis
- **Advanced Architecture & Security**:
  - Multi-stage Dockerfiles with separate `InternalPort` (for local hot-reloading) and `ImagePort` (for K8s routing).
  - Secure, unprivileged non-root containers across all languages (using standard users like `10001`, `101`).
- **Production-Ready Kubernetes**:
  - Run `dev-box k8s` to generate `Deployment`, `Service`, and `PVC` manifests.
  - Hardened by default: `seccompProfile`, dropped capabilities, `RunAsUser` non-root enforcement.
  - Zero-Trust networking via strict Ingress and Egress `NetworkPolicy` generation.
- **Cloud Infrastructure (IaC)**: Run `dev-box deploy` to generate Terraform modules for AWS, GCP, or Azure.
- **Built-in Addons**:
  - 🔄 **Nginx Reverse Proxy**: Automatic API routing for local development.
  - ☁️ **LocalStack**: AWS local testing environment.
  - 📈 **Observability**: Prometheus, Grafana, and Promtail/Loki pre-configured.
  - 🤖 **CI/CD**: GitHub Actions workflow generation for automated testing and deployments.

---

## 📦 Installation

*(Assuming you have Go installed)*

```bash
git clone https://github.com/Sousuke-Naritomi/dev-box.git
cd dev-box
go build -o dev-box .
```

Move the binary to your PATH to use it globally.

---

## 🚀 Usage

### 1. Initialize a Local Environment (`dev-box init`)

Generate your `docker-compose.yml`, scaffolding code, `.env`, and Nginx configurations. You can run this interactively or provide flags:

```bash
# Interactive mode
./dev-box init

# Non-interactive mode (CI/CD friendly)
./dev-box init --frontend react --backend go --database postgres --monitoring --cicd
```

**Helpful flags for `init`**:
- `--output-dir <dir>`: Generate files into a specific directory instead of `.`.
- `--dry-run`: Preview the generated files without writing them to disk.
- `--force`: Overwrite existing files without prompting.

### 2. Manage the Local Environment

Control your Docker Compose environment directly via the CLI:

```bash
./dev-box start    # Equivalent to docker compose up -d
./dev-box stop     # Equivalent to docker compose down
./dev-box status   # Equivalent to docker compose ps
./dev-box logs     # Equivalent to docker compose logs -f
```

### 3. Generate Kubernetes Manifests (`dev-box k8s`)

Export your selected stack into hardened Kubernetes YAMLs (creates `k8s-manifests/` by default).

```bash
./dev-box k8s --frontend react --backend go --database postgres --cache redis
```
This generates `api.yaml`, `frontend.yaml`, `db.yaml`, `cache.yaml`, `namespace.yaml`, and extremely robust `networkpolicy.yaml` definitions.

### 4. Generate Terraform Cloud Infrastructure (`dev-box deploy`)

Generate Terraform configurations to deploy your containers to the cloud.

```bash
./dev-box deploy --provider aws
```

---

## 🔒 Security Best Practices Implemented

`dev-box` does not just write "Hello World" boilerplate; it focuses heavily on enterprise security:
- **Randomized Credentials**: Automatically generates a cryptographically secure 20-character `.env` file on initialization and automatically adds it to `.gitignore`.
- **Zero-Trust K8s Network Policies**: Pods cannot communicate with each other or the internet unless explicitly allowed (e.g., API can talk to DB, but Frontend cannot talk to DB directly).
- **Distroless & Multi-stage**: Compiles backends/frontends and runs them on minimal footprints (e.g., Go on Alpine, React on `nginx-unprivileged`).
- **Filesystem Permissions**: Refuses to overwrite symbolic links and outputs Windows-specific ACL warnings for credential files.

## 🤝 Contributing
Contributions, issues, and feature requests are welcome! Feel free to check the issues page.
