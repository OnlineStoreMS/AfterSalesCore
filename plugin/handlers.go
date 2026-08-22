package plugin

import (
	"context"
	"net/http"
	"strings"

	"aftersalescore/internal/dto"
	"aftersalescore/internal/model"
	"aftersalescore/internal/pkg/httputil"
	"aftersalescore/internal/pkg/response"
	"aftersalescore/internal/plugindebug"
	"aftersalescore/internal/service"

	"github.com/gin-gonic/gin"
)

const contextShop = "plugin_shop"

type Handler struct {
	svc    *service.ShopService
	notify *service.NotificationService
	debug  *plugindebug.Store
}

func NewHandler(svc *service.ShopService, notify *service.NotificationService, debug *plugindebug.Store) *Handler {
	return &Handler{svc: svc, notify: notify, debug: debug}
}

func (h *Handler) Bind(c *gin.Context) {
	var in dto.PluginBindInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.Bind(in.BindCode)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handler) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		key, secret := pluginCreds(c)
		shop, err := h.svc.AuthenticatePlugin(key, secret)
		if err != nil {
			httputil.HandleServiceError(c, err)
			c.Abort()
			return
		}
		c.Set(contextShop, shop)
		c.Next()
	}
}

func (h *Handler) Heartbeat(c *gin.Context) {
	shop := mustShop(c)
	if shop == nil {
		response.Fail(c, http.StatusUnauthorized, service.ErrPluginAuth.Error())
		return
	}
	var in dto.PluginHeartbeatInput
	_ = c.ShouldBindJSON(&in)
	item, err := h.svc.Heartbeat(shop, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handler) Sync(c *gin.Context) {
	shop := mustShop(c)
	if shop == nil {
		response.Fail(c, http.StatusUnauthorized, service.ErrPluginAuth.Error())
		return
	}
	var in dto.PluginSyncInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.Sync(shop, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	if h.notify != nil {
		go h.notify.NotifyShop(context.Background(), shop.TenantID, shop.ID)
	}
	response.OK(c, item)
}

func pluginCreds(c *gin.Context) (key, secret string) {
	key = strings.TrimSpace(c.GetHeader("X-Plugin-Key"))
	secret = strings.TrimSpace(c.GetHeader("X-Plugin-Secret"))
	return key, secret
}

func mustShop(c *gin.Context) *model.MarketplaceShop {
	v, _ := c.Get(contextShop)
	shop, _ := v.(*model.MarketplaceShop)
	return shop
}
