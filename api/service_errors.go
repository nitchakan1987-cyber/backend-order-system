package api

import (
	"errors"
	"net/http"

	"orders_backend/repository"
	"orders_backend/service"

	"github.com/gin-gonic/gin"
)

func writeServiceError(c *gin.Context, err error) {
	var validation *service.ValidationError
	if errors.As(err, &validation) {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", validation.Fields)
		return
	}
	switch {
	case errors.Is(err, service.ErrForbidden):
		writeError(c, http.StatusForbidden, "FORBIDDEN", "Salesperson access is not allowed", nil)
	case errors.Is(err, repository.ErrCustomerAccess):
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", []apiFieldError{{Field: "customerId", Message: "Customer is not mapped to the salesperson"}})
	case errors.Is(err, repository.ErrDomesticCharges):
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", []apiFieldError{{Field: "freightCharge", Message: "Charges must be zero for DOMESTIC customers"}})
	case errors.Is(err, repository.ErrProductNotFound):
		writeError(c, http.StatusNotFound, "PRODUCT_NOT_FOUND", "Product was not found", nil)
	case errors.Is(err, repository.ErrPriceNotFound):
		writeError(c, http.StatusNotFound, "PRICE_NOT_FOUND", "No price is available for this product", nil)
	case errors.Is(err, repository.ErrPriceChanged):
		writeError(c, http.StatusConflict, "PRICE_CHANGED", "The product price has changed", nil)
	case errors.Is(err, repository.ErrOrderNotFound):
		writeError(c, http.StatusNotFound, "ORDER_NOT_FOUND", "Order was not found", nil)
	case errors.Is(err, repository.ErrOrderShipped):
		writeError(c, http.StatusConflict, "ORDER_ALREADY_SHIPPED", "Shipped orders cannot be changed", nil)
	case errors.Is(err, repository.ErrInvalidOrderItem):
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", []apiFieldError{{Field: "items", Message: "A product can appear only once in an order"}})
	case errors.Is(err, repository.ErrOrderTotalOverflow):
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Order total exceeds the supported amount", nil)
	default:
		writeError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service is temporarily unavailable", nil)
	}
}
