package internalapi

import (
	"net/http"
	"strconv"
	"strings"

	"aftersalescore/internal/pkg/httputil"
	"aftersalescore/internal/pkg/response"
	"aftersalescore/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	shops *service.ShopService
	token string
}

func NewHandler(shops *service.ShopService, token string) *Handler {
	return &Handler{shops: shops, token: strings.TrimSpace(token)}
}

func (h *Handler) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		got := strings.TrimSpace(c.GetHeader("X-Internal-Token"))
		if h.token == "" || got == "" || got != h.token {
			response.Fail(c, http.StatusUnauthorized, "invalid internal token")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (h *Handler) AgentShopCredential(c *gin.Context) {
	tenantID, _ := strconv.ParseUint(c.Query("tenantId"), 10, 64)
	platform := c.Query("platform")
	platformShopID := c.Query("platformShopId")
	cred, err := h.shops.AgentCredentialByPlatformShop(tenantID, platform, platformShopID)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, cred)
}
