# Kubernetes Admin CLI

A command-line tool for managing Kubernetes permissions, service accounts, and administrative tasks.

## Features

- Service Account Management
  - Create/Delete service accounts
  - List service accounts
  - Manage service account permissions
- Role and RoleBinding Management
  - Create/Delete roles and role bindings
  - List and describe roles
  - Manage role permissions
- Cluster Health Checks
  - Check node status
  - View pod distributions
  - Resource utilization

## Installation

1. Ensure you have Go 1.21 or later installed
2. Clone this repository
3. Run `go build -o k8s-admin`

## Usage

```bash
# List all service accounts
./k8s-admin sa list

# Create a new service account
./k8s-admin sa create --name my-service-account --namespace default

# Create a role
./k8s-admin role create --name pod-reader --verbs get,list,watch --resources pods

# Bind role to service account
./k8s-admin rolebinding create --name pod-reader-binding --role pod-reader --serviceaccount default:my-service-account
```

## Configuration

The tool uses your kubeconfig file located at `~/.kube/config` by default. You can specify a different config file using the `--kubeconfig` flag.

## License

MIT License
# k8-tui
