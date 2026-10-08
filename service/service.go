package service

import (
	"context"
	"errors"
	"time"

	"orders_backend/repository"
)

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrForbidden       = errors.New("forbidden")
)

type Repository interface {
	Authenticate(context.Context, string) (repository.Principal, bool, error)
	ListSalespersons(context.Context, int64) ([]repository.Salesperson, error)
	ListCustomers(context.Context, int64) ([]repository.Customer, error)
	ListProducts(context.Context) ([]repository.Product, error)
	LatestProductPrice(context.Context, int64) (repository.ProductPrice, error)
	ListOrders(context.Context, repository.OrderFilter) (repository.OrderList, error)
	LoadOrder(context.Context, int64, []int64) (repository.OrderDetail, error)
	ListDeliverySchedule(context.Context, []int64, time.Time, time.Time) ([]repository.DeliveryRow, error)
	CreateOrder(context.Context, repository.OrderDraft) (int64, error)
	UpdateOrder(context.Context, int64, []int64, repository.OrderDraft) error
	DeleteOrder(context.Context, int64, []int64) error
}

type Service struct {
	repository Repository
}

func New(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Authenticate(ctx context.Context, token string) (repository.Principal, error) {
	principal, valid, err := s.repository.Authenticate(ctx, token)
	if err != nil {
		return repository.Principal{}, err
	}
	if !valid {
		return repository.Principal{}, ErrUnauthenticated
	}
	if len(principal.SalespersonIDs) == 0 {
		return repository.Principal{}, ErrForbidden
	}
	return principal, nil
}

func (s *Service) Salespersons(ctx context.Context, principal repository.Principal) ([]repository.Salesperson, error) {
	return s.repository.ListSalespersons(ctx, principal.TokenID)
}

func (s *Service) Customers(ctx context.Context, principal repository.Principal, salespersonID int64) ([]repository.Customer, error) {
	return s.repository.ListCustomers(ctx, salespersonID)
}

func (s *Service) Products(ctx context.Context) ([]repository.Product, error) {
	return s.repository.ListProducts(ctx)
}

func (s *Service) ProductPrice(ctx context.Context, productID int64) (repository.ProductPrice, error) {
	return s.repository.LatestProductPrice(ctx, productID)
}

func (s *Service) Orders(ctx context.Context, principal repository.Principal, filter repository.OrderFilter) (repository.OrderList, error) {
	return s.repository.ListOrders(ctx, filter)
}

func (s *Service) Order(ctx context.Context, principal repository.Principal, orderID int64) (repository.OrderDetail, error) {
	return s.repository.LoadOrder(ctx, orderID, principal.SalespersonIDs)
}

func (s *Service) DeliverySchedule(ctx context.Context, principal repository.Principal, start, end time.Time) ([]repository.DeliveryRow, error) {
	return s.repository.ListDeliverySchedule(ctx, principal.SalespersonIDs, start, end)
}

func hasSalesperson(principal repository.Principal, id int64) bool {
	for _, allowed := range principal.SalespersonIDs {
		if allowed == id {
			return true
		}
	}
	return false
}
