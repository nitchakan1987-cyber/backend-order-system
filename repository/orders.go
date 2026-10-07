package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrOrderNotFound = errors.New("order not found")

var (
	ErrCustomerAccess     = errors.New("customer access denied")
	ErrDomesticCharges    = errors.New("domestic customer charges must be zero")
	ErrProductNotFound    = errors.New("product not found")
	ErrPriceChanged       = errors.New("product price changed")
	ErrOrderShipped       = errors.New("order already shipped")
	ErrInvalidOrderItem   = errors.New("invalid order item")
	ErrOrderTotalOverflow = errors.New("order total exceeds supported amount")
)

type OrderDraft struct {
	OrderDate       time.Time
	SalespersonID   int64
	CustomerID      int64
	FreightCents    int64
	InsuranceCents  int64
	SubtotalCents   int64
	AdditionalCents int64
	GrandTotalCents int64
	Items           []OrderDraftItem
}

type OrderDraftItem struct {
	ID             int64
	ProductID      int64
	DeliveryDate   time.Time
	Quantity       int64
	UnitPriceCents int64
	LineTotalCents int64
}

type OrderFilter struct {
	StartDate      time.Time
	EndDate        time.Time
	SalespersonID  int64
	CustomerID     int64
	SalespersonIDs []int64
	Page           int
	PageSize       int
}

type OrderSummary struct {
	OrderCount  int64  `json:"orderCount"`
	TotalAmount string `json:"totalAmount"`
	Currency    string `json:"currency"`
}

type OrderListItem struct {
	ID              int64        `json:"id"`
	OrderNo         string       `json:"orderNo"`
	OrderDate       string       `json:"orderDate"`
	CustomerID      int64        `json:"customerId"`
	CustomerName    string       `json:"customerName"`
	SalespersonID   int64        `json:"salespersonId"`
	SalespersonName string       `json:"salespersonName"`
	GrandTotal      string       `json:"grandTotal"`
	DeliveryStatus  string       `json:"deliveryStatus"`
	Actions         OrderActions `json:"actions"`
}

type OrderActions struct {
	CanView   bool `json:"canView"`
	CanEdit   bool `json:"canEdit"`
	CanDelete bool `json:"canDelete"`
}

type OrderList struct {
	Summary OrderSummary    `json:"summary"`
	Items   []OrderListItem `json:"items"`
	Meta    OrderPageMeta   `json:"meta"`
}

type OrderPageMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int64 `json:"totalPages"`
}

type OrderItem struct {
	ID           int64  `json:"id"`
	ProductID    int64  `json:"productId"`
	ProductCode  string `json:"productCode"`
	ProductName  string `json:"productName"`
	DeliveryDate string `json:"deliveryDate"`
	Quantity     int64  `json:"quantity"`
	UnitPrice    string `json:"unitPrice"`
	LineTotal    string `json:"lineTotal"`
}

type OrderDetail struct {
	ID                int64       `json:"id"`
	OrderNo           string      `json:"orderNo"`
	OrderDate         string      `json:"orderDate"`
	SalespersonID     int64       `json:"salespersonId"`
	CustomerID        int64       `json:"customerId"`
	DeliveryStatus    string      `json:"deliveryStatus"`
	Items             []OrderItem `json:"items"`
	FreightCharge     string      `json:"freightCharge"`
	InsuranceCharge   string      `json:"insuranceCharge"`
	Subtotal          string      `json:"subtotal"`
	AdditionalCharges string      `json:"additionalCharges"`
	GrandTotal        string      `json:"grandTotal"`
	Currency          string      `json:"currency"`
}

func (r *MySQL) ListOrders(ctx context.Context, filter OrderFilter) (OrderList, error) {
	filters := `o.order_date >= ? AND o.order_date < ? AND o.deleted_at IS NULL AND o.salesperson_id IN (` + placeholders(len(filter.SalespersonIDs)) + `)`
	filterArgs := []any{filter.StartDate.Format("2006-01-02"), filter.EndDate.AddDate(0, 0, 1).Format("2006-01-02")}
	for _, id := range filter.SalespersonIDs {
		filterArgs = append(filterArgs, id)
	}
	if filter.SalespersonID > 0 {
		filters += " AND o.salesperson_id = ?"
		filterArgs = append(filterArgs, filter.SalespersonID)
	}
	if filter.CustomerID > 0 {
		filters += " AND o.customer_id = ?"
		filterArgs = append(filterArgs, filter.CustomerID)
	}

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return OrderList{}, err
	}
	defer tx.Rollback()

	result := OrderList{
		Summary: OrderSummary{Currency: "THB"},
		Items:   make([]OrderListItem, 0),
		Meta:    OrderPageMeta{Page: filter.Page, PageSize: filter.PageSize},
	}
	totalByOrder := `SELECT order_id, SUM(quantity * price) AS subtotal FROM order_items GROUP BY order_id`
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*), CAST(COALESCE(SUM(COALESCE(item_totals.subtotal, 0) + COALESCE(o.freight_charge, 0) + COALESCE(o.insurance_charge, 0)), 0.00) AS CHAR)
		FROM orders AS o
		LEFT JOIN (`+totalByOrder+`) AS item_totals ON item_totals.order_id = o.id
		WHERE `+filters, filterArgs...).
		Scan(&result.Summary.OrderCount, &result.Summary.TotalAmount); err != nil {
		return OrderList{}, err
	}
	result.Meta.TotalItems = result.Summary.OrderCount
	if result.Meta.TotalItems > 0 {
		result.Meta.TotalPages = (result.Meta.TotalItems + int64(filter.PageSize) - 1) / int64(filter.PageSize)
	}

	args := append(append([]any(nil), filterArgs...), filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := tx.QueryContext(ctx, `
		SELECT o.id, o.order_no, COALESCE(DATE_FORMAT(o.order_date, '%Y-%m-%d'), ''),
			o.customer_id, c.name, o.salesperson_id, sp.name,
			CAST(COALESCE(item_totals.subtotal, 0) + COALESCE(o.freight_charge, 0) + COALESCE(o.insurance_charge, 0) AS CHAR),
			CASE WHEN EXISTS (
				SELECT 1 FROM order_items AS shipped
				WHERE shipped.order_id = o.id AND shipped.deliveryStatus = 'SHIPPED'
			) THEN 'SHIPPED' ELSE 'NOT_SHIPPED' END
		FROM orders AS o
		JOIN customers AS c ON c.id = o.customer_id
		JOIN salesperson AS sp ON sp.id = o.salesperson_id
		LEFT JOIN (`+totalByOrder+`) AS item_totals ON item_totals.order_id = o.id
		WHERE `+filters+`
		ORDER BY o.order_date DESC, o.id DESC
		LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return OrderList{}, err
	}
	for rows.Next() {
		var item OrderListItem
		if err := rows.Scan(&item.ID, &item.OrderNo, &item.OrderDate, &item.CustomerID,
			&item.CustomerName, &item.SalespersonID, &item.SalespersonName, &item.GrandTotal,
			&item.DeliveryStatus); err != nil {
			rows.Close()
			return OrderList{}, err
		}
		item.Actions = OrderActions{CanView: true, CanEdit: item.DeliveryStatus == "NOT_SHIPPED", CanDelete: item.DeliveryStatus == "NOT_SHIPPED"}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return OrderList{}, err
	}
	rows.Close()
	if err := tx.Commit(); err != nil {
		return OrderList{}, err
	}
	return result, nil
}

func (r *MySQL) LoadOrder(ctx context.Context, orderID int64, salespersonIDs []int64) (OrderDetail, error) {
	var detail OrderDetail
	args := []any{orderID}
	for _, id := range salespersonIDs {
		args = append(args, id)
	}
	err := r.db.QueryRowContext(ctx, `
		SELECT o.id, o.order_no, COALESCE(DATE_FORMAT(o.order_date, '%Y-%m-%d'), ''), o.salesperson_id,
			o.customer_id,
			CASE WHEN EXISTS (
				SELECT 1 FROM order_items AS shipped
				WHERE shipped.order_id = o.id AND shipped.deliveryStatus = 'SHIPPED'
			) THEN 'SHIPPED' ELSE 'NOT_SHIPPED' END,
			CAST(COALESCE(o.freight_charge, 0) AS CHAR), CAST(COALESCE(o.insurance_charge, 0) AS CHAR),
			CAST(COALESCE((SELECT SUM(item.quantity * item.price) FROM order_items AS item WHERE item.order_id = o.id), 0) AS CHAR),
			CAST(COALESCE(o.freight_charge, 0) + COALESCE(o.insurance_charge, 0) AS CHAR),
			CAST(COALESCE((SELECT SUM(item.quantity * item.price) FROM order_items AS item WHERE item.order_id = o.id), 0) + COALESCE(o.freight_charge, 0) + COALESCE(o.insurance_charge, 0) AS CHAR)
		FROM orders AS o
		WHERE o.id = ? AND o.deleted_at IS NULL AND o.salesperson_id IN (`+placeholders(len(salespersonIDs))+`)`, args...).
		Scan(&detail.ID, &detail.OrderNo, &detail.OrderDate, &detail.SalespersonID, &detail.CustomerID,
			&detail.DeliveryStatus, &detail.FreightCharge, &detail.InsuranceCharge,
			&detail.Subtotal, &detail.AdditionalCharges, &detail.GrandTotal)
	if errors.Is(err, sql.ErrNoRows) {
		return OrderDetail{}, ErrOrderNotFound
	}
	if err != nil {
		return OrderDetail{}, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT item.product_id, item.product_id, product.product_code, product.name,
			COALESCE(DATE_FORMAT(item.delivery_at, '%Y-%m-%d'), ''), item.quantity,
			CAST(item.price AS CHAR), CAST(item.quantity * item.price AS CHAR)
		FROM order_items AS item
		JOIN products AS product ON product.id = item.product_id
		WHERE item.order_id = ?
		ORDER BY item.delivery_at, item.product_id`, orderID)
	if err != nil {
		return OrderDetail{}, err
	}
	detail.Items = make([]OrderItem, 0)
	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(&item.ID, &item.ProductID, &item.ProductCode, &item.ProductName,
			&item.DeliveryDate, &item.Quantity, &item.UnitPrice, &item.LineTotal); err != nil {
			rows.Close()
			return OrderDetail{}, err
		}
		detail.Items = append(detail.Items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return OrderDetail{}, err
	}
	rows.Close()
	detail.Currency = "THB"
	return detail, nil
}
