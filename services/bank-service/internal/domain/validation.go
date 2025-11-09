package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	// Regex pattern for validating decimal amounts with up to 2 decimal places
	amountPattern = regexp.MustCompile(`^\d+(\.\d{1,2})?$`)
)

// ValidateAmount validates that an amount string is properly formatted.
// Returns an error if the amount is invalid.
func ValidateAmount(value string) error {
	if value == "" {
		return fmt.Errorf("amount value cannot be empty")
	}

	if !amountPattern.MatchString(value) {
		return fmt.Errorf("invalid amount format: must be a positive decimal with up to 2 decimal places")
	}

	// Parse to float to validate it's a valid number
	floatVal, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}

	if floatVal <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	return nil
}

// CompareAmounts compares two amount values.
// Returns:
//   - negative if a < b
//   - zero if a == b
//   - positive if a > b
//
// Note: This is a simplified comparison. For production use, consider using
// a proper decimal library like shopspring/decimal.
func CompareAmounts(a, b string) (int, error) {
	aFloat, err := strconv.ParseFloat(a, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount a: %w", err)
	}

	bFloat, err := strconv.ParseFloat(b, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount b: %w", err)
	}

	if aFloat < bFloat {
		return -1, nil
	} else if aFloat > bFloat {
		return 1, nil
	}
	return 0, nil
}

// SubtractAmounts subtracts b from a and returns the result.
// Note: This is a simplified implementation. For production use, consider using
// a proper decimal library like shopspring/decimal to avoid floating point precision issues.
func SubtractAmounts(a, b string) (string, error) {
	aFloat, err := strconv.ParseFloat(a, 64)
	if err != nil {
		return "", fmt.Errorf("invalid amount a: %w", err)
	}

	bFloat, err := strconv.ParseFloat(b, 64)
	if err != nil {
		return "", fmt.Errorf("invalid amount b: %w", err)
	}

	result := aFloat - bFloat
	return formatAmount(result), nil
}

// AddAmounts adds a and b and returns the result.
// Note: This is a simplified implementation. For production use, consider using
// a proper decimal library like shopspring/decimal to avoid floating point precision issues.
func AddAmounts(a, b string) (string, error) {
	aFloat, err := strconv.ParseFloat(a, 64)
	if err != nil {
		return "", fmt.Errorf("invalid amount a: %w", err)
	}

	bFloat, err := strconv.ParseFloat(b, 64)
	if err != nil {
		return "", fmt.Errorf("invalid amount b: %w", err)
	}

	result := aFloat + bFloat
	return formatAmount(result), nil
}

// formatAmount formats a float64 as a string with 2 decimal places.
func formatAmount(value float64) string {
	formatted := fmt.Sprintf("%.2f", value)
	// Remove trailing zeros after decimal point, but keep at least 2 decimal places
	if strings.Contains(formatted, ".") {
		formatted = strings.TrimRight(formatted, "0")
		formatted = strings.TrimRight(formatted, ".")
		// Ensure at least 2 decimal places
		parts := strings.Split(formatted, ".")
		if len(parts) == 1 {
			formatted += ".00"
		} else if len(parts[1]) == 1 {
			formatted += "0"
		}
	}
	return formatted
}

// ValidateCurrencyCode validates that a currency code follows ISO 4217 format.
func ValidateCurrencyCode(code string) error {
	if code == "" {
		return fmt.Errorf("currency code cannot be empty")
	}

	if len(code) != 3 {
		return fmt.Errorf("currency code must be 3 characters (ISO 4217)")
	}

	// Check if all characters are uppercase letters
	for _, c := range code {
		if c < 'A' || c > 'Z' {
			return fmt.Errorf("currency code must contain only uppercase letters")
		}
	}

	return nil
}
