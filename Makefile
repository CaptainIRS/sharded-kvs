BLUE := $(shell tput setaf 6)
YELLOW := $(shell tput setaf 3)
CLEAR := $(shell tput sgr0)

NAMESPACE := kvs
DOCKER_ENV := prod

deploy: helm_apply

fmt: 
	@echo "$(BLUE)Formatting code...$(CLEAR)"
	go fmt ./...
	@echo "$(BLUE)Formatting code...done$(CLEAR)"
	@echo "$(BLUE)Formatting helm files...$(CLEAR)"
	helmfile lint
	@echo "$(BLUE)Formatting helm files...done$(CLEAR)"

clean: helm_destroy
	minikube kubectl -- delete namespace $(NAMESPACE) || true
	minikube kubectl -- delete namespace ingress-nginx || true
	minikube kubectl -- delete namespace chaos-mesh || true
	rm -rf .deployment

sync: helm_sync

start_minikube: .deployment/minikube_start

dashboard:
	minikube dashboard --url=true

helm_apply: build_images
	@echo "$(BLUE)Deploying helm chart...$(CLEAR)"
	minikube kubectl -- create namespace $(NAMESPACE) || true
	minikube kubectl -- create namespace ingress-nginx || true
	minikube kubectl -- create namespace chaos-mesh || true
	helmfile apply
	@echo "$(BLUE)Deploying helm chart...done$(CLEAR)"

helm_destroy:
	@echo "$(BLUE)Deleting helm chart...$(CLEAR)"
	helmfile destroy --skip-charts || true
	@echo "$(BLUE)Deleting helm chart...done$(CLEAR)"

helm_sync: build_images
	@echo "$(BLUE)Syncing updates...$(CLEAR)"
	helmfile sync --skip-deps
	@echo "$(BLUE)Syncing updates...done$(CLEAR)"

build_images: .deployment/minikube_start
	@echo "$(BLUE)Building images...$(CLEAR)"
	./build.sh $(DOCKER_ENV)
	@echo "$(BLUE)Building images...done$(CLEAR)"

.deployment/minikube_start: .deployment/check_deps
	mkdir -p .deployment
	@echo "$(BLUE)Starting minikube...$(CLEAR)"
	minikube start --driver docker --extra-config=apiserver.service-node-port-range=8080-8080 --dns-domain localho.st  --ports 127.0.0.1:8080:8080 --cpus 2 --memory 4096
	@echo "$(BLUE)Starting minikube...done$(CLEAR)"
	touch .deployment/minikube_start

.deployment/check_deps:
	mkdir -p .deployment
	@echo "$(BLUE)Checking dependencies...$(CLEAR)"
	minikube version
	kubectl version --client=true
	docker --version
	docker buildx version
	helmfile version -o short
	helmfile init
	helmfile deps
	@echo "$(BLUE)Checking dependencies...done$(CLEAR)"
	touch .deployment/check_deps