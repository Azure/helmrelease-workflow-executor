
# Image URL to use all building/pushing image targets
IMG ?= azureorkestra/executor:latest

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: build 

# Run tests
test: fmt vet 
	go test ./... -coverprofile cover.out

# Build manager binary
build: fmt vet
	go build -o bin/executor main.go

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}
