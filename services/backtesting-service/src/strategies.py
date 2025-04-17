#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Strategy builder from JSON configuration.
Converts JSON strategy definitions to executable backtesting.py strategies.
"""

import logging
from typing import Dict, Any, Tuple, List, Optional, Callable, Type
import pandas as pd
from backtesting import Strategy

from src.indicators import calculate_indicator

logger = logging.getLogger(__name__)

class DynamicStrategy(Strategy):
    """
    A dynamic strategy class that can be configured with JSON.
    Uses backtesting.py's Strategy class as a base.
    """
    
    def init(self):
        """Initialize the strategy and calculate indicators."""
        # Setup indicators based on the strategy configuration
        self.buy_rules = self.strategy_config.get("buyRules", {})
        self.sell_rules = self.strategy_config.get("sellRules", {})
        
        # Process all indicators needed for the strategy
        self._process_rule_group(self.buy_rules)
        self._process_rule_group(self.sell_rules)
        
        # Define risk management parameters
        self.stop_loss_pct = self.params.get("stop_loss", 0)
        self.take_profit_pct = self.params.get("take_profit", 0)
        self.trailing_stop_pct = self.params.get("trailing_stop", 0)
        
        # Position sizing
        self.position_sizing = self.params.get("position_sizing", "fixed")
        self.risk_percentage = self.params.get("risk_percentage", 2.0)
    
    def _process_rule_group(self, rule_group: Dict[str, Any]) -> None:
        """Process a rule group recursively to identify and calculate indicators."""
        # Check if the rule group has a sequence
        if "_sequence" in rule_group:
            sequence = rule_group["_sequence"]
            for item in sequence:
                item_type = item.get("type")
                item_index = item.get("index")
                
                if item_type == "rule":
                    rule_key = f"rule{item_index}"
                    if rule_key in rule_group:
                        self._process_indicator(rule_group[rule_key].get("indicator", {}))
                
                elif item_type == "group":
                    group_key = f"group{item_index}"
                    if group_key in rule_group:
                        self._process_rule_group(rule_group[group_key])
        else:
            # Process each rule directly
            for key, value in rule_group.items():
                if key.startswith("rule"):
                    if "indicator" in value:
                        self._process_indicator(value["indicator"])
                elif key.startswith("group"):
                    self._process_rule_group(value)
    
    def _process_indicator(self, indicator_config: Dict[str, Any]) -> None:
        """Calculate a single indicator based on configuration."""
        indicator_name = indicator_config.get("name")
        settings = indicator_config.get("indicatorSettings", {})
        
        if not indicator_name:
            return
            
        # Calculate indicator using our dynamic indicator system
        # This adds the indicator values directly to the data
        calculate_indicator(self.data.df, indicator_name, settings)
    
    def _evaluate_rules(self, rules: Dict[str, Any], i: int) -> bool:
        """Evaluate rule structure against the current candle."""
        if "_sequence" in rules and any(k.startswith("operator") for k in rules):
            # Complex rule structure with sequence and operators
            return self._evaluate_complex_rules(rules, i)
        else:
            # Simple rule structure
            return self._evaluate_simple_rules(rules, i)
    
    def _evaluate_complex_rules(self, rules: Dict[str, Any], i: int) -> bool:
        """Evaluate complex rules with sequences and operators."""
        sequence = rules.get("_sequence", [])
        operators = {k: v for k, v in rules.items() if k.startswith("operator")}
        
        # Evaluate each item in the sequence
        results = []
        for item in sequence:
            item_type = item.get("type")
            item_index = item.get("index")
            
            if item_type == "rule":
                rule_key = f"rule{item_index}"
                if rule_key in rules:
                    results.append(self._evaluate_rule(rules[rule_key], i))
            
            elif item_type == "group":
                group_key = f"group{item_index}"
                if group_key in rules:
                    results.append(self._evaluate_rules(rules[group_key], i))
            
            elif item_type == "operator":
                # Operators are handled in the next step
                pass
        
        # Apply operators
        if not results:
            return False
            
        result = results[0]
        for j in range(1, len(results)):
            op_key = f"operator{j-1}"
            if op_key in rules:
                op = rules[op_key]
                if op == "AND":
                    result = result and results[j]
                elif op == "OR":
                    result = result or results[j]
        
        return result
    
    def _evaluate_simple_rules(self, rules: Dict[str, Any], i: int) -> bool:
        """Evaluate simple rules without sequence."""
        results = []
        operators = []
        
        # Extract rules and operators
        for key, value in rules.items():
            if key.startswith("rule"):
                results.append(self._evaluate_rule(value, i))
            elif key.startswith("operator"):
                operators.append(value)
        
        # If no rules, return False
        if not results:
            return False
            
        # Apply operators
        result = results[0]
        for j in range(1, len(results)):
            if j-1 < len(operators):
                op = operators[j-1]
                if op == "AND":
                    result = result and results[j]
                elif op == "OR":
                    result = result or results[j]
        
        return result
    
    def _evaluate_rule(self, rule: Dict[str, Any], i: int) -> bool:
        """Evaluate a single rule against the current candle."""
        try:
            indicator = rule.get("indicator", {})
            condition = rule.get("condition", {})
            
            indicator_name = indicator.get("name")
            settings = indicator.get("indicatorSettings", {})
            
            # Get the indicator value
            indicator_value = self._get_indicator_value(indicator_name, settings, i)
            
            # Evaluate the condition
            condition_value = float(condition.get("value", 0))
            condition_symbol = condition.get("symbol", "==")
            
            if condition_symbol == "<":
                return indicator_value < condition_value
            elif condition_symbol == ">":
                return indicator_value > condition_value
            elif condition_symbol == "<=":
                return indicator_value <= condition_value
            elif condition_symbol == ">=":
                return indicator_value >= condition_value
            elif condition_symbol == "==":
                return indicator_value == condition_value
            elif condition_symbol == "!=":
                return indicator_value != condition_value
            
            return False
        except Exception as e:
            logger.error(f"Error evaluating rule: {str(e)}")
            return False
    
    def _get_indicator_value(self, indicator_name: str, settings: Dict[str, Any], i: int) -> float:
        """Get the indicator value for the current candle."""
        try:
            # For most indicators, we can simply look up the calculated value in the data
            # The indicator columns are named based on indicator and parameters
            
            # Build a pattern to search for in column names
            indicator_pattern = f"{indicator_name}"
            
            # Find matching columns
            matching_columns = [col for col in self.data.df.columns if indicator_pattern in col]
            
            if matching_columns:
                # Use the first matching column (in most cases there will be only one)
                return self.data.df[matching_columns[0]].iloc[i]
            
            # Special handling for built-in backtesting.py indicators
            if indicator_name == "Price":
                return self.data.Close[i]
            
            return 0.0
        except Exception as e:
            logger.error(f"Error getting indicator value: {str(e)}")
            return 0.0
    
    def _calculate_position_size(self, price: float) -> float:
        """Calculate position size based on position sizing strategy."""
        equity = self.equity
        
        if self.position_sizing == "fixed":
            return 0.1 * equity / price  # 10% of equity
        elif self.position_sizing == "percentage":
            return (equity * self.risk_percentage / 100) / price
        elif self.position_sizing == "risk_based":
            if self.stop_loss_pct == 0:
                return 0.1 * equity / price  # Fallback to 10% of equity
            return (equity * self.risk_percentage / 100) / (price * self.stop_loss_pct / 100)
        else:
            # Default to 10% of equity
            return 0.1 * equity / price
    
    def next(self):
        """
        This method is called for each candle in the data.
        It's the main strategy logic implementation.
        """
        # Current candle index
        i = len(self.data) - 1
        
        # Check for buy signal if we're not in a position
        if not self.position:
            if self.buy_rules and self._evaluate_rules(self.buy_rules, i):
                # Calculate position size
                size = self._calculate_position_size(self.data.Close[i])
                
                # Place a buy order
                self.buy(size=size)
                
                # Set up stop loss and take profit if defined
                if self.stop_loss_pct > 0:
                    self.stop_loss = self.data.Close[i] * (1 - self.stop_loss_pct / 100)
                
                if self.take_profit_pct > 0:
                    self.take_profit = self.data.Close[i] * (1 + self.take_profit_pct / 100)
        
        # Check for sell signal if we're in a position
        elif self.position.is_long:
            # Update trailing stop if enabled
            if self.trailing_stop_pct > 0:
                # Calculate new stop level based on highest price since entry
                highest_price = max(self.data.High[self.position.entry_time:i+1])
                new_stop = highest_price * (1 - self.trailing_stop_pct / 100)
                
                # Update stop loss if the new stop is higher
                if not hasattr(self, "stop_loss") or new_stop > self.stop_loss:
                    self.stop_loss = new_stop
            
            # Check if we should sell based on the strategy rules
            if self.sell_rules and self._evaluate_rules(self.sell_rules, i):
                self.position.close()

def build_strategy(strategy_config: Dict[str, Any], params: Dict[str, Any]) -> Type[Strategy]:
    """Build a dynamic strategy from JSON configuration."""
    # Create a customized strategy class
    class CustomStrategy(DynamicStrategy):
        # Store configuration as class variables
        strategy_config = strategy_config
        params = params
    
    return CustomStrategy

def validate_strategy(strategy_config: Dict[str, Any]) -> Tuple[bool, str]:
    """Validate a strategy structure."""
    # Basic validation: check if buyRules or sellRules exist
    if not strategy_config.get("buyRules") and not strategy_config.get("sellRules"):
        return False, "Strategy must have at least buyRules or sellRules"
    
    # Validate buy rules if present
    if "buyRules" in strategy_config:
        valid, message = validate_rules(strategy_config["buyRules"])
        if not valid:
            return False, f"Invalid buy rules: {message}"
    
    # Validate sell rules if present
    if "sellRules" in strategy_config:
        valid, message = validate_rules(strategy_config["sellRules"])
        if not valid:
            return False, f"Invalid sell rules: {message}"
    
    return True, "Strategy structure is valid"

def validate_rules(rules: Dict[str, Any]) -> Tuple[bool, str]:
    """Recursively validate rules structure."""
    # Check if rules has a sequence
    if "_sequence" in rules:
        sequence = rules["_sequence"]
        
        # Validate sequence structure
        for item in sequence:
            if "type" not in item or "index" not in item:
                return False, "Sequence items must have type and index"
            
            item_type = item["type"]
            item_index = item["index"]
            
            if item_type == "rule":
                rule_key = f"rule{item_index}"
                if rule_key not in rules:
                    return False, f"Referenced rule {rule_key} not found"
                
                # Validate the individual rule
                valid, message = validate_rule(rules[rule_key])
                if not valid:
                    return False, message
            
            elif item_type == "group":
                group_key = f"group{item_index}"
                if group_key not in rules:
                    return False, f"Referenced group {group_key} not found"
                
                # Recursively validate the group
                valid, message = validate_rules(rules[group_key])
                if not valid:
                    return False, message
            
            elif item_type == "operator":
                op_key = f"operator{item_index}"
                if op_key not in rules:
                    return False, f"Referenced operator {op_key} not found"
                
                if rules[op_key] not in ["AND", "OR"]:
                    return False, f"Operator {op_key} must be AND or OR"
    else:
        # Simple rules without sequence
        # Validate each rule and operator
        for key, value in rules.items():
            if key.startswith("rule"):
                valid, message = validate_rule(value)
                if not valid:
                    return False, message
            elif key.startswith("operator"):
                if value not in ["AND", "OR"]:
                    return False, f"Operator {key} must be AND or OR"
            elif key.startswith("group"):
                valid, message = validate_rules(value)
                if not valid:
                    return False, message
    
    return True, "Rules structure is valid"

def validate_rule(rule: Dict[str, Any]) -> Tuple[bool, str]:
    """Validate a single rule."""
    # Check for required fields
    if "indicator" not in rule:
        return False, "Rule must have an indicator"
    
    if "condition" not in rule:
        return False, "Rule must have a condition"
    
    # Validate indicator
    indicator = rule["indicator"]
    if "name" not in indicator:
        return False, "Indicator must have a name"
    
    # Validate condition
    condition = rule["condition"]
    if "value" not in condition:
        return False, "Condition must have a value"
    
    if "symbol" not in condition:
        return False, "Condition must have a symbol"
    
    # Validate condition symbol
    if condition["symbol"] not in ["<", ">", "<=", ">=", "==", "!="]:
        return False, "Condition symbol must be one of: <, >, <=, >=, ==, !="
    
    return True, "Rule is valid"