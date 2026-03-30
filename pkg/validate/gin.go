package validate

import (
	"github.com/gin-gonic/gin"
)

// BindJSON decodes the body and runs validate tags on the struct (field tag `validate:"..."`).
// Combine with binding tags on the same struct if you want both Gin binding and validate rules.
func BindJSON(c *gin.Context, dst interface{}) error {
	if err := c.ShouldBindJSON(dst); err != nil {
		return err
	}
	return V.Struct(dst)
}
