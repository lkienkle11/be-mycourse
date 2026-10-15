package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/slug"
)

// FailIfSlugConflict writes the shared HTTP 409 SlugConflict envelope (with
// the free `recommended_slug`) and returns true when err wraps a
// *slug.ConflictError; otherwise it writes nothing and returns false. One
// mapper for every slugged entity (course, taxonomy, user) instead of a copy
// per delivery package.
func FailIfSlugConflict(c *gin.Context, err error) bool {
	var conflict *slug.ConflictError
	if !errors.As(err, &conflict) {
		return false
	}
	Fail(c, http.StatusConflict, apperrors.SlugConflict,
		apperrors.DefaultMessage(apperrors.SlugConflict),
		gin.H{"recommended_slug": conflict.RecommendedSlug})
	return true
}
