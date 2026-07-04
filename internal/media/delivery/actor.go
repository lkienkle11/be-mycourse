package delivery

import (
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/media/application"
	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/middleware"
)

func mediaActorFromContext(c *gin.Context) (userID, userCode string) {
	if v, ok := c.Get(middleware.ContextUserID); ok {
		if s, ok := v.(string); ok {
			userID = strings.TrimSpace(s)
		}
	}
	if v, ok := c.Get(middleware.ContextUserCode); ok {
		if s, ok := v.(string); ok {
			userCode = strings.TrimSpace(s)
		}
	}
	return userID, userCode
}

func mediaActorDisplayName(c *gin.Context) string {
	if v, ok := c.Get(middleware.ContextDisplayName); ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func enrichUploaderIdentityFromActor(c *gin.Context, files []*domain.File) {
	if len(files) == 0 {
		return
	}
	actorID, _ := mediaActorFromContext(c)
	displayName := mediaActorDisplayName(c)
	if actorID == "" || displayName == "" {
		return
	}
	for _, f := range files {
		if f == nil || strings.TrimSpace(f.UserID) != actorID {
			continue
		}
		if strings.TrimSpace(f.DisplayName) == "" {
			f.DisplayName = displayName
		}
	}
}

func applyMediaActorToCreateInput(c *gin.Context, req *application.CreateFileInput) {
	userID, userCode := mediaActorFromContext(c)
	req.UserID = userID
	req.UserCode = userCode
	req.Visibility = application.NormalizeMediaVisibility(req.Visibility)
}
