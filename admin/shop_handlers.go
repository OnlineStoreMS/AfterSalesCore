package admin

import (
	"net/http"
	"strconv"

	"aftersalescore/internal/dto"
	"aftersalescore/internal/pkg/authcontext"
	"aftersalescore/internal/pkg/httputil"
	"aftersalescore/internal/pkg/response"
	"aftersalescore/internal/service"

	"github.com/gin-gonic/gin"
)

type ShopHandler struct {
	svc *service.ShopService
}

func NewShopHandler(svc *service.ShopService) *ShopHandler {
	return &ShopHandler{svc: svc}
}

func (h *ShopHandler) ss(c *gin.Context) *service.ShopService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *ShopHandler) List(c *gin.Context) {
	list, err := h.ss(c).List()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, list)
}

func (h *ShopHandler) Get(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).Get(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *ShopHandler) Create(c *gin.Context) {
	var in dto.ShopCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).Create(&in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *ShopHandler) Update(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.ShopUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).Update(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *ShopHandler) Delete(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.ss(c).Delete(id); err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *ShopHandler) GetPluginSetting(c *gin.Context) {
	response.OK(c, h.ss(c).GetPluginSetting())
}

func (h *ShopHandler) SavePluginSetting(c *gin.Context) {
	var in dto.PluginSetting
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.ss(c).SavePluginSetting(in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *ShopHandler) RequestSync(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).RequestSync(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *ShopHandler) ResetBind(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.ss(c).ResetBind(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *ShopHandler) Workbench(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	page, pageSize := httputil.ParsePage(c)
	data, err := h.ss(c).Workbench(id, c.Query("cardKey"), c.Query("keyword"), page, pageSize)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, data)
}

func (h *ShopHandler) Tickets(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.ss(c).ListTickets(id, c.Query("cardKey"), c.Query("keyword"), page, pageSize)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *ShopHandler) Returns(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	var shopID uint64
	if raw := c.Query("shopId"); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid shopId")
			return
		}
		shopID = id
	}
	list, total, err := h.ss(c).ListReturns(dto.ReturnListQuery{
		ShopID:     shopID,
		Keyword:    c.Query("keyword"),
		ReturnFrom: c.Query("returnFrom"),
		ReturnTo:   c.Query("returnTo"),
		ApplyFrom:  c.Query("applyFrom"),
		ApplyTo:    c.Query("applyTo"),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}
