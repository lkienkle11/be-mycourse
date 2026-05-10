package mapping

import (
	"encoding/json"
	"fmt"
	"strings"

	"mycourse-io-be/dto"
	pkgerrors "mycourse-io-be/pkg/errors"
)

// UnmarshalBunnyVideoWebhookRequestJSON parses raw JSON into dto.BunnyVideoWebhookRequest.
// On JSON syntax failure it wraps pkgerrors.ErrBunnyWebhookJSONInvalid.
func UnmarshalBunnyVideoWebhookRequestJSON(raw []byte) (dto.BunnyVideoWebhookRequest, error) {
	var req dto.BunnyVideoWebhookRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return dto.BunnyVideoWebhookRequest{}, fmt.Errorf("%w: %v", pkgerrors.ErrBunnyWebhookJSONInvalid, err)
	}
	return req, nil
}

// ValidateBunnyVideoWebhookRequest checks library id, guid, and status range per webhook contract.
func ValidateBunnyVideoWebhookRequest(req dto.BunnyVideoWebhookRequest) error {
	if req.VideoLibraryID <= 0 || strings.TrimSpace(req.VideoGUID) == "" || req.Status < 0 || req.Status > 10 {
		return pkgerrors.ErrBunnyWebhookPayloadInvalid
	}
	return nil
}
