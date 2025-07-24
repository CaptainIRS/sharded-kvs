<h1 align="center">Sharded Key-Value Store</h1>
<p align="center">
    A Model Sharded Key-Value Store Implementation using Go and Kubernetes<br><br>
    <img width="708" alt="Screenshot 2024-07-29 at 12 07 53 PM" src="https://github.com/user-attachments/assets/5c290093-17d0-4560-97f8-03e629d65f5a">
</p>

## Features
* **Sharding**: The key-value store is sharded across multiple nodes using [Consistent Hashing with Bounded Loads](https://research.google/blog/consistent-hashing-with-bounded-loads/).
* **Replication**: The key-value store supports replication across multiple replicas for each shard. The number of replicas can be configured (default: 3). It uses the [Raft Consensus Algorithm](https://raft.github.io/).
* **Load Balancing**: The key-value store uses a round-robin load balancer to distribute the requests across the replicas using the [NGINX Ingress Controller](https://kubernetes.github.io/ingress-nginx/). Leader forwarding is used to forward the requests to the leader replica.
* **Scalability**: The key-value store can be deployed on Kubernetes with a configurable number of shards and replicas and can be changed dynamically using the [Helm](https://helm.sh/) values file. (Run `make deploy` to apply the changes after updating the [helmfile.yaml](./helmfile.yaml) file)
* **Upgradability**: The key-value store can be upgraded without any downtime using the [RollingUpdate](https://kubernetes.io/docs/tutorials/kubernetes-basics/update/update-intro/) strategy in Kubernetes. (Run `make sync` after updating the source code).
* **Fault Simulation**: The system can simulate network partitions and shard failures using [Chaos Mesh](https://chaos-mesh.org/). Visit http://chaos-dashboard.svc.localho.st/chaos-mesh/ to access the Chaos Mesh dashboard and experiment with different fault scenarios.
* **Tracing**: The system provides logs of traffic among all the shards logged to the standard output by capturing packets using eBPF. The logs can be viewed using the `stern` tool. (Run `stern -l group=shard-0 -n=kvs -t=short` to view the logs of the shard-0 replica group).
* **Portability**: The system can be run in any platform (Windows, macOS, Linux) with no changes to the codebase since it is built using Go and Kubernetes.


## Prerequisites (development)
1. Install [Go](https://go.dev/doc/install). Make sure to set the PATH environment variable correctly.
2. Install the [Protobuf Compiler](https://grpc.io/docs/protoc-installation).
3. Install the Go Protobuf Compiler plugins:
    ```bash
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
    ```

## Prerequisites (deployment)
1. Install [Docker](https://docs.docker.com/get-docker/).
2. Install [Docker Buildx](https://github.com/docker/buildx?tab=readme-ov-file#installing).
3. Install [MiniKube](https://minikube.sigs.k8s.io/docs/start/).
4. Install [Kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl).
5. Install [Helmfile](https://helmfile.readthedocs.io/en/latest/#installation).
6. Install `stern`:
    ```bash
    go install github.com/stern/stern@latest
    ```

## Repository Structure

- `cmd/`: Contains the code for CLI binaries that users can use to interact with the key-value store.
- `internal/`: Contains the core implementation of the key-value store.
  - `internal/protos/`: Contains the generated Go code for the Protobuf messages and services.
    - `fsm.proto`: Contains the proto definition for the Raft state machine.
    - `kv.proto`: Contains the proto definition for the key-value store service through which clients can interact with the key-value store.
    - `shard.proto`: Contains the proto definition for the shard service through which replica groups can interact with each other (for forwarding request to the shard containing the required shard).
    - `replica.proto`: Contains the proto definition for the replica service through which replicas can interact with each other (for leader forwarding).
  - `internals/raft`: Contains the implementation of the Raft state machine.
  - `internals/capture`: Contains the code for tracing network requests by capturing with eBPF.
- `protos/`: Contains the Protobuf definitions for the messages and services used in the key-value store.

## Usage

### Makefile Targets

Run `make "target"` where `"target"` is one of the following:
- `deploy`: Deploy the system (Key-Value Store Server) in Kubernetes.
- `client`: Run the client.
- `clean`: Remove the system from Kubernetes.
- `sync`: Sync any changes in the system to Kubernetes.
- `dashboard`: Open the Kubernetes dashboard.
- `proto`: Generate the Go code from the Protobuf definitions.
- `fmt`: Format the Go code and helm templates before committing.

### Viewing Logs/Traces

* To view all logs of the cluster:
    ```console
    $ stern -n kvs -t=short
    ```
* To view logs of a specific replica group (e.g., shard-0):
    ```console
    $ stern -l group=shard-0 -n kvs -t=short
    ```
* To view logs of a specific replica (e.g., shard-0-replica-0):
    ```console
    $ stern -l shard-0-replica-0 -n kvs -t=short
    ```
* To remove logs of heartbeat messages add `-e="Sending heartbeat"` to the command:
    ```console
    $ stern -n kvs -t=short -e="Sending heartbeat"
    ```

## System Architecture

![sharded-kvs drawio](https://github.com/user-attachments/assets/f60bd480-27a9-426f-9de5-a199a26da5ed)

## Kubernetes Architecture

```mermaid
graph LR;
 client([client]).->ingress[Ingress];
 ingress-->|routing rule|service0[Service<br>LB];
 ingress-->|routing rule|service1[Service<br>LB];
 ingress-->|routing rule|service2[Service<br>LB];
 subgraph shard0[shard-0 StatefulSet]
 service0-->pod00[Pod<br>replica-0];
 service0-->pod01[Pod<br>replica-1];
 service0-->pod02[Pod<br>replica-2];
 end
 subgraph shard1[shard-1 StatefulSet]
 service1-->pod10[Pod<br>replica-0];
 service1-->pod11[Pod<br>replica-1];
 service1-->pod12[Pod<br>replica-2];
 end
 subgraph shard2[shard-2 StatefulSet]
 service2-->pod20[Pod<br>replica-0];
 service2-->pod21[Pod<br>replica-1];
 service2-->pod22[Pod<br>replica-2];
 end
 classDef plain fill:#ddd,stroke:#fff,stroke-width:4px,color:#000;
 classDef k8s fill:#326ce5,stroke:#fff,stroke-width:4px,color:#fff;
 classDef cluster fill:#fff,stroke:#bbb,stroke-width:2px,color:#326ce5;
 class ingress,service0,ingress1,service1,ingress2,service2,pod01,pod02,pod00,pod11,pod12,pod10,pod21,pod22,pod20 k8s;
 class client plain;
 class shard0,shard1,shard2 cluster;
```
