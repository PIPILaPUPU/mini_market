package service

import (
	"errors"
	"testing"
	"time"

	"minimarket/buySell-app/internal/model"
)

func TestValidateCard(t *testing.T) {
	now := time.Date(2026, time.September, 2, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		card     model.Card
		wantErr  error
		system   string
		lastFour string
	}{
		{
			name: "valid Visa with separators",
			card: model.Card{
				CardNumber: "4242 4242-4242 4242",
				ExpMonth:   9,
				ExpYear:    2026,
				CVV:        "123",
			},
			system:   "visa",
			lastFour: "4242",
		},
		{
			name: "valid Mastercard",
			card: model.Card{
				CardNumber: "5555555555554444",
				ExpMonth:   12,
				ExpYear:    2030,
				CVV:        "777",
			},
			system:   "mastercard",
			lastFour: "4444",
		},
		{
			name: "valid Mir",
			card: model.Card{
				CardNumber: "2202200222022004",
				ExpMonth:   12,
				ExpYear:    2030,
				CVV:        "123",
			},
			system:   "mir",
			lastFour: "2004",
		},
		{
			name: "invalid Luhn checksum",
			card: model.Card{
				CardNumber: "4242424242424241",
				ExpMonth:   12,
				ExpYear:    2030,
				CVV:        "123",
			},
			wantErr: ErrInvalidCardNumber,
		},
		{
			name: "number contains a letter",
			card: model.Card{
				CardNumber: "424242424242424x",
				ExpMonth:   12,
				ExpYear:    2030,
				CVV:        "123",
			},
			wantErr: ErrInvalidCardNumber,
		},
		{
			name: "unsupported payment system",
			card: model.Card{
				CardNumber: "6011111111111117",
				ExpMonth:   12,
				ExpYear:    2030,
				CVV:        "123",
			},
			wantErr: ErrUnsupportedCard,
		},
		{
			name: "invalid expiration month",
			card: model.Card{
				CardNumber: "4242424242424242",
				ExpMonth:   13,
				ExpYear:    2030,
				CVV:        "123",
			},
			wantErr: ErrInvalidExpirationDate,
		},
		{
			name: "expired card",
			card: model.Card{
				CardNumber: "4242424242424242",
				ExpMonth:   8,
				ExpYear:    2026,
				CVV:        "123",
			},
			wantErr: ErrExpiredCard,
		},
		{
			name: "CVV has invalid length",
			card: model.Card{
				CardNumber: "4242424242424242",
				ExpMonth:   12,
				ExpYear:    2030,
				CVV:        "12",
			},
			wantErr: ErrInvalidCVV,
		},
		{
			name: "CVV contains a letter",
			card: model.Card{
				CardNumber: "4242424242424242",
				ExpMonth:   12,
				ExpYear:    2030,
				CVV:        "12x",
			},
			wantErr: ErrInvalidCVV,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateCard(tt.card, now)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("validateCard() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got.paymentSystem != tt.system {
				t.Errorf("payment system = %q, want %q", got.paymentSystem, tt.system)
			}
			if got.lastFour != tt.lastFour {
				t.Errorf("last four = %q, want %q", got.lastFour, tt.lastFour)
			}
		})
	}
}
