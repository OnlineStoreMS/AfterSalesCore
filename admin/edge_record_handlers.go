package admin

import (
	"io"
	"net/http"
	"net/url"
	"strconv"

	"aftersalescore/internal/dto"
	"aftersalescore/internal/pkg/httputil"
	"aftersalescore/internal/pkg/response"
	"aftersalescore/internal/repo"
	"aftersalescore/internal/service"

	"github.com/gin-gonic/gin"
)

type EdgeRecordHandler struct {
	svc *service.EdgeRecordService
}

func NewEdgeRecordHandler(svc *service.EdgeRecordService) *EdgeRecordHandler {
	return &EdgeRecordHandler{svc: svc}
}

func (h *EdgeRecordHandler) List(c *gin.Context) {
	page, pageSize := httputil.ParsePage(c)
	list, total, err := h.svc.List(c.Request.Context(), c.GetHeader("Authorization"), repo.EdgeRecordListFilter{
		Type: c.Query("type"), TrackingNo: c.Query("trackingNo"),
		EdgeID: c.Query("edgeId"), Status: c.Query("status"),
		WithGoods: c.Query("withGoods") == "1" || c.Query("withGoods") == "true",
		Page: page, PageSize: pageSize,
	})
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, response.PageResult(list, total, page, pageSize))
}

func (h *EdgeRecordHandler) Stats(c *gin.Context) {
	stats, err := h.svc.DashboardStats()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, stats)
}

func (h *EdgeRecordHandler) Get(c *gin.Context) {
	id, err := parseInt64ID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.svc.Get(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *EdgeRecordHandler) Create(c *gin.Context) {
	var in dto.EdgeRecordCreateInput
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

func (h *EdgeRecordHandler) UploadVideo(c *gin.Context) {
	id, err := parseInt64ID(c)
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
	item, err := h.svc.UploadVideo(id, file, durationSec)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *EdgeRecordHandler) UploadPhoto(c *gin.Context) {
	id, err := parseInt64ID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "请上传图片")
		return
	}
	item, err := h.svc.UploadPhoto(id, file)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *EdgeRecordHandler) Complete(c *gin.Context) {
	id, err := parseInt64ID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in dto.EdgeRecordCompleteInput
	_ = c.ShouldBindJSON(&in)
	item, err := h.svc.Complete(id, &in)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *EdgeRecordHandler) Delete(c *gin.Context) {
	id, err := parseInt64ID(c)
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

func (h *EdgeRecordHandler) BatchDelete(c *gin.Context) {
	var in dto.EdgeRecordBatchDeleteInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	n, err := h.svc.BatchDelete(in.IDs)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": n})
}

func (h *EdgeRecordHandler) VideoDownload(c *gin.Context) {
	id, err := parseInt64ID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	file, err := h.svc.OpenVideoDownload(id)
	if err != nil {
		httputil.HandleServiceError(c, err)
		return
	}
	defer file.Reader.Close()

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", attachmentDisposition(file.Filename))
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, file.Reader)
}

func attachmentDisposition(filename string) string {
	return `attachment; filename="` + filename + `"; filename*=UTF-8''` + url.PathEscape(filename)
}

func parseInt64ID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}
