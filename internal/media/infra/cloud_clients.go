package infra

import (
	"net/http"

	"github.com/Backblaze/blazer/b2"
	gcdn "github.com/G-Core/gcorelabscdn-go"
	bunnystorage "github.com/l0wl3vel/bunny-storage-go-sdk"
)

// CloudClients stores initialized media provider SDK/HTTP clients.
// This is infra-only: the domain layer never depends on concrete SDK types.
type CloudClients struct {
	B2Client     *b2.Client
	B2BucketName string
	BunnyStorage *bunnystorage.Client
	GcoreService *gcdn.Service
	HTTPClient   *http.Client
}
