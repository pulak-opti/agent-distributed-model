REGISTRY ?= pkbhowmick
IMAGE ?= agent
VERSION ?= v0.0.1

deploy-to-kind:
	docker build -t $(REGISTRY)/$(IMAGE):$(VERSION) .
	kind load docker-image $(REGISTRY)/$(IMAGE):$(VERSION)
