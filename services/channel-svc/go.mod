module github.com/segfaultsyndicate/kalakriti/services/channel-svc

go 1.23.0

require (
	github.com/google/uuid v1.6.0
	github.com/segfaultsyndicate/kalakriti/pkg v0.0.0
	github.com/segmentio/kafka-go v0.4.47
	golang.org/x/crypto v0.26.0
)

require (
	github.com/stretchr/testify v1.9.0
)

replace github.com/segfaultsyndicate/kalakriti/pkg => ../../pkg
