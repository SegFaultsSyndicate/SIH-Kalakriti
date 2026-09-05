// tests/integration/main_test.go
package integration

import (
	"log"
	"os"
	"testing"

	"github.com/ZoroNewbie00/kalakriti/services/bff/bfftest"
)

var (
	testServer   *bfftest.TestServer
	artisanToken string
	buyerToken   string
	adminToken   string
)

func TestMain(m *testing.M) {
	var err error
	testServer, err = bfftest.Start()
	if err != nil {
		log.Fatalf("failed to start test bff server: %v", err)
	}
	defer testServer.Close()

	artisanToken = testServer.ArtisanToken
	buyerToken = testServer.BuyerToken
	adminToken = testServer.AdminToken

	code := m.Run()
	os.Exit(code)
}
