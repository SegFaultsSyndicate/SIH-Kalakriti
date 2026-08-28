module github.com/segfaultsyndicate/kalakriti/services/search-svc

go 1.23.0

require (
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.6.0
	github.com/pgvector/pgvector-go v0.2.2
	github.com/redis/go-redis/v9 v9.6.1
	github.com/segfaultsyndicate/kalakriti/pkg v0.0.0
	github.com/segmentio/kafka-go v0.4.47
	golang.org/x/sync v0.8.0
	golang.org/x/text v0.17.0
	google.golang.org/grpc v1.66.0
)

require github.com/stretchr/testify v1.9.0

replace github.com/segfaultsyndicate/kalakriti/pkg => ../../pkg
