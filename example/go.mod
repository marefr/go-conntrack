module github.com/marefr/go-conntrack/v2/example

go 1.23.0

require (
	github.com/marefr/go-conntrack/providers/prometheus v0.0.0-00010101000000-000000000000
	github.com/marefr/go-conntrack/providers/trace v0.0.0-00010101000000-000000000000
	github.com/marefr/go-conntrack/v2 v2.0.9
	github.com/prometheus/client_golang v1.21.1
	golang.org/x/net v0.37.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.62.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	golang.org/x/sys v0.31.0 // indirect
	google.golang.org/protobuf v1.36.1 // indirect
)

replace github.com/marefr/go-conntrack/v2 => ../

replace github.com/marefr/go-conntrack/providers/prometheus => ../providers/prometheus

replace github.com/marefr/go-conntrack/providers/trace => ../providers/trace
