package admin

import (
	"net/http"
	"strconv"

	"aftersalescore/internal/dto"
	"aftersalescore/internal/pkg/authcontext"
	"aftersalescore/internal/pkg/httputil"
	"aftersalescore/internal/pkg/response"
	"aftersalescore/internal/repo"
	"aftersalescore/internal/service"

	"github.com/gin-gonic/gin"
)

type UnboxingHandler struct {
	svc *service.UnboxingService
}

func NewUnboxingHandler(svc *service.UnboxingService) *UnboxingHandler {
	return &UnboxingHandler{svc: svc}
}

func (h *UnboxingHandler) us(c *gin.Context) *service.UnboxingService {
	return h.svc.ForTenant(authcontext.TenantID(c))
}

func (h *UnboxingHandler) List(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.us(c).List(repo.UnboxingListFilter{
		TrackingNo: c.Query("trackingNo"),
		Page: page, PageSize: pageSize,
	})
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *UnboxingHandler) Get(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.us(c).Get(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *UnboxingHandler) Create(c *gin.Context) {
	var in dto.UnboxingCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	claims := authcontext.Claims(c)
	name := ""
	if claims != nil {
		name = claims.DisplayName
	}
	item, err := h.us(c).Create(&in, authcontext.UserID(c), name)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *UnboxingHandler) UploadVideo(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "请上传视频文件")
		return
	}
	durationSec, _ := strconv.Atoi(c.PostForm("durationSec"))
	item, err := h.us(c).UploadVideo(id, file, durationSec)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *UnboxingHandler) UploadPhoto(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "请上传图片")
		return
	}
	item, err := h.us(c).UploadPhoto(id, file, c.PostForm("issueRemark"))
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *UnboxingHandler) Complete(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.UnboxingCompleteInput
	_ = c.ShouldBindJSON(&in)
	item, err := h.us(c).Complete(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *UnboxingHandler) VideoDownload(c *gin.Context) {
	id, err := httputil.ParseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	resp, err := h.us(c).VideoDownload(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, resp)
}
