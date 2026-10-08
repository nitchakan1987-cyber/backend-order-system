package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strings"
	"time"

	"orders_backend/repository"
)

const maxMoneyCents int64 = 999999999999999999

var (
	decimalPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`)
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	return "request validation failed"
}

type OrderRequest struct {
	OrderDate       string           `json:"orderDate"`
	SalespersonID   int64            `json:"salespersonId"`
	CustomerID      int64            `json:"customerId"`
	Items           []OrderItemInput `json:"items"`
	FreightCharge   string           `json:"freightCharge"`
	InsuranceCharge string           `json:"insuranceCharge"`
}

type OrderItemInput struct {
	ProductID    int64  `json:"productId"`
	DeliveryDate string `json:"deliveryDate"`
	Quantity     int64  `json:"quantity"`
	UnitPrice    string `json:"unitPrice"`
}

func (s *Service) CreateOrder(ctx context.Context, principal repository.Principal, request OrderRequest) (repository.OrderDetail, error) {
	draft, fields := NormalizeOrder(request)
	if len(fields) > 0 {
		return repository.OrderDetail{}, &ValidationError{Fields: fields}
	}
	orderID, err := s.repository.CreateOrder(ctx, draft)
	if err != nil {
		return repository.OrderDetail{}, err
	}
	return s.repository.LoadOrder(ctx, orderID)
}

func (s *Service) UpdateOrder(ctx context.Context, principal repository.Principal, orderID int64, request OrderRequest) (repository.OrderDetail, error) {
	draft, fields := NormalizeOrder(request)
	if len(fields) > 0 {
		return repository.OrderDetail{}, &ValidationError{Fields: fields}
	}
	if err := s.repository.UpdateOrder(ctx, orderID, principal.SalespersonIDs, draft); err != nil {
		return repository.OrderDetail{}, err
	}
	return s.repository.LoadOrder(ctx, orderID)
}

func (s *Service) DeleteOrder(ctx context.Context, principal repository.Principal, orderID int64) error {
	return s.repository.DeleteOrder(ctx, orderID, principal.SalespersonIDs)
}

func NormalizeOrder(request OrderRequest) (repository.OrderDraft, []FieldError) {
	var fields []FieldError
	date, dateErr := time.Parse("2006-01-02", request.OrderDate)
	if dateErr != nil {
		fields = append(fields, FieldError{Field: "orderDate", Message: "Required date in YYYY-MM-DD format"})
	}
	if request.SalespersonID <= 0 {
		fields = append(fields, FieldError{Field: "salespersonId", Message: "Must be a positive integer"})
	}
	if request.CustomerID <= 0 {
		fields = append(fields, FieldError{Field: "customerId", Message: "Must be a positive integer"})
	}
	freight, freightErr := ParseMoney(request.FreightCharge, false)
	if freightErr != nil {
		fields = append(fields, FieldError{Field: "freightCharge", Message: "Must be a non-negative decimal string"})
	}
	insurance, insuranceErr := ParseMoney(request.InsuranceCharge, false)
	if insuranceErr != nil {
		fields = append(fields, FieldError{Field: "insuranceCharge", Message: "Must be a non-negative decimal string"})
	}
	if len(request.Items) == 0 {
		fields = append(fields, FieldError{Field: "items", Message: "At least one item is required"})
	}
	draft := repository.OrderDraft{
		OrderDate: date, SalespersonID: request.SalespersonID, CustomerID: request.CustomerID,
		FreightCents: freight, InsuranceCents: insurance,
		Items: make([]repository.OrderDraftItem, 0, len(request.Items)),
	}
	seen := make(map[string]struct{}, len(request.Items))
	for index, item := range request.Items {
		prefix := fmt.Sprintf("items[%d]", index)
		if item.ProductID <= 0 {
			fields = append(fields, FieldError{Field: prefix + ".productId", Message: "Must be a positive integer"})
		}
		if item.Quantity <= 0 || item.Quantity > math.MaxInt32 {
			fields = append(fields, FieldError{Field: prefix + ".quantity", Message: "Must be a positive integer within the supported range"})
		}
		deliveryDate, itemDateErr := time.Parse("2006-01-02", item.DeliveryDate)
		if itemDateErr != nil {
			fields = append(fields, FieldError{Field: prefix + ".deliveryDate", Message: "Required date in YYYY-MM-DD format"})
		} else if dateErr == nil && deliveryDate.Before(date) {
			fields = append(fields, FieldError{Field: prefix + ".deliveryDate", Message: "Must not be before orderDate"})
		}
		if item.ProductID > 0 {
			key := fmt.Sprint(item.ProductID)
			if _, exists := seen[key]; exists {
				fields = append(fields, FieldError{Field: prefix, Message: "A product can appear only once in an order"})
			}
			seen[key] = struct{}{}
		}
		draft.Items = append(draft.Items, repository.OrderDraftItem{
			ProductID: item.ProductID, DeliveryDate: deliveryDate,
			Quantity: item.Quantity,
		})
	}
	if len(fields) > 0 {
		return draft, fields
	}
	additional := freight + insurance
	if additional > maxMoneyCents {
		return draft, []FieldError{{Field: "freightCharge", Message: "Total charges exceed the supported amount"}}
	}
	draft.AdditionalCents = additional
	return draft, nil
}

func ParseMoney(value string, positive bool) (int64, error) {
	value = strings.TrimSpace(value)
	if !decimalPattern.MatchString(value) {
		return 0, errors.New("invalid decimal")
	}
	rat, ok := new(big.Rat).SetString(value)
	if !ok || rat.Sign() < 0 || positive && rat.Sign() == 0 {
		return 0, errors.New("invalid amount")
	}
	scaled := new(big.Rat).Mul(rat, big.NewRat(100, 1))
	quotient := new(big.Int)
	remainder := new(big.Int)
	quotient.QuoRem(scaled.Num(), scaled.Denom(), remainder)
	if new(big.Int).Lsh(remainder, 1).Cmp(scaled.Denom()) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if !quotient.IsInt64() || quotient.Int64() > maxMoneyCents {
		return 0, errors.New("amount is too large")
	}
	return quotient.Int64(), nil
}
