package entities

import (
	"net/http"

	"github.com/Backblaze/blazer/b2"
	gcdn "github.com/G-Core/gcorelabscdn-go"
	bunnystorage "github.com/l0wl3vel/bunny-storage-go-sdk"
)

// CloudClients stores initialized media provider SDK/HTTP clients.
type CloudClients struct {
	B2Client     *b2.Client
	B2BucketName string
	BunnyStorage *bunnystorage.Client
	GcoreService *gcdn.Service
	HTTPClient   *http.Client
}

// BunnyVideoDetail mirrors Bunny Stream get-video payload fields used by service layer.
type BunnyVideoDetail struct {
	VideoLibraryID int     `json:"videoLibraryId"`
	GUID           string  `json:"guid"`
	Length         float64 `json:"length"`
	Status         int     `json:"status"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	Framerate      float64 `json:"framerate"`
	Bitrate        int     `json:"bitrate"`
	VideoCodec     string  `json:"videoCodec"`
	AudioCodec     string  `json:"audioCodec"`
	HasMP4Fallback bool    `json:"hasMP4Fallback"`
}
