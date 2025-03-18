PROVIDER_MODULES ?= $(shell find $(PWD)/providers  -name "go.mod" | xargs dirname)
MODULES          ?= $(PROVIDER_MODULES) example

GOTOOL=go tool
COPYRIGHT=$(GOTOOL) copyright

.PHONY: replace-add replace-drop lint test $(MODULES)

replace-add replace-drop lint test: $(MODULES)
$(MODULES):
	$(MAKE) -C $@ $(MAKECMDGOALS)

lint:
	golangci-lint run --config=.golangci.yml

test:
	go test ./...
