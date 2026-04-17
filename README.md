<div align="center">
  <img src="frontend/public/favicon.svg" width="100" height="100" alt="Lumos" />
  <h1>Lumos</h1>
  <p>Kubernetes-native configuration management — sync external configs into ConfigMaps, declaratively.</p>
</div>

---

Lumos is a Kubernetes operator that pulls configuration from **Git repositories** and keeps your ConfigMaps in sync. Define what you want with two CRDs; Lumos handles the rest.

## Features

- **Git-native**: Git (HTTPS/SSH) as the configuration source
- **Namespace & cluster scoped**: `ConfigStore` (namespaced) and `ClusterConfigStore` (cluster-wide)
- **Flexible format**: `Raw` for full-file storage, `Env` for flat key/value parsing
- **Encrypted secrets**: SOPS integration for sensitive data
- **Automatic refresh**: configurable `refreshInterval` per `ExternalConfig`
- **Web dashboard**: built-in React UI for visibility into sync status and config stores

## How It Works

```
ConfigStore ──► provider (Git)
     │
     └── ExternalConfig ──► ConfigMap
             (refreshInterval, data mappings)
```

1. Create a `ConfigStore` pointing at your Git repo.
2. Create an `ExternalConfig` referencing that store, listing which files/keys to sync.
3. Lumos creates and keeps a `ConfigMap` up to date on every `refreshInterval`.

## Quick Start

### 1. Install CRDs

```bash
make install
```

### 2. Run the operator

```bash
make run
```

### 3. Create a ConfigStore

```yaml
apiVersion: lumos.io/v1alpha1
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
      name: git-credentials   # Secret with keys: username, password (or token)
```

### 4. Sync configuration with ExternalConfig

```yaml
apiVersion: lumos.io/v1alpha1
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
    - source: config/app.yaml   # path in the Git repo
      key: app.yaml             # key in the resulting ConfigMap
      format: Raw
    - source: config/env.json   # parsed as flat key/value pairs
      format: Env
  target:
    name: app-configmap         # defaults to ExternalConfig name if omitted
```

The resulting `ConfigMap` is created (or updated) in the same namespace. Check sync status:

```bash
kubectl get externalconfig app-config -o wide
# NAME          STORE           READY   SYNCED AT              VERSION
# app-config    my-git-store    True    2026-04-17T10:00:00Z   a1b2c3d
```

## API Reference

### ConfigStore / ClusterConfigStore

| Field | Description |
|---|---|
| `spec.provider` | `Git` |
| `spec.git.url` | HTTPS or SSH repo URL |
| `spec.git.branch` | Branch to track (default: `main`) |
| `spec.git.secretRef` | Secret with `username`+`password`/`token` or `sshPrivateKey` |

### ExternalConfig

| Field | Description |
|---|---|
| `spec.storeRef.name` | ConfigStore or ClusterConfigStore name |
| `spec.storeRef.kind` | `ConfigStore` (default) or `ClusterConfigStore` |
| `spec.refreshInterval` | How often to re-sync, e.g. `5m`, `1h` |
| `spec.data[].source` | File path in the Git repo |
| `spec.data[].key` | ConfigMap key name (required for `Raw` format) |
| `spec.data[].format` | `Raw` (default) or `Env` |
| `spec.target.name` | ConfigMap name (defaults to ExternalConfig name) |

## Encrypted Secrets (SOPS + Age)

Lumos can decrypt SOPS-encrypted files from Git and write the plaintext key-value pairs into a Kubernetes `Secret`.

### How it works

```
ConfigStore (Git) ──► EncryptedSecret ──► K8s Secret
                           │
                    age key (K8s Secret)
```

1. Store your SOPS-encrypted files (`.yaml`, `.json`, `.env`, `.ini`) in Git.
2. Create a K8s Secret containing your age private key under the `keys.txt` data key.
3. Create an `EncryptedSecret` referencing the store, the age key, and the files to decrypt.

### 1. Encrypt a file with SOPS + Age

```bash
# Generate an age key pair
age-keygen -o age.agekey

# Encrypt a secret file
sops --encrypt --age $(grep "public key" age.agekey | awk '{print $NF}') secrets/app.yaml > secrets/app.enc.yaml

# Commit the encrypted file
git add secrets/app.enc.yaml && git commit -m "add encrypted secrets"
```

### 2. Store the age private key in Kubernetes

```bash
kubectl create secret generic age-key \
  --from-file=keys.txt=age.agekey
```

### 3. Create an EncryptedSecret

```yaml
apiVersion: lumos.io/v1alpha1
kind: EncryptedSecret
metadata:
  name: app-secrets
  namespace: default
spec:
  storeRef:
    name: my-git-store
    kind: ConfigStore
  ageKeyRef:
    name: age-key           # K8s Secret containing keys.txt
  refreshInterval: 5m
  data:
    - source: secrets/app.enc.yaml
  target:
    name: app-secret        # K8s Secret to create (defaults to EncryptedSecret name)
```

Lumos decrypts each file and merges all top-level keys into the target `Secret`.

```bash
kubectl get encryptedsecret app-secrets -o wide
# NAME          STORE           TARGET       READY   SYNCED AT              VERSION
# app-secrets   my-git-store    app-secret   True    2026-04-17T10:00:00Z   a1b2c3d
```

### EncryptedSecret API Reference

| Field | Description |
|---|---|
| `spec.storeRef.name` | ConfigStore or ClusterConfigStore name |
| `spec.storeRef.kind` | `ConfigStore` (default) or `ClusterConfigStore` |
| `spec.ageKeyRef.name` | K8s Secret containing the age private key under `keys.txt` |
| `spec.refreshInterval` | How often to re-sync, e.g. `5m`, `1h` (default: `5m`) |
| `spec.data[].source` | Path to a SOPS-encrypted file in the Git repo |
| `spec.target.name` | K8s Secret name to write decrypted data into (defaults to `EncryptedSecret` name) |

## Dashboard

Lumos ships a web dashboard for inspecting config stores and sync state.

```bash
# Start the API server + dashboard
make run
# Open http://localhost:8080
```

## Tech Stack

| Layer | Technology |
|---|---|
| Operator | Go, Kubebuilder v4, controller-runtime |
| Providers | go-git |
| Encryption | SOPS |
| Frontend | React 19, TypeScript, Vite, TailwindCSS, Radix UI |
| Testing | Ginkgo v2, Gomega |

## Development

```bash
# Generate CRD manifests and deepcopy
make generate manifests

# Run tests
make test

# Build the Docker image
make docker-build IMG=lumos:dev
```

## License

Apache License 2.0
