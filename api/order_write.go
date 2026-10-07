package api

import (
	"net/http"
	"strconv"

	"orders_backend/service"

	"github.com/gin-gonic/gin"
)

func (h *Handler) createOrder(c *gin.Context) {
	var request service.OrderRequest
	if err := decodeRequest(c, &request); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", []apiFieldError{{Field: "body", Message: "Must be valid JSON with only supported fields"}})
		return
	}
	result, err := h.service.CreateOrder(c.Request.Context(), toRepositoryPrincipal(getPrincipal(c)), request)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.Header("Location", "/api/v1/orders/"+strconv.FormatInt(result.ID, 10))
	c.JSON(http.StatusCreated, gin.H{"data": result})
}
