# AresOperator - a multi-framework unified kubernetes operator
This repository implements a universal kubernetes operator for multiple AI training frameworks. It privides several machenisms such as state deducing and roles dependency handling, aiming at the general orchestration capability for AI workloads. See [AresJob CRD definition](https://github.com/AresSys/AresOperator/blob/main/config/crd/bases/ares.io_aresjobs.yaml).

The individually adapted frameworks are as follows.
- Tensorflow
- Pytorch
- MPI (for frameworks which support communiation via MPI, such as Horovod and Pytorch)
- Bagua
- Dask

We also provide the `custom` framework for not listed frameworks, and you can customize roles dependency and state deducing as required.

See [samples](https://github.com/AresSys/AresOperator/tree/main/config/samples) for all frameworks.

### Prerequisites
- Kubernetes
- Kubectl

### Installation
#### Deploy the operator

Install Operator on an existing Kubernetes cluster.
```shell
git clone https://github.com/AresSys/AresOperator.git
cd AresOperator

# install crd
kubectl apply -f build/crd/.

# observer
kubectl apply -f build/observer.yaml

# operator
kubectl apply -f build/operator.yaml
```

Verify ares-operator is running
```yaml
kubectl -n ares get pods

NAME                             READY   STATUS    RESTARTS   AGE
ares-observer-777b5c9cb7-59tv6   0/1     Running   0          17s
ares-observer-777b5c9cb7-qqfdn   0/1     Running   0          17s
ares-operator-55499bcb4c-sprq9   0/1     Running   0          10s
```

### Examples

- MPI
```shell
# replace ${IMAGE} with a MPI image
kubectl apply -f config/samples/mpi-sample.yaml
```
Verify pods startup dependency
```yaml
kubectl get pods

NAME                            READY   STATUS              RESTARTS   AGE
mpi-sample-launcher-0           0/1     Init:0/1            0          18s
mpi-sample-worker-0             0/1     ContainerCreating   0          18s
```

NOTE: See [AresJobInitializer](https://github.com/AresSys/AresJobInitializer) for the implementation of the initializer for AresOperator.