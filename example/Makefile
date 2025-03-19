replace-add:
	go mod edit -replace github.com/marefr/go-conntrack/v2=../
	go mod edit -replace github.com/marefr/go-conntrack/providers/prometheus=../providers/prometheus
	go mod edit -replace github.com/marefr/go-conntrack/providers/trace=../providers/trace
	go mod tidy

replace-drop:
	go mod edit -dropreplace=github.com/marefr/go-conntrack/v2
	go mod edit -dropreplace=github.com/marefr/go-conntrack/providers/prometheus
	go mod edit -dropreplace=github.com/marefr/go-conntrack/providers/trace
	go mod tidy

deps:
	go mod tidy

lint:
	golangci-lint run --config=../.golangci.yml

test:
