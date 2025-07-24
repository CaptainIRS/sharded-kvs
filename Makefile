BLUE := $(shell tput setaf 6)
YELLOW := $(shell tput setaf 3)
CLEAR := $(shell tput sgr0)

NAMESPACE := kvs
DOCKER_ENV := prod

.PHONY: protos dnsedit

deploy: helm_apply

client:
	go run cmd/client/client.go

fmt: 
	@echo "$(BLUE)Formatting code...$(CLEAR)"
	go fmt ./...
	@echo "$(BLUE)Formatting code...done$(CLEAR)"
	@echo "$(BLUE)Formatting helm files...$(CLEAR)"
	helmfile lint
	@echo "$(BLUE)Formatting helm files...done$(CLEAR)"

clean:
	kind delete cluster --name sharded-kvs-cluster
	rm -rf .deployment

sync: helm_sync

start_kind: .deployment/kind_start

dashboard: .deployment/kubeconfig
	k9s --kubeconfig .deployment/kubeconfig

helm_apply: build_images
	@echo "$(BLUE)Deploying helm chart...$(CLEAR)"
	kubectl create namespace $(NAMESPACE) || true
	kubectl create namespace ingress-nginx || true
	kubectl create namespace chaos-mesh || true
	helmfile apply --suppress-secrets
	@echo "$(BLUE)Deploying helm chart...done$(CLEAR)"

protos:
	@echo "$(BLUE)Generating protos...$(CLEAR)"
	protoc --go_out=internal \
    	--go-grpc_out=internal \
    	--go_opt=paths=source_relative \
    	--go-grpc_opt=paths=source_relative \
    	protos/*.proto
	@echo "$(BLUE)Generating protos...done$(CLEAR)"

dnsedit:
	KUBE_EDITOR="vim" kubectl -n kube-system edit configmaps coredns -o yaml

helm_sync: build_images
	@echo "$(BLUE)Syncing updates...$(CLEAR)"
	helmfile sync --skip-deps
	@echo "$(BLUE)Syncing updates...done$(CLEAR)"

build_images: .deployment/kind_start
	@echo "$(BLUE)Building images...$(CLEAR)"
	./build.sh $(DOCKER_ENV)
	@echo "$(BLUE)Building images...done$(CLEAR)"

.deployment/kind_start: .deployment/check_deps
	mkdir -p .deployment
	@echo "$(BLUE)Creating kind cluster...$(CLEAR)"
	kind create cluster --config deploy/kind/cluster.yaml
	@echo "$(BLUE)Creating kind cluster...done$(CLEAR)"
	touch .deployment/kind_start

.deployment/kubeconfig: .deployment/kind_start
	kind get kubeconfig --name sharded-kvs-cluster > .deployment/kubeconfig

.deployment/check_deps:
	mkdir -p .deployment
	@echo "$(BLUE)Checking dependencies...$(CLEAR)"
	kind version
	kubectl version --client=true
	k9s version
	docker --version
	docker buildx version
	helmfile version -o short
	helmfile init
	helmfile deps
	@echo "$(BLUE)Checking dependencies...done$(CLEAR)"
	touch .deployment/check_deps