package api

import (
	"net/http"

	"orders_backend/service"

	"github.com/gin-gonic/gin"
)

func (h *Handler) updateOrder(c *gin.Context) {
	orderID, err := parsePositiveInt(c.Param("orderId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid orderId", []apiFieldError{{Field: "orderId", Message: "Must be a positive integer"}})
		return
	}
	var request service.OrderRequest
	if err := decodeRequest(c, &request); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", []apiFieldError{{Field: "body", Message: "Must be valid JSON with only supported fields"}})
		return
	}
	result, err := h.service.UpdateOrder(c.Request.Context(), toRepositoryPrincipal(getPrincipal(c)), orderID, request)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	writeSuccess(c, result, nil)
}

func (h *Handler) deleteOrder(c *gin.Context) {
	orderID, err := parsePositiveInt(c.Param("orderId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid orderId", []apiFieldError{{Field: "orderId", Message: "Must be a positive integer"}})
		return
	}
	if err := h.service.DeleteOrder(c.Request.Context(), toRepositoryPrincipal(getPrincipal(c)), orderID); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
