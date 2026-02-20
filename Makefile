IMG ?= appservice-operator:latest
KUBECONFIG ?= ~/.kube/config

.PHONY: all
all: build

.PHONY: build
build:
	go build -o bin/manager main.go

.PHONY: run
run:
	go run main.go

.PHONY: docker-build
docker-build:
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push:
	docker push ${IMG}

.PHONY: install
install:
	kubectl apply -f config/crd/appservice-crd.yaml

.PHONY: uninstall
uninstall:
	kubectl delete -f config/crd/appservice-crd.yaml

.PHONY: deploy
deploy:
	kubectl apply -f config/rbac/role.yaml
	kubectl apply -f config/manager/deployment.yaml

.PHONY: undeploy
undeploy:
	kubectl delete -f config/manager/deployment.yaml
	kubectl delete -f config/rbac/role.yaml

.PHONY: sample
sample:
	kubectl apply -f config/samples/appservice-sample.yaml

.PHONY: clean-sample
clean-sample:
	kubectl delete -f config/samples/appservice-sample.yaml

.PHONY: test
test:
	go test ./... -v

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: tidy
tidy:
	go mod tidy