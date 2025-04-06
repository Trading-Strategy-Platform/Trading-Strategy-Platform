package validator

import (
	"errors"
	"fmt"
	"strconv"

	"services/strategy-service/internal/model"
)

// ValidateStrategyStructure validates the structure of a strategy
func ValidateStrategyStructure(structure *model.Structure) error {
	if len(structure.BuyRules) == 0 {
		return errors.New("buy rules cannot be empty")
	}

	if len(structure.SellRules) == 0 {
		return errors.New("sell rules cannot be empty")
	}

	// Validate buy rules
	if err := validateRules(structure.BuyRules); err != nil {
		return fmt.Errorf("invalid buy rules: %w", err)
	}

	// Validate sell rules
	if err := validateRules(structure.SellRules); err != nil {
		return fmt.Errorf("invalid sell rules: %w", err)
	}

	return nil
}

// validateRules validates an array of rules
func validateRules(rules []model.Rule) error {
	for i, rule := range rules {
		if err := validateRule(rule); err != nil {
			return fmt.Errorf("rule %d: %w", i+1, err)
		}
	}
	return nil
}

// validateRule validates a single rule or group of rules
func validateRule(rule model.Rule) error {
	// Validate rule type
	if rule.Type != "rule" && rule.Type != "group" {
		return fmt.Errorf("invalid rule type: %s", rule.Type)
	}

	// Validate operator
	if rule.Operator != "AND" && rule.Operator != "OR" {
		return fmt.Errorf("invalid operator: %s", rule.Operator)
	}

	// Validate rule-specific fields
	if rule.Type == "rule" {
		return validateSingleRule(rule)
	} else if rule.Type == "group" {
		return validateRuleGroup(rule)
	}

	return nil
}

// validateSingleRule validates a single rule
func validateSingleRule(rule model.Rule) error {
	if rule.Indicator == nil {
		return errors.New("indicator is required for rule type")
	}

	if rule.Indicator.ID == "" {
		return errors.New("indicator ID is required")
	}

	if rule.Indicator.Name == "" {
		return errors.New("indicator name is required")
	}

	if rule.Condition == nil {
		return errors.New("condition is required for rule type")
	}

	if rule.Condition.Symbol == "" {
		return errors.New("comparison symbol is required")
	}

	if !isValidComparisonSymbol(rule.Condition.Symbol) {
		return fmt.Errorf("invalid comparison symbol: %s", rule.Condition.Symbol)
	}

	if rule.Value == "" {
		return errors.New("comparison value is required")
	}

	// Try to parse the value as a number for non-string comparison validation
	if rule.Condition.Symbol != "==" && rule.Condition.Symbol != "!=" {
		if _, err := strconv.ParseFloat(rule.Value, 64); err != nil {
			return fmt.Errorf("invalid numeric value: %s", rule.Value)
		}
	}

	// Validate indicator settings
	if err := validateIndicatorSettings(rule.Indicator.Name, rule.IndicatorSettings); err != nil {
		return err
	}

	return nil
}

// validateRuleGroup validates a group of rules
func validateRuleGroup(rule model.Rule) error {
	if len(rule.Rules) == 0 {
		return errors.New("rule group cannot be empty")
	}

	// Validate nested rules
	for i, nestedRule := range rule.Rules {
		if err := validateRule(nestedRule); err != nil {
			return fmt.Errorf("nested rule %d: %w", i+1, err)
		}
	}

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
