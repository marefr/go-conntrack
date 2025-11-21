replace-add:
	go mod edit -replace github.com/marefr/go-conntrack/v2=../../
	go mod tidy

replace-drop:
	go mod edit -dropreplace=github.com/marefr/go-conntrack/v2
	go mod tidy

deps:
	go mod tidy

lint:
	golangci-lint run --config=../../.golangci.yml

test:
	go test ./...

test-race:
# Skipping race detector due to https://github.com/golang/go/issues/76402
	go test ./...

.PHONY: replace-add replace-drop deps lint test test-race
