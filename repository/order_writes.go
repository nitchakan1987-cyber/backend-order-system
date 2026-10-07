package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"time"
)

func (r *MySQL) CreateOrder(ctx context.Context, draft OrderDraft) (int64, error) {
	tx, orderNo, releaseLock, err := r.beginOrderTransaction(ctx, draft.OrderDate)
	if err != nil {
		return 0, err
	}
	defer releaseLock()
	defer tx.Rollback()
	if err := validateOrder(ctx, tx, &draft, nil); err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO orders (order_no, salesperson_id, customer_id, order_date, freight_charge, insurance_charge)
		VALUES (?, ?, ?, ?, ?, ?)`, orderNo, draft.SalespersonID, draft.CustomerID,
		draft.OrderDate.Format("2006-01-02"), moneyString(draft.FreightCents), moneyString(draft.InsuranceCents))
	if err != nil {
		return 0, err
	}
	orderID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	for _, item := range draft.Items {
		if err := insertOrderItem(ctx, tx, orderID, item); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return orderID, nil
}

func (r *MySQL) UpdateOrder(ctx context.Context, orderID int64, salespersonIDs []int64, draft OrderDraft) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	args := []any{orderID}
	for _, id := range salespersonIDs {
		args = append(args, id)
	}
	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT CASE WHEN EXISTS (
			SELECT 1 FROM order_items AS shipped
			WHERE shipped.order_id = orders.id AND shipped.deliveryStatus = 'SHIPPED'
		) THEN 'SHIPPED' ELSE 'NOT_SHIPPED' END
		FROM orders WHERE id = ? AND deleted_at IS NULL
		AND salesperson_id IN (`+placeholders(len(salespersonIDs))+`) FOR UPDATE`, args...).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrOrderNotFound
	}
	if err != nil {
		return err
	}
	if status == "SHIPPED" {
		return ErrOrderShipped
	}
	oldPrices, err := orderItemPrices(ctx, tx, orderID)
	if err != nil {
		return err
	}
	if err := validateOrder(ctx, tx, &draft, oldPrices); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE orders SET order_date = ?, salesperson_id = ?, customer_id = ?,
			freight_charge = ?, insurance_charge = ?, update_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL`, draft.OrderDate.Format("2006-01-02"),
		draft.SalespersonID, draft.CustomerID, moneyString(draft.FreightCents),
		moneyString(draft.InsuranceCents), orderID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM order_items WHERE order_id = ?`, orderID); err != nil {
		return err
	}
	for _, item := range draft.Items {
		if err := insertOrderItem(ctx, tx, orderID, item); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *MySQL) DeleteOrder(ctx context.Context, orderID int64, salespersonIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	args := []any{orderID}
	for _, id := range salespersonIDs {
		args = append(args, id)
	}
	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT CASE WHEN EXISTS (
			SELECT 1 FROM order_items AS shipped
			WHERE shipped.order_id = orders.id AND shipped.deliveryStatus = 'SHIPPED'
		) THEN 'SHIPPED' ELSE 'NOT_SHIPPED' END
		FROM orders WHERE id = ? AND deleted_at IS NULL
		AND salesperson_id IN (`+placeholders(len(salespersonIDs))+`) FOR UPDATE`, args...).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrOrderNotFound
	}
	if err != nil {
		return err
	}
	if status == "SHIPPED" {
		return ErrOrderShipped
	}
	if _, err := tx.ExecContext(ctx, `UPDATE orders SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL`, orderID); err != nil {
		return err
	}
	return tx.Commit()
}

func validateOrder(ctx context.Context, tx *sql.Tx, draft *OrderDraft, oldPrices map[int64]int64) error {
	marketType, err := customerMarketType(ctx, tx, draft.SalespersonID, draft.CustomerID)
	if err != nil {
		return err
	}
	if marketType != "EXPORT" && (draft.FreightCents != 0 || draft.InsuranceCents != 0) {
		return ErrDomesticCharges
	}
	seen := make(map[int64]bool, len(draft.Items))
	var subtotal big.Int
	for index := range draft.Items {
		item := &draft.Items[index]
		if seen[item.ProductID] {
			return ErrInvalidOrderItem
		}
		seen[item.ProductID] = true
		price, exists := oldPrices[item.ProductID]
		if !exists {
			price, err = productPriceCents(ctx, tx, item.ProductID)
			if err != nil {
				return err
			}
			if item.UnitPriceCents > 0 && item.UnitPriceCents != price {
				return ErrPriceChanged
			}
		}
		item.UnitPriceCents = price
		line := new(big.Int).Mul(big.NewInt(price), big.NewInt(item.Quantity))
		if !line.IsInt64() || line.Int64() > maxMoneyCents {
			return ErrOrderTotalOverflow
		}
		item.LineTotalCents = line.Int64()
		subtotal.Add(&subtotal, line)
	}
	additional := new(big.Int).Add(big.NewInt(draft.FreightCents), big.NewInt(draft.InsuranceCents))
	grandTotal := new(big.Int).Add(new(big.Int).Set(&subtotal), additional)
	if !subtotal.IsInt64() || subtotal.Int64() > maxMoneyCents ||
		!additional.IsInt64() || additional.Int64() > maxMoneyCents ||
		!grandTotal.IsInt64() || grandTotal.Int64() > maxMoneyCents {
		return ErrOrderTotalOverflow
	}
	draft.SubtotalCents = subtotal.Int64()
	draft.AdditionalCents = additional.Int64()
	draft.GrandTotalCents = grandTotal.Int64()
	return nil
}

func customerMarketType(ctx context.Context, tx *sql.Tx, salespersonID, customerID int64) (string, error) {
	var marketType string
	err := tx.QueryRowContext(ctx, `
		SELECT CASE WHEN UPPER(c.marketType) = 'EXPORT' THEN 'EXPORT' ELSE 'DOMESTIC' END
		FROM salesperson_customers AS mapping
		JOIN customers AS c ON c.id = mapping.customer_id
		WHERE mapping.salesperson_id = ? AND mapping.customer_id = ? AND mapping.active = TRUE`,
		salespersonID, customerID).Scan(&marketType)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrCustomerAccess
	}
	return marketType, err
}

func productPriceCents(ctx context.Context, tx *sql.Tx, productID int64) (int64, error) {
	var price int64
	err := tx.QueryRowContext(ctx, `SELECT CAST(ROUND(price * 100) AS SIGNED) FROM products WHERE id = ?`, productID).Scan(&price)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrProductNotFound
	}
	return price, err
}

func orderItemPrices(ctx context.Context, tx *sql.Tx, orderID int64) (map[int64]int64, error) {
	rows, err := tx.QueryContext(ctx, `SELECT product_id, CAST(ROUND(price * 100) AS SIGNED) FROM order_items WHERE order_id = ? FOR UPDATE`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	prices := make(map[int64]int64)
	for rows.Next() {
		var productID, price int64
		if err := rows.Scan(&productID, &price); err != nil {
			return nil, err
		}
		prices[productID] = price
	}
	return prices, rows.Err()
}

func insertOrderItem(ctx context.Context, tx *sql.Tx, orderID int64, item OrderDraftItem) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO order_items (order_id, product_id, quantity, price, delivery_at, deliveryStatus)
		VALUES (?, ?, ?, ?, ?, 'NOT_SHIPPED')`, orderID, item.ProductID, item.Quantity,
		moneyString(item.UnitPriceCents), item.DeliveryDate.Format("2006-01-02"))
	return err
}

func (r *MySQL) beginOrderTransaction(ctx context.Context, date time.Time) (*sql.Tx, string, func(), error) {
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return nil, "", func() {}, err
	}
	month := date.Format("200601")
	lockName := "orders:" + month
	var acquired sql.NullInt64
	if err := conn.QueryRowContext(ctx, `SELECT GET_LOCK(?, 10)`, lockName).Scan(&acquired); err != nil || !acquired.Valid || acquired.Int64 != 1 {
		conn.Close()
		if err != nil {
			return nil, "", func() {}, err
		}
		return nil, "", func() {}, errors.New("could not reserve order number")
	}
	release := func() {
		var released sql.NullInt64
		_ = conn.QueryRowContext(context.Background(), `SELECT RELEASE_LOCK(?)`, lockName).Scan(&released)
		_ = conn.Close()
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		release()
		return nil, "", func() {}, err
	}
	pattern := "^BPI-" + month + `[0-9]{4}$`
	var sequence int64
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(CAST(SUBSTRING(order_no, 11) AS UNSIGNED)), 0) + 1
		FROM orders WHERE order_no REGEXP ?`, pattern).Scan(&sequence); err != nil {
		tx.Rollback()
		release()
		return nil, "", func() {}, err
	}
	return tx, fmt.Sprintf("BPI-%s%04d", month, sequence), release, nil
}

const maxMoneyCents int64 = 999999999999999999

func moneyString(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}
