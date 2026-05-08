package mapping

import (
	"mycourse-io-be/dto"
	"mycourse-io-be/models"
)

func BuildMeResponseFromUser(user models.User, permissions []string) *dto.MeResponse {
	return &dto.MeResponse{
		UserID:         user.ID,
		UserCode:       user.UserCode,
		Email:          user.Email,
		DisplayName:    user.DisplayName,
		Avatar:         ToMediaFilePublicFromModel(user.AvatarFile),
		EmailConfirmed: user.EmailConfirmed,
		IsDisabled:     user.IsDisable,
		CreatedAt:      user.CreatedAt.Unix(),
		Permissions:    permissions,
	}
}
