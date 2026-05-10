package mapping

import (
	"mycourse-io-be/dto"
)

// ToLoginSessionTokensResponse maps issued token strings to the public login/confirm/refresh JSON payload.
func ToLoginSessionTokensResponse(accessToken, refreshToken, sessionID string) dto.LoginSessionTokensResponse {
	return dto.LoginSessionTokensResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		SessionID:    sessionID,
	}
}
