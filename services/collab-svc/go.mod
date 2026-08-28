module github.com/ZoroNewbie00/kalakriti/services/collab-svc

go 1.23.0

require (
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.6.0
	github.com/ZoroNewbie00/kalakriti/pkg v0.0.0
	github.com/segmentio/kafka-go v0.4.47
	google.golang.org/grpc v1.66.0
	google.golang.org/protobuf v1.34.2
)

require (
	github.com/pressly/goose/v3 v3.21.1
	github.com/stretchr/testify v1.9.0
	github.com/testcontainers/testcontainers-go v0.33.0
	github.com/testcontainers/testcontainers-go/modules/postgres v0.33.0
)

replace github.com/ZoroNewbie00/kalakriti/pkg => ../../pkg
