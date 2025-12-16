IMG_REGISTRY ?= localcruipath.azurecr.io
IMG_REPOSITORY ?= certificate-initializer-injector
IMG_TAG ?= latest


# Image URL to use all building/pushing image targets
IMG = ${IMG_REGISTRY}/${IMG_REPOSITORY}:${IMG_TAG}

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

docker-build-rhel-trustor:
	docker build cert-trustor/rhel -t ${IMG} && docker push ${IMG}

docker-build-debian-trustor:
	docker build cert-trustor/debian -t ${IMG} && docker push ${IMG}
