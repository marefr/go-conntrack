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

.PHONY: replace-add replace-drop deps lint test
