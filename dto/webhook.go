package dto

type BunnyVideoWebhookRequest struct {
	VideoLibraryID string `json:"video_library_id" binding:"required"`
	VideoGUID      string `json:"video_guid" binding:"required"`
	Status         int    `json:"status" binding:"required"`
}
