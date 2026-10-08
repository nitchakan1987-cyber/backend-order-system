package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"orders_backend/repository"
	"orders_backend/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *service.Service
}

type principal struct {
	tokenID        int64
	salespersonIDs []int64
}

type apiFieldError = service.FieldError

type errorEnvelope struct {
	Error struct {
		Code      string          `json:"code"`
		Message   string          `json:"message"`
		Fields    []apiFieldError `json:"fields,omitempty"`
		RequestID string          `json:"requestId"`
	} `json:"error"`
}

func New(service *service.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(group *gin.RouterGroup) {
	group.Use(h.requestID(), h.authenticate())
	group.GET("/salespersons", h.listSalespersons)
	group.GET("/customers", h.listCustomers)
	group.GET("/customers/all", h.listAllCustomers)
	group.GET("/products", h.listProducts)
	group.GET("/products/:productId/price", h.getProductPrice)
	group.GET("/orders", h.listOrders)
	group.GET("/orders/:orderId", h.getOrder)
	group.POST("/orders", h.createOrder)
	group.PUT("/orders/:orderId", h.updateOrder)
	group.DELETE("/orders/:orderId", h.deleteOrder)
	group.GET("/reports/delivery-schedule", h.deliverySchedule)
	group.GET("/reports/delivery-schedule/export", h.exportDeliverySchedule)
}

func (h *Handler) requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		var value [16]byte
		if _, err := rand.Read(value[:]); err != nil {
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Request could not be processed", nil)
			c.Abort()
			return
		}
		requestID := hex.EncodeToString(value[:])
		c.Set("requestId", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func (h *Handler) authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		log.Printf("Authorization = %q", auth)
		parts := strings.Fields(auth)
		log.Printf("parts = %#v, len = %d", parts, len(parts))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required", nil)
			c.Abort()
			return
		}

		current, err := h.service.Authenticate(c.Request.Context(), parts[1])

		log.Printf("Authenticate result: current=%+v, err=%v", current, err)

		if errors.Is(err, service.ErrUnauthenticated) {
			writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required", nil)
			c.Abort()
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			writeError(c, http.StatusForbidden, "FORBIDDEN", "No salesperson access is assigned", nil)
			c.Abort()
			return
		}
		if err != nil {
			writeError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service is temporarily unavailable", nil)
			c.Abort()
			return
		}
		c.Set("principal", principal{tokenID: current.TokenID, salespersonIDs: current.SalespersonIDs})
		c.Next()
	}
}

func (h *Handler) listSalespersons(c *gin.Context) {
	current := getPrincipal(c)
	items, err := h.service.Salespersons(c.Request.Context(), repository.Principal{
		TokenID: current.tokenID, SalespersonIDs: current.salespersonIDs,
	})
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service is temporarily unavailable", nil)
		return
	}
	writeSuccess(c, items, nil)
}

func (h *Handler) listCustomers(c *gin.Context) {
	current := getPrincipal(c)
	var salespersonID int64
	if rawID := c.Query("salespersonId"); rawID != "" {
		parsedID, err := parsePositiveInt(rawID)
		if err != nil {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid salespersonId", []apiFieldError{{Field: "salespersonId", Message: "Must be a positive integer"}})
			return
		}
		salespersonID = parsedID
	}
	items, err := h.service.Customers(c.Request.Context(), repository.Principal{
		TokenID: current.tokenID, SalespersonIDs: current.salespersonIDs,
	}, salespersonID)
	if errors.Is(err, service.ErrForbidden) {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "Salesperson access is not allowed", nil)
		return
	}
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service is temporarily unavailable", nil)
		return
	}
	writeSuccess(c, items, nil)
}
func (h *Handler) listAllCustomers(c *gin.Context) {
	items, err := h.service.AllCustomers(c.Request.Context())
	if err != nil {
		writeError(
			c,
			http.StatusServiceUnavailable,
			"SERVICE_UNAVAILABLE",
			"Service is temporarily unavailable",
			nil,
		)
		return
	}
	writeSuccess(c, items, nil)
}
func (h *Handler) listProducts(c *gin.Context) {
	items, err := h.service.Products(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service is temporarily unavailable", nil)
		return
	}
	writeSuccess(c, items, nil)
}

func (h *Handler) getProductPrice(c *gin.Context) {
	productID, err := parsePositiveInt(c.Param("productId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid productId", []apiFieldError{{Field: "productId", Message: "Must be a positive integer"}})
		return
	}
	result, err := h.service.ProductPrice(c.Request.Context(), productID)
	if errors.Is(err, repository.ErrPriceNotFound) {
		writeError(c, http.StatusNotFound, "PRICE_NOT_FOUND", "No price is available for this product", nil)
		return
	}
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service is temporarily unavailable", nil)
		return
	}
	writeSuccess(c, result, nil)
}

func writeSuccess(c *gin.Context, data any, meta any) {
	body := gin.H{"data": data}
	if meta != nil {
		body["meta"] = meta
	}
	c.JSON(http.StatusOK, body)
}

func writeError(c *gin.Context, status int, code, message string, fields []apiFieldError) {
	var body errorEnvelope
	body.Error.Code = code
	body.Error.Message = message
	body.Error.Fields = fields
	if requestID, exists := c.Get("requestId"); exists {
		body.Error.RequestID, _ = requestID.(string)
	}
	c.JSON(status, body)
}

func parsePositiveInt(value string) (int64, error) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("must be a positive integer")
	}
	return parsed, nil
}

func placeholders(count int) string {
	parts := make([]string, count)
	for index := range parts {
		parts[index] = "?"
	}
	return strings.Join(parts, ",")
}

func int64Args(values []int64) []any {
	args := make([]any, len(values))
	for index, value := range values {
		args[index] = value
	}
	return args
}

func getPrincipal(c *gin.Context) principal {
	value, _ := c.Get("principal")
	current, _ := value.(principal)
	return current
}

func hasSalesperson(current principal, id int64) bool {
	for _, allowed := range current.salespersonIDs {
		if id == allowed {
			return true
		}
	}
	return false
}
