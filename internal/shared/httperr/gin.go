package httperr

import (
	"github.com/gin-gonic/gin"
)

// Abort registers err on the Gin context and stops the handler chain without writing the body.
// The global ErrorHandler middleware writes the JSON response after Next() returns.
func Abort(c *gin.Context, err error) {
	_ = c.Error(err)
	c.Abort()
}
