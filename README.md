# StatefulSet Admission Webhook
This project implements a Kubernetes admission webhook that receives StatefulSet update events. If the volume claim template PVC size of a StatefulSet is increased (by comparing against the last-applied-configuration), the admission webhook will perform an orphan delete operation on the Kubernetes StatefulSet.

## Introduction
The StatefulSet Admission Webhook is a Kubernetes admission controller designed to enforce specific policies on StatefulSet updates. It ensures that volume claim template PVC size increases are handled properly by performing an orphan delete operation on affected StatefulSets.

## Installation
### Prerequisites
- Kubernetes cluster (version X.X.X or later)
- kubectl configured to access the Kubernetes cluster

### Steps
1. Clone this repository:
   ```console
   git clone <repository-url>
   ```
1. Deploy the admission webhook to your Kubernetes cluster:
   ```console
   kubectl apply -f deployment.yaml
   ```
1. Verify that the admission webhook is running:
   ```console
   kubectl get pods -n <namespace>
   ```

### Usage
1. Make changes to the volume claim template PVC size in a StatefulSet.
1. Check the logs of the admission webhook to see if the orphan delete operation was triggered:
   ```console
   kubectl logs <admission-webhook-pod> -n <namespace>
   ```

### Configuration
- NAMESPACE: The namespace where the admission webhook is deployed.
- DEPLOYMENT_NAME: The name of the deployment for the admission webhook.
- WEBHOOK_SERVICE_NAME: The name of the service exposing the webhook.
- WEBHOOK_SERVICE_PORT: The port on which the webhook service listens.
- CA_BUNDLE: The CA bundle is used to verify client certificates.

## License
This project is licensed under the Apache-2.0 License.
