package health

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"orders_backend/response"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		response.Error(
			w,
			http.StatusServiceUnavailable,
			"DATABASE_UNAVAILABLE",
			"Database unavailable",
		)
		return
	}

	response.Success(w, "API is running", map[string]any{
		"database": "connected",
	})
}
