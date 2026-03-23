package api

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all API routes on the router.
func RegisterRoutes(r *gin.Engine, h *Handler) {
	r.POST("/set", h.Set)
	r.GET("/get/:key", h.Get)
	r.DELETE("/delete/:key", h.Delete)
	r.POST("/publish", h.Publish)
	r.GET("/subscribe/:channel", h.Subscribe)
}
