ARCH := $(shell uname -m | sed 's/x86_64/amd64/' )
OS_NAME := $(shell uname -s | tr '[:upper:]' '[:lower:]')

IMG_REGISTRY ?= localcruipath.azurecr.io
IMG_REPOSITORY ?= admctl-webhook
IMG_TAG ?= latest


# Image URL to use all building/pushing image targets
IMG = ${IMG_REGISTRY}/${IMG_REPOSITORY}:${IMG_TAG}

$(KIND):
	mkdir -p $(KIND)

LOCALBIN ?= $(shell pwd)/bin
KIND ?= $(LOCALBIN)/kind

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: build

# Build manager binary
build:
	go build -o bin/webhook main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: build
	go run ./main.go

# Build the docker image
docker-build:
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	if docker manifest inspect ${IMG}; then \
		echo "Docker image ${IMG} already exists in the registry."; \
	else \
		echo "Docker image ${IMG} not found in the registry. Pushing the image..."; \
		docker push ${IMG}; \
	fi

kind-setup:
	mkdir -p $(LOCALBIN)
	curl -Lo $(KIND) https://kind.sigs.k8s.io/dl/v0.17.0/kind-$(OS_NAME)-$(ARCH)
	chmod +x $(KIND)
	$(KIND) delete cluster --name kind-webhook || true
	$(KIND) create cluster --name kind-webhook --image=kindest/node:v1.25.3
	$(KIND) get kubeconfig --name kind-webhook > ${HOME}/.kube/config

test-setup:
	kubectl create namespace cert-manager
	kubectl label namespace cert-manager cert-manager.io/disable-validation=true
	kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.7.1/cert-manager.yaml --validate=false
	kubectl get pods --namespace cert-manager
	sleep 60
	helm install whtest "tests/whhelm"  --set imagetag=${IMAGETAG} --debug
	export KUBECONFIG=${HOME}/.kube/config

run-test:
	export KUBECONFIG=${HOME}/.kube/config; \
	go run  -v ./tests/main.go

test: kind-setup test-setup run-test
