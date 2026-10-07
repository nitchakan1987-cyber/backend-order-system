package repository

import (
	"context"
	"time"
)

type DeliveryRow struct {
	DeliveryDate      string `json:"deliveryDate"`
	OrderID           int64  `json:"orderId"`
	OrderNo           string `json:"orderNo"`
	CustomerID        int64  `json:"customerId"`
	CustomerName      string `json:"customerName"`
	ProductID         int64  `json:"productId"`
	ProductCode       string `json:"productCode"`
	ProductName       string `json:"productName"`
	QuantityToDeliver int64  `json:"quantityToDeliver"`
	DeliveryStatus    string `json:"deliveryStatus"`
}

func (r *MySQL) ListDeliverySchedule(ctx context.Context, salespersonIDs []int64, start, end time.Time) ([]DeliveryRow, error) {
	if len(salespersonIDs) == 0 {
		return []DeliveryRow{}, nil
	}
	args := []any{start.Format("2006-01-02"), end.AddDate(0, 0, 1).Format("2006-01-02")}
	for _, id := range salespersonIDs {
		args = append(args, id)
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT DATE_FORMAT(item.delivery_at, '%Y-%m-%d'), o.id, o.order_no,
			c.id, c.name, p.id, p.product_code, p.name,
			CASE WHEN item.deliveryStatus = 'NOT_SHIPPED' THEN item.quantity ELSE 0 END,
			item.deliveryStatus
		FROM order_items AS item
		JOIN orders AS o ON o.id = item.order_id
		JOIN customers AS c ON c.id = o.customer_id
		JOIN products AS p ON p.id = item.product_id
		WHERE item.delivery_at >= ? AND item.delivery_at < ?
			AND o.deleted_at IS NULL
			AND o.salesperson_id IN (`+placeholders(len(salespersonIDs))+`)
		ORDER BY item.delivery_at, o.order_no, item.product_id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]DeliveryRow, 0)
	for rows.Next() {
		var item DeliveryRow
		if err := rows.Scan(&item.DeliveryDate, &item.OrderID, &item.OrderNo,
			&item.CustomerID, &item.CustomerName, &item.ProductID, &item.ProductCode,
			&item.ProductName, &item.QuantityToDeliver, &item.DeliveryStatus); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
