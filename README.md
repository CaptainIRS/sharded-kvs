# Sharded Key-Value Store

## Roadmap
- [ ] Getting Started
  - [x] Simple RPC service to serve as a starting point.
  - [ ] Implement endpoints for a key-value store in a single node.
    - [ ] Implement `Get` endpoint.
    - [ ] Implement `Put` endpoint.
    - [ ] Implement `Delete` endpoint.
- [ ] TBD

## Prerequisites
1. Install [Go](https://go.dev/doc/install). Make sure to set the PATH environment variable correctly.
2. Install the [Protobuf Compiler](https://grpc.io/docs/protoc-installation).
3. Install the Go Protobuf Compiler plugins:
    ```bash
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
    ```

## Repository Structure

- `cmd/`: Contains the code for CLI binaries that users can use to interact with the key-value store.
- `internal/`: Contains the core implementation of the key-value store.
  - `internal/protos/`: Contains the generated Go code for the Protobuf messages and services.
- `protos/`: Contains the Protobuf definitions for the messages and services used in the key-value store.

## Usage

1. Start the key-value store server:
    ```console
    $ go run cmd/server/main.go
    ```
2. Start the key-value store client:
    ```console
    $ go run cmd/client/main.go
    ```

## Generating Go Code from Protobuf Definitions

To generate the Go code from the Protobuf definitions, run the following command:
```console
$ protoc --go_out=internal \
  --go-grpc_out=internal \
  --go_opt=paths=source_relative \
  --go-grpc_opt=paths=source_relative \
  protos/*.proto
```
