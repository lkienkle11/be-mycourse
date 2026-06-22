package infra

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	bunnystorage "github.com/l0wl3vel/bunny-storage-go-sdk"
)

// CloudClients stores initialized media provider SDK/HTTP clients.
// This is infra-only: the domain layer never depends on concrete SDK types.
type CloudClients struct {
	R2Client     *s3.Client
	R2BucketName string
	BunnyStorage *bunnystorage.Client
	HTTPClient   *http.Client
}
