package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"orders_backend/service"

	"github.com/gin-gonic/gin"
)

func (h *Handler) deliverySchedule(c *gin.Context) {
	start, end, ok := parseDeliveryRange(c)
	if !ok {
		return
	}
	items, err := h.service.DeliverySchedule(c.Request.Context(), toRepositoryPrincipal(getPrincipal(c)), start, end)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service is temporarily unavailable", nil)
		return
	}
	writeSuccess(c, gin.H{"items": items}, gin.H{"totalItems": len(items)})
}

func (h *Handler) exportDeliverySchedule(c *gin.Context) {
	start, end, ok := parseDeliveryRange(c)
	if !ok {
		return
	}
	generatedAt := time.Now().In(time.FixedZone("Asia/Bangkok", 7*60*60))
	result, err := h.service.ExportDeliverySchedule(c.Request.Context(), toRepositoryPrincipal(getPrincipal(c)), start, end, generatedAt, strings.TrimSpace(c.Query("format")))
	if errors.Is(err, service.ErrInvalidExportFormat) {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid export format", []apiFieldError{{Field: "format", Message: "Must be pdf or xlsx"}})
		return
	}
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Report export is unavailable", nil)
		return
	}
	filename := fmt.Sprintf("delivery-schedule-%s-%s.%s", start.Format("20060102"), end.Format("20060102"), result.Extension)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Data(http.StatusOK, result.ContentType, result.Content)
}

func parseDeliveryRange(c *gin.Context) (time.Time, time.Time, bool) {
	start, err := parseQueryDate(c, "deliveryFrom")
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid delivery date range", []apiFieldError{{Field: "deliveryFrom", Message: "Required date in YYYY-MM-DD format"}})
		return time.Time{}, time.Time{}, false
	}
	end, err := parseQueryDate(c, "deliveryTo")
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid delivery date range", []apiFieldError{{Field: "deliveryTo", Message: "Required date in YYYY-MM-DD format"}})
		return time.Time{}, time.Time{}, false
	}
	if end.Before(start) {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid delivery date range", []apiFieldError{{Field: "deliveryTo", Message: "Must not be before deliveryFrom"}})
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}
