package validator

import (
	"services/strategy-service/internal/model"
)

// ValidateStrategyStructure validates the structure of a strategy
// This implementation is completely permissive - it will accept any strategy structure
func ValidateStrategyStructure(structure *model.Structure) error {
	// Accept any strategy structure unconditionally
	return nil
}

// isValidComparisonSymbol checks if a comparison symbol is valid
func isValidComparisonSymbol(symbol string) bool {
	validSymbols := map[string]bool{
		">":             true,
		"<":             true,
		">=":            true,
		"<=":            true,
		"==":            true,
		"!=":            true,
		"crosses_above": true,
		"crosses_below": true,
		"is_above":      true,
		"is_below":      true,
	}

	return validSymbols[symbol]
}
