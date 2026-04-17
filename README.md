<div align="center">
  <img src="frontend/public/favicon.svg" width="100" height="100" alt="Lumos" />
  <h1>Lumos</h1>
  <p>Kubernetes-native configuration management for syncing Git-backed config into ConfigMaps and Secrets.</p>
</div>

---

Lumos is a Kubernetes operator that pulls configuration from Git repositories and keeps Kubernetes resources in sync declaratively.

It currently supports:

- `ConfigStore` and `ClusterConfigStore` for connecting to Git repositories
- `ExternalConfig` for syncing files into `ConfigMap`
- `EncryptedSecret` for decrypting SOPS-encrypted files and syncing them into `Secret`
- A built-in dashboard and API for inspecting stores and sync status

## Features

- Git as the source of truth over HTTPS or SSH
- Namespace-scoped and cluster-scoped stores
- `Raw` and `Env` data mapping modes
- SOPS + Age decryption for encrypted secrets
- Configurable per-resource refresh intervals
- Built-in dashboard for operators and developers

## Architecture

```text
ConfigStore / ClusterConfigStore
              |
              v
         Git provider
          /       \
         v         v
ExternalConfig  EncryptedSecret
      |               |
      v               v
 ConfigMap         Secret
```

## Quick Start

### 1. Install the CRDs

```bash
make install
```

### 2. Run Lumos locally

```bash
make run
```

The dashboard API and embedded frontend are served at `http://localhost:8090`.

### 3. Create a Git-backed store

```yaml
apiVersion: sync.lumos.io/v1alpha1
kind: ConfigStore
metadata:
  name: my-git-store
  namespace: default
spec:
  provider: Git
  git:
    url: https://github.com/my-org/config-repo
    branch: main
    secretRef:
      name: git-credentials
```

Credential secret example:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: git-credentials
  namespace: default
type: Opaque
stringData:
  username: git
  password: <token-or-password>
```

### 4. Sync a ConfigMap with ExternalConfig

```yaml
apiVersion: sync.lumos.io/v1alpha1
kind: ExternalConfig
metadata:
  name: app-config
  namespace: default
spec:
  storeRef:
    name: my-git-store
    kind: ConfigStore
  refreshInterval: 5m
  data:
    - source: config/app.yaml
      key: app.yaml
      format: Raw
    - source: config/env.json
      format: Env
  target:
    name: app-configmap
```

Check sync status:

```bash
kubectl get externalconfig app-config -n default -o wide
```

### 5. Sync a Secret with EncryptedSecret

```yaml
apiVersion: sync.lumos.io/v1alpha1
kind: EncryptedSecret
metadata:
  name: app-secrets
  namespace: default
spec:
  storeRef:
    name: my-git-store
    kind: ConfigStore
  ageKeyRef:
    name: age-key
  refreshInterval: 5m
  data:
    - source: secrets/app.enc.yaml
  target:
    name: app-secret
```

Lumos reads the SOPS-encrypted file from Git, decrypts it with the Age private key stored in Kubernetes, and writes the merged top-level keys into the target `Secret`.

## Example Manifests

Ready-to-apply examples live in [`examples/`](examples/).

Suggested apply order:

```bash
kubectl apply -f examples/git-credentials-secret.example.yaml
kubectl apply -f examples/configstore.yaml
kubectl apply -f examples/externalconfig.yaml
```

For encrypted secrets:

```bash
kubectl apply -f examples/age-key-secret.example.yaml
kubectl apply -f examples/encryptedsecret.yaml
```

## API Reference

### ConfigStore / ClusterConfigStore

| Field | Description |
|---|---|
| `spec.provider` | Currently `Git` |
| `spec.git.url` | HTTPS or SSH repository URL |
| `spec.git.branch` | Branch to track, defaults to `main` |
| `spec.git.secretRef` | Secret with `username` + `password`/`token`, or `sshPrivateKey` |

### ExternalConfig

| Field | Description |
|---|---|
| `spec.storeRef.name` | Referenced `ConfigStore` or `ClusterConfigStore` name |
| `spec.storeRef.kind` | `ConfigStore` or `ClusterConfigStore`, defaults to `ConfigStore` |
| `spec.refreshInterval` | Re-sync interval such as `5m` or `1h` |
| `spec.data[].source` | File path in the Git repository |
| `spec.data[].key` | Required when `format: Raw` |
| `spec.data[].format` | `Raw` or `Env` |
| `spec.target.name` | Output `ConfigMap` name, defaults to resource name |

### EncryptedSecret

| Field | Description |
|---|---|
| `spec.storeRef.name` | Referenced `ConfigStore` or `ClusterConfigStore` name |
| `spec.storeRef.kind` | `ConfigStore` or `ClusterConfigStore`, defaults to `ConfigStore` |
| `spec.ageKeyRef.name` | Secret containing the Age private key in `keys.txt` |
| `spec.refreshInterval` | Re-sync interval such as `5m` or `1h`, defaults to `5m` |
| `spec.data[].source` | Path to a SOPS-encrypted file in the Git repository |
| `spec.target.name` | Output `Secret` name, defaults to resource name |

## SOPS + Age Workflow

### 1. Generate an Age key pair

```bash
age-keygen -o age.agekey
```

### 2. Encrypt a file with SOPS

```bash
sops --encrypt --age $(grep "public key" age.agekey | awk '{print $NF}') secrets/app.yaml > secrets/app.enc.yaml
```

### 3. Store the private key in Kubernetes

```bash
kubectl create secret generic age-key \
  --from-file=keys.txt=age.agekey \
  -n default
```

### 4. Apply EncryptedSecret

```bash
kubectl apply -f examples/encryptedsecret.yaml
kubectl get encryptedsecret app-secrets -n default -o wide
```

## Dashboard

When Lumos is running locally:

```bash
make run
```

Open `http://localhost:8090`.

## Development

### Common commands

```bash
make generate manifests
make run
make test
make docker-build IMG=lumos:dev
make build-installer
```

### Frontend-only development

```bash
cd frontend
npm install
npm run dev
```

## Release Workflow

Build the install manifest:

```bash
make build-installer
```

This generates [`dist/install.yaml`](dist/install.yaml), which is the file you can attach to a GitHub release or publish for one-command installation:

```bash
kubectl apply -f <release-install-yaml-url>
```

## Tech Stack

| Layer | Technology |
|---|---|
| Operator | Go, Kubebuilder v4, controller-runtime |
| Providers | go-git |
| Encryption | SOPS |
| Frontend | React 19, TypeScript, Vite, Tailwind CSS, Radix UI |
| Testing | Ginkgo v2, Gomega |

## License

Apache License 2.0
