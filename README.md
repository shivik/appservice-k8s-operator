# Kubernetes Operator - AppService Controller

## Introduction to Kubernetes Operators

Kubernetes Operators are software extensions that use custom resources to manage applications and their components. They follow Kubernetes principles, notably the control loop pattern, to automate complex application management tasks.

### What is an Operator?

An operator is a method of packaging, deploying, and managing a Kubernetes application. It extends the Kubernetes API to create, configure, and manage instances of complex applications on behalf of a Kubernetes user. Operators use Custom Resource Definitions (CRDs) to represent application-specific resources and controllers to maintain the desired state.

### Key Concepts

**Custom Resource Definition (CRD)**: Extends Kubernetes API with custom resource types
**Controller**: Watches resources and reconciles actual state with desired state
**Reconciliation Loop**: Continuously ensures the current state matches the desired state
**Finalizers**: Cleanup logic executed before resource deletion
**Status Subresource**: Tracks the operational state of custom resources
**Owner References**: Establishes parent-child relationships between resources

### Operator Pattern Benefits

- Automates operational knowledge
- Reduces manual intervention
- Ensures consistency across environments
- Enables self-healing capabilities
- Simplifies complex application lifecycle management

## Project Overview

This project demonstrates a production-ready Kubernetes operator that manages `AppService` custom resources. The operator automatically creates and manages Deployments and Services based on the AppService specification.

### Architecture

```
AppService (CRD)
    ↓
Controller watches AppService resources
    ↓
Reconciliation Loop
    ↓
Creates/Updates Deployment + Service
    ↓
Updates AppService Status
```

### Features

- Custom Resource Definition for AppService
- Automated Deployment and Service creation
- Replica management and scaling
- Environment variable injection
- Resource limits configuration
- Status tracking with conditions
- Finalizer-based cleanup
- Health and readiness probes
- Leader election support
- Metrics and monitoring endpoints

## Project Structure

```
.
├── api/v1alpha1/              # API definitions
│   ├── appservice_types.go    # AppService CRD types
│   └── groupversion_info.go   # API group version
├── controllers/               # Controller implementations
│   └── appservice_controller.go
├── pkg/
│   ├── k8s/                   # Kubernetes client wrapper
│   │   └── client.go
│   ├── reconciler/            # Reconciliation utilities
│   │   └── reconciler.go
│   └── watcher/               # Event watching utilities
│       └── watcher.go
├── config/
│   ├── crd/                   # CRD manifests
│   │   └── appservice-crd.yaml
│   ├── rbac/                  # RBAC configuration
│   │   └── role.yaml
│   ├── manager/               # Operator deployment
│   │   └── deployment.yaml
│   └── samples/               # Example resources
│       └── appservice-sample.yaml
├── main.go                    # Operator entry point
├── Dockerfile                 # Container image
├── Makefile                   # Build automation
└── go.mod                     # Go dependencies
```

## Prerequisites

- Go 1.21 or higher
- Docker (for building images)
- kubectl configured with cluster access
- Kubernetes cluster (minikube, kind, or cloud provider)

## Installation

### Quick Start (Hot Run)

For a single command to install and run everything:

```bash
make hot-run
```

This command will:
1. Install the CRD
2. Setup RBAC permissions
3. Tidy Go dependencies
4. Start the operator locally

### Manual Installation

#### Step 1: Clone and Setup

```bash
cd <your-path-where-you-cloned-this-code>
go mod tidy
```

#### Step 2: Install CRD

```bash
make install
```

Verify CRD installation:
```bash
kubectl get crd appservices.example.com
```

#### Step 3: Setup RBAC

```bash
kubectl apply -f config/rbac/role.yaml
```

#### Step 4: Run Operator Locally

```bash
make run
```

Or build and run:
```bash
make build
./bin/manager
```

#### Step 5: Deploy to Cluster

Build and push image:
```bash
make docker-build
make docker-push
```

Deploy operator:
```bash
make deploy
```

Verify deployment:
```bash
kubectl get pods -l app=appservice-operator
```

## Usage

### Create AppService Resource

```bash
make sample
```

Or manually:
```bash
kubectl apply -f config/samples/appservice-sample.yaml
```

### Check AppService Status

```bash
kubectl get appservices
kubectl describe appservice nginx-app
```

### View Created Resources

```bash
kubectl get deployments
kubectl get services
kubectl get pods
```

### Scale Application

```bash
kubectl patch appservice nginx-app -p '{"spec":{"replicas":5}}' --type=merge
```

### Update Image

```bash
kubectl patch appservice nginx-app -p '{"spec":{"image":"nginx:1.22"}}' --type=merge
```

### Delete AppService

```bash
kubectl delete appservice nginx-app
```

## How It Works

### Reconciliation Loop

1. **Watch**: Controller watches AppService resources
2. **Get**: Retrieves current AppService state
3. **Compare**: Compares desired vs actual state
4. **Create/Update**: Creates or updates Deployment and Service
5. **Status Update**: Updates AppService status with current state
6. **Requeue**: Schedules next reconciliation

### Controller Logic

The controller implements the following workflow:

```go
Reconcile(request) {
    1. Fetch AppService resource
    2. Handle deletion (finalizer cleanup)
    3. Check if Deployment exists
       - If not, create Deployment
       - If exists, update if spec changed
    4. Check if Service exists
       - If not, create Service
    5. Update AppService status
       - Set phase (Running/Pending)
       - Update available replicas
       - Set conditions
    6. Requeue after 30 seconds
}
```

### Status Management

The operator tracks:
- **Phase**: Current operational phase (Initializing, Running, Pending, Failed)
- **Available Replicas**: Number of ready pods
- **Conditions**: Detailed status conditions with reasons
- **Last Reconcile Time**: Timestamp of last reconciliation

### Finalizers

Finalizers ensure proper cleanup:
```go
1. Resource marked for deletion
2. Finalizer prevents immediate deletion
3. Cleanup logic executes
4. Finalizer removed
5. Resource deleted
```

## Advanced Features

### Event Watching

The watcher package provides event handling:
```go
watcher := NewResourceWatcher(client, obj, resyncPeriod)
watcher.AddEventHandler(handler)
watcher.Start(ctx)
```

### Custom Reconciliation

The reconciler package offers utilities:
```go
reconciler := BaseReconciler{Client: client, Scheme: scheme}
result := reconciler.RequeueAfter(30 * time.Second)
```

### Kubernetes Client Wrapper

Simplified client operations:
```go
client := k8s.NewClient(kubeconfig)
deployment, err := client.GetDeployment(ctx, namespace, name)
```

## Monitoring

### Metrics Endpoint

Metrics available at `:8080/metrics`

### Health Checks

- Liveness: `:8081/healthz`
- Readiness: `:8081/readyz`

### Logs

View operator logs:
```bash
kubectl logs -f deployment/appservice-operator
```

## Development

### Run Tests

```bash
make test
```

### Format Code

```bash
make fmt
```

### Vet Code

```bash
make vet
```

### Clean Up

```bash
make clean-sample
make undeploy
make uninstall
```

## Troubleshooting

### CRD Not Found

```bash
kubectl get crd appservices.example.com
make install
```

### RBAC Permissions

```bash
kubectl get clusterrole appservice-operator-role
kubectl get clusterrolebinding appservice-operator-rolebinding
```

### Operator Not Starting

```bash
kubectl logs deployment/appservice-operator
kubectl describe deployment appservice-operator
```

### Resource Not Reconciling

```bash
kubectl describe appservice <name>
kubectl get events --sort-by='.lastTimestamp'
```

## Best Practices

1. **Idempotency**: Reconciliation logic should be idempotent
2. **Error Handling**: Always handle errors gracefully
3. **Status Updates**: Keep status subresource updated
4. **Finalizers**: Use finalizers for cleanup
5. **Owner References**: Set owner references for garbage collection
6. **Logging**: Use structured logging
7. **Metrics**: Expose relevant metrics
8. **Testing**: Write comprehensive tests

## Contributing

This is a demonstration project showcasing Kubernetes operator patterns. Key learning areas:

- Custom Resource Definitions
- Controller implementation
- Reconciliation loops
- Event handling
- Status management
- RBAC configuration
- Operator deployment

## License

MIT License