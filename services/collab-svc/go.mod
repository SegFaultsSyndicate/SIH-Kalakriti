module github.com/ZoroNewbie00/kalakriti/services/collab-svc

go 1.25.0

require (
	github.com/ZoroNewbie00/kalakriti/pkg v0.0.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.6.0
	github.com/segmentio/kafka-go v0.4.47
	google.golang.org/grpc v1.66.0
	google.golang.org/protobuf v1.36.10
)

require github.com/pgvector/pgvector-go v0.2.2

require (
	github.com/caarlos0/env/v11 v11.2.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.18 // indirect
	github.com/xdg-go/scram v1.2.0 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240604185151-ef581f913117 // indirect
)

replace github.com/ZoroNewbie00/kalakriti/pkg => ../../pkg
