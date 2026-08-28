module github.com/segfaultsyndicate/kalakriti/services/bff

go 1.23.0

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/redis/go-redis/v9 v9.6.1
	github.com/segfaultsyndicate/kalakriti/pkg v0.0.0
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/net v0.26.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240604185151-ef581f913117 // indirect
	google.golang.org/grpc v1.66.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

replace github.com/segfaultsyndicate/kalakriti/pkg => ../../pkg
