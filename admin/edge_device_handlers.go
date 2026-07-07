package admin

import (
	"net/http"

	"aftersalescore/internal/dto"
	"aftersalescore/internal/pkg/httputil"
	"aftersalescore/internal/pkg/response"
	"aftersalescore/internal/service"

	"github.com/gin-gonic/gin"
)

type EdgeDeviceHandler struct {
	svc *service.EdgeDeviceService
}

func NewEdgeDeviceHandler(svc *service.EdgeDeviceService) *EdgeDeviceHandler {
	return &EdgeDeviceHandler{svc: svc}
}

func (h *EdgeDeviceHandler) List(c *gin.Context) {
	list, err := h.svc.List()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *EdgeDeviceHandler) Create(c *gin.Context) {
	var in dto.EdgeDeviceCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.Create(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *EdgeDeviceHandler) Update(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.EdgeDeviceUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.Update(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *EdgeDeviceHandler) Delete(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *EdgeDeviceHandler) Probe(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.svc.Probe(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *EdgeDeviceHandler) Sync(c *gin.Context) {
	if err := h.svc.SyncFromRecords(); err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	list, err := h.svc.List()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, list)
}
