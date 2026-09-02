package service

import (
	"errors"
	"strings"
	"time"

	"minimarket/buySell-app/internal/model"
)

var (
	ErrInvalidCardNumber     = errors.New("invalid card number")
	ErrUnsupportedCard       = errors.New("unsupported payment system")
	ErrInvalidExpirationDate = errors.New("invalid expiration date")
	ErrExpiredCard           = errors.New("card is expired")
	ErrInvalidCVV            = errors.New("cvv must contain 3 digits")
)

type validatedCard struct {
	paymentSystem string
	lastFour      string
}

func isDigits(number string) bool {
	if number == "" {
		return false
	}
	for _, r := range number {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func paymentSystem(number string) (string, bool) {
	switch {
	case strings.HasPrefix(number, "4") &&
		(len(number) == 13 || len(number) == 16 || len(number) == 19):
		return "visa", true
	case isMastercard(number):
		return "mastercard", true
	case len(number) == 16 && len(number) >= 4:
		prefix := number[:4]
		if prefix >= "2200" && prefix <= "2204" {
			return "mir", true
		}
	}
	return "", false
}

func isMastercard(number string) bool {
	if len(number) != 16 {
		return false
	}
	firstTwo := number[:2]
	if firstTwo >= "51" && firstTwo <= "55" {
		return true
	}
	firstFour := number[:4]
	return firstFour >= "2221" && firstFour <= "2720"
}

func luhnCheck(number string) bool {
	sum := 0
	double := false

	for i := len(number) - 1; i >= 0; i-- {
		digit := int(number[i] - '0')

		if double {
			digit *= 2

			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		double = !double
	}

	return sum%10 == 0
}

func validateCard(card model.Card, now time.Time) (validatedCard, error) {
	number := strings.NewReplacer(" ", "", "-", "").Replace(card.CardNumber)
	if len(number) < 13 || len(number) > 19 || !isDigits(number) || !luhnCheck(number) {
		return validatedCard{}, ErrInvalidCardNumber
	}

	system, ok := paymentSystem(number)
	if !ok {
		return validatedCard{}, ErrUnsupportedCard
	}

	if card.ExpMonth < 1 || card.ExpMonth > 12 || card.ExpYear < 1 {
		return validatedCard{}, ErrInvalidExpirationDate
	}
	if card.ExpYear < now.Year() ||
		(card.ExpYear == now.Year() && card.ExpMonth < int(now.Month())) {
		return validatedCard{}, ErrExpiredCard
	}

	if len(card.CVV) != 3 || !isDigits(card.CVV) {
		return validatedCard{}, ErrInvalidCVV
	}

	return validatedCard{
		paymentSystem: system,
		lastFour:      number[len(number)-4:],
	}, nil
}
