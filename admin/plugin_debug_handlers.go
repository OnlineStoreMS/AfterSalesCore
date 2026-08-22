package admin

import (
	"errors"
	"net/http"

	"aftersalescore/internal/plugindebug"
	"aftersalescore/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type PluginDebugHandler struct {
	store *plugindebug.Store
}

func NewPluginDebugHandler(store *plugindebug.Store) *PluginDebugHandler {
	return &PluginDebugHandler{store: store}
}

func (h *PluginDebugHandler) List(c *gin.Context) {
	if h.store == nil {
		response.Fail(c, http.StatusServiceUnavailable, "debug log store unavailable")
		return
	}
	list, err := h.store.List()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, gin.H{"dir": h.store.Dir(), "list": list})
}

func (h *PluginDebugHandler) Get(c *gin.Context) {
	if h.store == nil {
		response.Fail(c, http.StatusServiceUnavailable, "debug log store unavailable")
		return
	}
	rec, err := h.store.Get(c.Param("name"))
	if err != nil {
		if errors.Is(err, plugindebug.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, plugindebug.ErrBadName) {
			response.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, rec)
}
