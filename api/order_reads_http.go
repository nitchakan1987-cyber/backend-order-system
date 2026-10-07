package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"orders_backend/repository"
	"orders_backend/service"

	"github.com/gin-gonic/gin"
)

func (h *Handler) listOrders(c *gin.Context) {
	startDate, err := parseQueryDate(c, "startDate")
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid date range", []apiFieldError{{Field: "startDate", Message: "Required date in YYYY-MM-DD format"}})
		return
	}
	endDate, err := parseQueryDate(c, "endDate")
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid date range", []apiFieldError{{Field: "endDate", Message: "Required date in YYYY-MM-DD format"}})
		return
	}
	if endDate.Before(startDate) {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid date range", []apiFieldError{{Field: "endDate", Message: "Must not be before startDate"}})
		return
	}
	page, err := queryInt(c, "page", 1, 1, int(^uint(0)>>1))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid pagination", []apiFieldError{{Field: "page", Message: "Must be an integer greater than or equal to 1"}})
		return
	}
	pageSize, err := queryInt(c, "pageSize", 50, 1, 100)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid pagination", []apiFieldError{{Field: "pageSize", Message: "Must be between 1 and 100"}})
		return
	}
	var salespersonID, customerID int64
	if raw := c.Query("salespersonId"); raw != "" {
		salespersonID, err = parsePositiveInt(raw)
		if err != nil {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid salespersonId", []apiFieldError{{Field: "salespersonId", Message: "Must be a positive integer"}})
			return
		}
	}
	if raw := c.Query("customerId"); raw != "" {
		customerID, err = parsePositiveInt(raw)
		if err != nil {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid customerId", []apiFieldError{{Field: "customerId", Message: "Must be a positive integer"}})
			return
		}
	}
	current := getPrincipal(c)
	result, err := h.service.Orders(c.Request.Context(), toRepositoryPrincipal(current), repository.OrderFilter{
		StartDate: startDate, EndDate: endDate, SalespersonID: salespersonID,
		CustomerID: customerID, Page: page, PageSize: pageSize,
	})
	if errors.Is(err, service.ErrForbidden) {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "Salesperson access is not allowed", nil)
		return
	}
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service is temporarily unavailable", nil)
		return
	}
	writeSuccess(c, gin.H{"summary": result.Summary, "items": result.Items}, result.Meta)
}

func (h *Handler) getOrder(c *gin.Context) {
	orderID, err := parsePositiveInt(c.Param("orderId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid orderId", []apiFieldError{{Field: "orderId", Message: "Must be a positive integer"}})
		return
	}
	result, err := h.service.Order(c.Request.Context(), toRepositoryPrincipal(getPrincipal(c)), orderID)
	if errors.Is(err, repository.ErrOrderNotFound) {
		writeError(c, http.StatusNotFound, "ORDER_NOT_FOUND", "Order was not found", nil)
		return
	}
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service is temporarily unavailable", nil)
		return
	}
	writeSuccess(c, result, nil)
}

func toRepositoryPrincipal(current principal) repository.Principal {
	return repository.Principal{TokenID: current.tokenID, SalespersonIDs: current.salespersonIDs}
}

func parseQueryDate(c *gin.Context, key string) (time.Time, error) {
	value := c.Query(key)
	if value == "" {
		return time.Time{}, strconv.ErrSyntax
	}
	return time.Parse("2006-01-02", value)
}

func queryInt(c *gin.Context, key string, fallback, minimum, maximum int) (int, error) {
	value := c.Query(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minimum || parsed > maximum {
		return 0, strconv.ErrSyntax
	}
	return parsed, nil
}
