package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"my-project/internal/pubsub"
	"my-project/internal/service"
)

// Handler holds API handlers.
type Handler struct {
	svc     *service.Service
	pubsub  *pubsub.PubSub
}

// NewHandler creates a new Handler.
func NewHandler(svc *service.Service, ps *pubsub.PubSub) *Handler {
	return &Handler{svc: svc, pubsub: ps}
}

// SetRequest is the JSON body for POST /set.
type SetRequest struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
	TTL   *int64 `json:"ttl,omitempty"`
}

// Set handles POST /set.
func (h *Handler) Set(c *gin.Context) {
	var req SetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.svc.Set(req.Key, req.Value, req.TTL)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Get handles GET /get/:key.
func (h *Handler) Get(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}
	value, ok := h.svc.Get(key)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"key": key, "value": value})
}

// Delete handles DELETE /delete/:key.
func (h *Handler) Delete(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}
	_, ok := h.svc.Get(key)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	h.svc.Delete(key)
	c.Status(http.StatusNoContent)
}

// PublishRequest is the JSON body for POST /publish.
type PublishRequest struct {
	Channel string `json:"channel" binding:"required"`
	Message string `json:"message" binding:"required"`
}

// Publish handles POST /publish.
func (h *Handler) Publish(c *gin.Context) {
	if h.pubsub == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "pubsub not configured"})
		return
	}
	var req PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.pubsub.Publish(req.Channel, req.Message)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Subscribe handles GET /subscribe/:channel - streams messages as Server-Sent Events.
func (h *Handler) Subscribe(c *gin.Context) {
	if h.pubsub == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "pubsub not configured"})
		return
	}
	channel := c.Param("channel")
	if channel == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel is required"})
		return
	}

	sub := h.pubsub.Subscribe(channel)
	defer sub.Unsubscribe()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case msg, ok := <-sub.Ch:
			if !ok {
				return
			}
			escaped := strings.ReplaceAll(msg, "\n", "\\n")
			escaped = strings.ReplaceAll(escaped, "\r", "\\r")
			c.SSEvent("message", escaped)
			c.Writer.Flush()
		}
	}
}
