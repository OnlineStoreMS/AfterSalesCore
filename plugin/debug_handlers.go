package plugin

import (
	"net/http"

	"aftersalescore/internal/dto"
	"aftersalescore/internal/plugindebug"
	"aftersalescore/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func (h *Handler) UploadDebugLog(c *gin.Context) {
	if h.debug == nil {
		response.Fail(c, http.StatusServiceUnavailable, "debug log store unavailable")
		return
	}
	shop := mustShop(c)
	if shop == nil {
		response.Fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in dto.PluginDebugLogInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	events := make([]plugindebug.Event, 0, len(in.Events))
	for _, e := range in.Events {
		events = append(events, plugindebug.Event{
			Ms:    e.Ms,
			At:    e.At,
			Level: e.Level,
			Step:  e.Step,
			Data:  e.Data,
		})
	}
	name, err := h.debug.Save(&plugindebug.Record{
		RunID:      in.RunID,
		Kind:       in.Kind,
		OK:         in.OK,
		Error:      in.Error,
		DurationMs: in.DurationMs,
		Version:    in.Version,
		ShopID:     shop.ID,
		ShopName:   shop.Name,
		Meta:       in.Meta,
		Events:     events,
	})
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, dto.PluginDebugLogResult{Name: name, Dir: h.debug.Dir()})
}
