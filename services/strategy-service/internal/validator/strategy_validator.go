// services/strategy-service/internal/validator/strategy_validator.go
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

// validateIndicatorSettings validates settings for specific indicators
func validateIndicatorSettings(indicatorName string, settings map[string]interface{}) error {
	switch indicatorName {
	case "RSI":
		return validateRSISettings(settings)
	case "Bollinger Bands":
		return validateBollingerBandsSettings(settings)
	case "MACD":
		return validateMACDSettings(settings)
	case "Moving Average":
		return validateMovingAverageSettings(settings)
	case "Stochastic":
		return validateStochasticSettings(settings)
	default:
		// For unknown indicators, we'll be permissive since they might be custom
		return nil
	}
}

// validateRSISettings validates RSI indicator settings
func validateRSISettings(settings map[string]interface{}) error {
	// Check required parameters
	period, ok := settings["period"]
	if !ok {
		return errors.New("RSI requires 'period' parameter")
	}

	// Validate period
	periodFloat, ok := period.(float64)
	if !ok {
		return errors.New("RSI period must be a number")
	}

	if periodFloat < 2 || periodFloat > 100 {
		return errors.New("RSI period must be between 2 and 100")
	}

	return nil
}

// validateBollingerBandsSettings validates Bollinger Bands indicator settings
func validateBollingerBandsSettings(settings map[string]interface{}) error {
	// Check required parameters
	period, periodOk := settings["period"]
	deviations, deviationsOk := settings["deviations"]

	if !periodOk {
		return errors.New("Bollinger Bands requires 'period' parameter")
	}

	if !deviationsOk {
		return errors.New("Bollinger Bands requires 'deviations' parameter")
	}

	// Validate period
	periodFloat, ok := period.(float64)
	if !ok {
		return errors.New("Bollinger Bands period must be a number")
	}

	if periodFloat < 2 || periodFloat > 100 {
		return errors.New("Bollinger Bands period must be between 2 and 100")
	}

	// Validate deviations
	deviationsFloat, ok := deviations.(float64)
	if !ok {
		return errors.New("Bollinger Bands deviations must be a number")
	}

	if deviationsFloat < 0.1 || deviationsFloat > 5 {
		return errors.New("Bollinger Bands deviations must be between 0.1 and 5")
	}

	return nil
}

// validateMACDSettings validates MACD indicator settings
func validateMACDSettings(settings map[string]interface{}) error {
	// Check required parameters
	fastPeriod, fastOk := settings["fastPeriod"]
	slowPeriod, slowOk := settings["slowPeriod"]
	signalPeriod, signalOk := settings["signalPeriod"]

	if !fastOk {
		return errors.New("MACD requires 'fastPeriod' parameter")
	}

	if !slowOk {
		return errors.New("MACD requires 'slowPeriod' parameter")
	}

	if !signalOk {
		return errors.New("MACD requires 'signalPeriod' parameter")
	}

	// Validate fast period
	fastFloat, ok := fastPeriod.(float64)
	if !ok {
		return errors.New("MACD fastPeriod must be a number")
	}

	if fastFloat < 2 || fastFloat > 100 {
		return errors.New("MACD fastPeriod must be between 2 and 100")
	}

	// Validate slow period
	slowFloat, ok := slowPeriod.(float64)
	if !ok {
		return errors.New("MACD slowPeriod must be a number")
	}

	if slowFloat < 2 || slowFloat > 100 {
		return errors.New("MACD slowPeriod must be between 2 and 100")
	}

	if slowFloat <= fastFloat {
		return errors.New("MACD slowPeriod must be greater than fastPeriod")
	}

	// Validate signal period
	signalFloat, ok := signalPeriod.(float64)
	if !ok {
		return errors.New("MACD signalPeriod must be a number")
	}

	if signalFloat < 2 || signalFloat > 100 {
		return errors.New("MACD signalPeriod must be between 2 and 100")
	}

	return nil
}

// validateMovingAverageSettings validates Moving Average indicator settings
func validateMovingAverageSettings(settings map[string]interface{}) error {
	// Check required parameters
	period, periodOk := settings["period"]
	maType, maTypeOk := settings["type"]

	if !periodOk {
		return errors.New("Moving Average requires 'period' parameter")
	}

	// Validate period
	periodFloat, ok := period.(float64)
	if !ok {
		return errors.New("Moving Average period must be a number")
	}

	if periodFloat < 2 || periodFloat > 200 {
		return errors.New("Moving Average period must be between 2 and 200")
	}

	// Validate MA type if provided
	if maTypeOk {
		maTypeStr, ok := maType.(string)
		if !ok {
			return errors.New("Moving Average type must be a string")
		}

		validTypes := map[string]bool{
			"sma":   true, // Simple Moving Average
			"ema":   true, // Exponential Moving Average
			"wma":   true, // Weighted Moving Average
			"dema":  true, // Double Exponential Moving Average
			"tema":  true, // Triple Exponential Moving Average
			"trima": true, // Triangular Moving Average
			"kama":  true, // Kaufman Adaptive Moving Average
			"mama":  true, // MESA Adaptive Moving Average
		}

		if !validTypes[maTypeStr] {
			return fmt.Errorf("invalid Moving Average type: %s", maTypeStr)
		}
	}

	return nil
}

// validateStochasticSettings validates Stochastic indicator settings
func validateStochasticSettings(settings map[string]interface{}) error {
	// Check required parameters
	kPeriod, kOk := settings["kPeriod"]
	dPeriod, dOk := settings["dPeriod"]
	slowing, slowingOk := settings["slowing"]

	if !kOk {
		return errors.New("Stochastic requires 'kPeriod' parameter")
	}

	if !dOk {
		return errors.New("Stochastic requires 'dPeriod' parameter")
	}

	if !slowingOk {
		return errors.New("Stochastic requires 'slowing' parameter")
	}

	// Validate kPeriod
	kFloat, ok := kPeriod.(float64)
	if !ok {
		return errors.New("Stochastic kPeriod must be a number")
	}

	if kFloat < 1 || kFloat > 100 {
		return errors.New("Stochastic kPeriod must be between 1 and 100")
	}

	// Validate dPeriod
	dFloat, ok := dPeriod.(float64)
	if !ok {
		return errors.New("Stochastic dPeriod must be a number")
	}

	if dFloat < 1 || dFloat > 100 {
		return errors.New("Stochastic dPeriod must be between 1 and 100")
	}

	// Validate slowing
	slowingFloat, ok := slowing.(float64)
	if !ok {
		return errors.New("Stochastic slowing must be a number")
	}

	if slowingFloat < 1 || slowingFloat > 100 {
		return errors.New("Stochastic slowing must be between 1 and 100")
	}

	return nil
}
