# Note: Macos requires `brew install findutils` to support xargs -i

PROVIDER_MODULES ?= $(shell find $(PWD)/providers  -name "go.mod" | xargs dirname)
MODULES          ?= $(PROVIDER_MODULES) example

GOTOOL=go tool
COPYRIGHT=$(GOTOOL) copyright

.PHONY: replace-add replace-drop lint test $(MODULES)

replace-add replace-drop lint test: $(MODULES)
$(MODULES):
	$(MAKE) -C $@ $(MAKECMDGOALS)

define require_clean_work_tree
	@git update-index -q --ignore-submodules --refresh

	@if ! git diff-files --quiet --ignore-submodules --; then \
			echo >&2 "cannot $1: you have unstaged changes."; \
			git diff-files --name-status -r --ignore-submodules -- >&2; \
			echo >&2 "Please commit or stash them."; \
			exit 1; \
	fi

	@if ! git diff-index --cached --quiet HEAD --ignore-submodules --; then \
			echo >&2 "cannot $1: your index contains uncommitted changes."; \
			git diff-index --cached --name-status -r --ignore-submodules HEAD -- >&2; \
			echo >&2 "Please commit or stash them."; \
			exit 1; \
	fi

endef

lint:
	@echo ">> ensuring copyright headers"
	@$(COPYRIGHT) $(shell go list -f "{{.Dir}}" ./... | xargs -i find "{}" -name "*.go")
	@$(call require_clean_work_tree,"set copyright headers")
	@echo ">> ensured all .go files have copyright headers"

	@echo "Running lint"
	@$(call require_clean_work_tree,"before lint")
	golangci-lint run --config=.golangci.yml
	@$(call require_clean_work_tree,"lint and format files")

test:
	go test ./...
