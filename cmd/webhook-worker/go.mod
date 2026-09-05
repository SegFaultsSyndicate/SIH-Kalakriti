module github.com/ZoroNewbie00/kalakriti/cmd/webhook-worker

go 1.25.0

require (
	github.com/ZoroNewbie00/kalakriti/pkg v0.0.0
	github.com/lib/pq v1.10.9
)

require github.com/google/uuid v1.6.0 // indirect

replace github.com/ZoroNewbie00/kalakriti/pkg => ../../pkg
