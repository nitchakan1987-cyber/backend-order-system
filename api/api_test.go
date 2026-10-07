package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"orders_backend/service"

	"github.com/gin-gonic/gin"
)

func TestAPIEndpointsRequireBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiGroup := router.Group("/api/v1")
	New(service.New(nil)).Register(apiGroup)
	requests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/salespersons"},
		{http.MethodGet, "/api/v1/customers"},
		{http.MethodGet, "/api/v1/products"},
		{http.MethodGet, "/api/v1/products/1/price"},
		{http.MethodGet, "/api/v1/orders"},
		{http.MethodGet, "/api/v1/orders/1"},
		{http.MethodPost, "/api/v1/orders"},
		{http.MethodPut, "/api/v1/orders/1"},
		{http.MethodDelete, "/api/v1/orders/1"},
		{http.MethodGet, "/api/v1/reports/delivery-schedule"},
		{http.MethodGet, "/api/v1/reports/delivery-schedule/export"},
	}
	for _, item := range requests {
		t.Run(item.method+" "+item.path, func(t *testing.T) {
			request := httptest.NewRequest(item.method, item.path, nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
			var body errorEnvelope
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body.Error.Code != "UNAUTHENTICATED" || body.Error.RequestID == "" {
				t.Fatalf("unexpected error response: %#v", body.Error)
			}
		})
	}
}
