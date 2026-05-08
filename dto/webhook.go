package dto

type BunnyVideoWebhookRequest struct {
	VideoLibraryID int    `json:"VideoLibraryId" binding:"required"`
	VideoGUID      string `json:"VideoGuid" binding:"required"`
	Status         int    `json:"Status" binding:"required,min=0,max=10"`
}
