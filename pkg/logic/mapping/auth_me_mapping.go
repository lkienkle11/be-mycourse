package mapping

import (
	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/entities"
)

// BuildMeProfileFromUser builds the service-layer /me projection (cache + auth services).
func BuildMeProfileFromUser(user models.User, permissions []string) *entities.MeProfile {
	return &entities.MeProfile{
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

// ToMeResponseFromProfile maps MeProfile to the HTTP DTO (handlers only).
func ToMeResponseFromProfile(p *entities.MeProfile) *dto.MeResponse {
	if p == nil {
		return nil
	}
	return &dto.MeResponse{
		UserID:         p.UserID,
		UserCode:       p.UserCode,
		Email:          p.Email,
		DisplayName:    p.DisplayName,
		Avatar:         p.Avatar,
		EmailConfirmed: p.EmailConfirmed,
		IsDisabled:     p.IsDisabled,
		CreatedAt:      p.CreatedAt,
		Permissions:    p.Permissions,
	}
}
