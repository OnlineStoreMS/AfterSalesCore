package admin

import (
	"net/http"

	"aftersalescore/internal/dto"
	"aftersalescore/internal/pkg/authcontext"
	"aftersalescore/internal/pkg/httputil"
	"aftersalescore/internal/pkg/response"
	"aftersalescore/internal/service"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	svc *service.NotificationService
}

func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

func (h *NotificationHandler) ns(c *gin.Context) *service.NotificationService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *NotificationHandler) Get(c *gin.Context) {
	view, err := h.ns(c).GetView()
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, view)
}

func (h *NotificationHandler) Save(c *gin.Context) {
	var in dto.NotificationConfig
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	view, err := h.ns(c).SaveConfig(in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, view)
}

func (h *NotificationHandler) Test(c *gin.Context) {
	var req struct {
		Text string `json:"text"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.ns(c).Test(c.Request.Context(), req.Text); err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *NotificationHandler) TestBarcode(c *gin.Context) {
	if err := h.ns(c).TestBarcode(c.Request.Context()); err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *NotificationHandler) Run(c *gin.Context) {
	result, err := h.ns(c).RunPoll(c.Request.Context(), 0)
	if err != nil {
		if result != nil && result.Sent > 0 {
			c.JSON(http.StatusBadGateway, gin.H{
				"code":    http.StatusBadGateway,
				"message": err.Error(),
				"data":    result,
			})
			return
		}
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *NotificationHandler) ResetState(c *gin.Context) {
	view, cleared, err := h.ns(c).ResetState()
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"cleared": cleared, "view": view})
}
