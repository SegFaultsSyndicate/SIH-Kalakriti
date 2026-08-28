module github.com/<org>/kalakriti/services/insight-svc

go 1.23

require (
	github.com/<org>/kalakriti/pkg v0.0.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.5.5
	github.com/jung-kurt/gofpdf v1.16.2
	google.golang.org/grpc v1.62.1
	google.golang.org/protobuf v1.33.0
)

replace github.com/<org>/kalakriti/pkg => ../../pkg
