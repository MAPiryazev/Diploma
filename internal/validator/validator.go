package validator

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/errors"
)

func ValidateEmail(email string) error {
	const emailPattern = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	if !regexp.MustCompile(emailPattern).MatchString(email) {
		return &apperrors.ValidationError{Field: "email", Message: "invalid format"}
	}
	return nil
}

func ValidateUserName(name string) error {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 255 {
		return &apperrors.ValidationError{Field: "name", Message: "length must be 2-255 characters"}
	}
	return nil
}

func ValidateAccountNumber(number string) error {
	number = strings.TrimSpace(number)
	if len(number) < 5 || len(number) > 50 {
		return &apperrors.ValidationError{Field: "number", Message: "length must be 5-50 characters"}
	}
	return nil
}

func ValidateCategoryName(name string) error {
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 255 {
		return &apperrors.ValidationError{Field: "name", Message: "length must be 1-255 characters"}
	}
	return nil
}

func ValidateProviderName(name string) error {
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 255 {
		return &apperrors.ValidationError{Field: "name", Message: "length must be 1-255 characters"}
	}
	return nil
}

func ValidateTransactionAmount(amount string) error {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return &apperrors.ValidationError{Field: "amount", Message: "required"}
	}

	val, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return &apperrors.ValidationError{Field: "amount", Message: "must be numeric"}
	}

	if val <= 0 {
		return &apperrors.ValidationError{Field: "amount", Message: "must be greater than 0"}
	}

	if val > 999999999.99 {
		return &apperrors.ValidationError{Field: "amount", Message: "exceeds maximum"}
	}

	return nil
}

func ValidateTransactionType(txType string) error {
	txType = strings.TrimSpace(txType)
	validTypes := map[string]bool{
		"income":   true,
		"expense":  true,
		"transfer": true,
	}
	if !validTypes[txType] {
		return &apperrors.ValidationError{Field: "type", Message: "must be income, expense or transfer"}
	}
	return nil
}

func ValidateTransactionStatus(status string) error {
	status = strings.TrimSpace(status)
	validStatuses := map[string]bool{
		"pending": true,
		"done":    true,
		"failed":  true,
	}
	if !validStatuses[status] {
		return &apperrors.ValidationError{Field: "status", Message: "must be pending, done or failed"}
	}
	return nil
}

func ValidateCurrency(currency string) error {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if len(currency) != 3 {
		return &apperrors.ValidationError{Field: "currency", Message: "must be 3-char code"}
	}
	if !regexp.MustCompile(`^[A-Z]{3}$`).MatchString(currency) {
		return &apperrors.ValidationError{Field: "currency", Message: "must contain only letters"}
	}
	return nil
}

func ValidateTimestamp(ts string) error {
	ts = strings.TrimSpace(ts)
	if ts == "" {
		return &apperrors.ValidationError{Field: "occurred_at", Message: "required"}
	}

	_, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return &apperrors.ValidationError{Field: "occurred_at", Message: "invalid RFC3339 format"}
	}
	return nil
}

func ValidateDateRange(from, to string) error {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)

	if from == "" || to == "" {
		return &apperrors.ValidationError{Field: "date_range", Message: "from and to are required"}
	}

	fromTime, err := time.Parse(time.RFC3339, from)
	if err != nil {
		return &apperrors.ValidationError{Field: "from", Message: "invalid RFC3339 format"}
	}

	toTime, err := time.Parse(time.RFC3339, to)
	if err != nil {
		return &apperrors.ValidationError{Field: "to", Message: "invalid RFC3339 format"}
	}

	if fromTime.After(toTime) {
		return &apperrors.ValidationError{Field: "date_range", Message: "from must be before to"}
	}

	if toTime.Sub(fromTime).Hours() > 87600 { // ~10 years
		return &apperrors.ValidationError{Field: "date_range", Message: "range too large"}
	}

	return nil
}

func ValidateUUID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return &apperrors.ValidationError{Field: "id", Message: "required"}
	}

	const uuidPattern = `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
	if !regexp.MustCompile(uuidPattern).MatchString(id) {
		return &apperrors.ValidationError{Field: "id", Message: "invalid UUID format"}
	}
	return nil
}

func ValidateTransactionAccounts(txType, fromID, toID string) error {
	if txType == "transfer" {
		if fromID == "" {
			return &apperrors.ValidationError{Field: "from_account_id", Message: "required for transfer"}
		}
		if toID == "" {
			return &apperrors.ValidationError{Field: "to_account_id", Message: "required for transfer"}
		}
		if fromID == toID {
			return &apperrors.ValidationError{Field: "accounts", Message: "from and to accounts must be different"}
		}
		return nil
	}

	if txType == "income" {
		if toID == "" {
			return &apperrors.ValidationError{Field: "to_account_id", Message: "required for income"}
		}
	}

	if txType == "expense" {
		if fromID == "" {
			return &apperrors.ValidationError{Field: "from_account_id", Message: "required for expense"}
		}
	}

	return nil
}
