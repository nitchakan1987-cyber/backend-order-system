package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
)

var ErrPriceNotFound = errors.New("price not found")

type MySQL struct {
	db *sql.DB
}

type Principal struct {
	TokenID        int64
	SalespersonIDs []int64
}

type Salesperson struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type Customer struct {
	ID            int64  `json:"id"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	SalespersonID int64  `json:"salespersonId"`
	MarketType    string `json:"marketType"`
}
type CustomerAll struct {
	ID         int64  `json:"id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	MarketType string `json:"marketType"`
}

type Product struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type ProductPrice struct {
	ProductID   int64  `json:"productId"`
	UnitPrice   string `json:"unitPrice"`
	Currency    string `json:"currency"`
	EffectiveAt string `json:"effectiveAt"`
}

func NewMySQL(db *sql.DB) *MySQL {
	return &MySQL{db: db}
}

func (r *MySQL) Authenticate(ctx context.Context, token string) (Principal, bool, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.id, s.salesperson_id
		FROM api_tokens AS t
		LEFT JOIN api_token_salespersons AS s ON s.token_id = t.id
		WHERE t.token_hash = ? AND t.active = TRUE`, TokenHash(token))
	if err != nil {
		return Principal{}, false, err
	}
	defer rows.Close()

	var current Principal
	validToken := false
	for rows.Next() {
		var salespersonID sql.NullInt64
		if err := rows.Scan(&current.TokenID, &salespersonID); err != nil {
			return Principal{}, false, err
		}
		validToken = true
		if salespersonID.Valid {
			current.SalespersonIDs = append(current.SalespersonIDs, salespersonID.Int64)
		}
	}
	if err := rows.Err(); err != nil {
		return Principal{}, false, err
	}
	return current, validToken, nil
}

func (r *MySQL) ListSalespersons(ctx context.Context, tokenID int64) ([]Salesperson, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT s.id, s.code, s.name
		FROM salesperson as s
		ORDER BY s.name, s.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Salesperson, 0)
	for rows.Next() {
		var item Salesperson
		var code sql.NullString
		if err := rows.Scan(&item.ID, &code, &item.Name); err != nil {
			return nil, err
		}
		item.Code = code.String
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *MySQL) ListCustomers(
	ctx context.Context,
	salespersonID int64,
) ([]Customer, error) {

	query := `
		SELECT DISTINCT
			c.id,
			c.code,
			c.name,
			ord.salesperson_id,
			CASE
				WHEN UPPER(c.marketType) = 'EXPORT' THEN 'EXPORT'
				ELSE 'DOMESTIC'
			END AS market_type
		FROM customers AS c
		INNER JOIN orders AS ord
			ON ord.customer_id = c.id
		WHERE ord.salesperson_id = ?
		ORDER BY c.name, c.id
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		salespersonID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Customer, 0)

	for rows.Next() {
		var item Customer
		var code sql.NullString

		if err := rows.Scan(
			&item.ID,
			&code,
			&item.Name,
			&item.SalespersonID,
			&item.MarketType,
		); err != nil {
			return nil, err
		}

		item.Code = code.String
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *MySQL) ListAllCustomers(
	ctx context.Context,
) ([]CustomerAll, error) {

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id,
			code,
			name,
			CASE
				WHEN UPPER(marketType) = 'EXPORT' THEN 'EXPORT'
				ELSE 'DOMESTIC'
			END AS market_type
		FROM customers
		ORDER BY name, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]CustomerAll, 0)

	for rows.Next() {
		var item CustomerAll
		var code sql.NullString

		if err := rows.Scan(
			&item.ID,
			&code,
			&item.Name,
			&item.MarketType,
		); err != nil {
			return nil, err
		}

		item.Code = code.String
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *MySQL) ListProducts(ctx context.Context) ([]Product, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, product_code, name
		FROM products
		ORDER BY name, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Product, 0)
	for rows.Next() {
		var item Product
		if err := rows.Scan(&item.ID, &item.Code, &item.Name); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *MySQL) LatestProductPrice(ctx context.Context, productID int64) (ProductPrice, error) {
	var price ProductPrice
	err := r.db.QueryRowContext(ctx, `
		SELECT p.id, CAST(p.price AS CHAR), 'THB',
			DATE_FORMAT(COALESCE(p.update_at, p.created_at, CURRENT_TIMESTAMP), '%Y-%m-%dT%H:%i:%s')
		FROM products AS p
		WHERE p.id = ?`, productID).Scan(&price.ProductID, &price.UnitPrice, &price.Currency, &price.EffectiveAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ProductPrice{}, ErrPriceNotFound
	}
	if err != nil {
		return ProductPrice{}, err
	}
	price.EffectiveAt += "+07:00"
	return price, nil
}

func TokenHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func placeholders(count int) string {
	return strings.TrimSuffix(strings.Repeat("?,", count), ",")
}
