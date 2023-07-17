REGISTRY ?= pkbhowmick
IMAGE ?= agent-poc
VERSION ?= v0.0.1

deploy-to-kind:
	docker build -t $(REGISTRY)/$(IMAGE):$(VERSION) .
	kind load docker-image $(REGISTRY)/$(IMAGE):$(VERSION)
