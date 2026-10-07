package service

import "testing"

func TestParseMoneyRoundsHalfUp(t *testing.T) {
	tests := []struct {
		value string
		want  int64
	}{
		{value: "1.004", want: 100},
		{value: "1.005", want: 101},
		{value: "15000.00", want: 1500000},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			got, err := ParseMoney(test.value, true)
			if err != nil {
				t.Fatalf("ParseMoney(%q) returned error: %v", test.value, err)
			}
			if got != test.want {
				t.Fatalf("ParseMoney(%q) = %d, want %d", test.value, got, test.want)
			}
		})
	}
}

func TestNormalizeOrderRejectsDuplicateProductAndDeliveryDate(t *testing.T) {
	request := OrderRequest{
		OrderDate:       "2026-10-06",
		SalespersonID:   1,
		CustomerID:      1,
		FreightCharge:   "0.00",
		InsuranceCharge: "0.00",
		Items: []OrderItemInput{
			{ProductID: 1, DeliveryDate: "2026-10-15", Quantity: 1, UnitPrice: "150.00"},
			{ProductID: 1, DeliveryDate: "2026-10-15", Quantity: 2, UnitPrice: "150.00"},
		},
	}
	_, fields := NormalizeOrder(request)
	if len(fields) != 1 || fields[0].Field != "items[1]" {
		t.Fatalf("NormalizeOrder returned fields %#v, want duplicate item validation", fields)
	}
}
