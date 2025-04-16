#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Dynamic indicator discovery and management.
Automatically discovers indicators from TA-Lib and pandas-ta.
"""

import inspect
import logging
import pandas as pd
import pandas_ta as ta
from typing import Dict, List, Any, Callable, Optional
import talib
from talib import abstract

logger = logging.getLogger(__name__)

# Dictionary to store discovered indicators
INDICATORS: Dict[str, Dict[str, Any]] = {}

# Mapping of frontend indicator names to backend implementations
INDICATOR_MAPPING: Dict[str, str] = {
    "RSI": "RSI",
    "Moving Average": "SMA",
    "SMA": "SMA",
    "EMA": "EMA",
    "MACD": "MACD",
    "Bollinger Bands": "BBANDS",
    "Stochastic": "STOCH",
    "Average True Range": "ATR",
    "Commodity Channel Index": "CCI",
    "Relative Strength Index": "RSI"
}

def discover_talib_indicators() -> None:
    """Dynamically discover all TA-Lib indicators and their parameters."""
    # Get all available functions
    functions = talib.get_functions()
    
    for func_name in functions:
        try:
            # Get the function
            func = getattr(abstract, func_name)
            
            # Get function info
            info = abstract.Function(func_name).info
            
            # Extract parameter information
            parameters = []
            for param_name, param_default in info.parameters.items():
                param_type = "int" if isinstance(param_default, int) else "float"
                parameters.append({
                    "name": param_name,
                    "default": str(param_default),
                    "type": param_type
                })
            
            # Store indicator info
            INDICATORS[func_name] = {
                "name": func_name,
                "description": info.display_name if hasattr(info, 'display_name') else func_name,
                "parameters": parameters,
                "function": func,
                "source": "talib"
            }
        except Exception as e:
            logger.warning(f"Failed to process TA-Lib indicator {func_name}: {str(e)}")

def discover_pandas_ta_indicators() -> None:
    """Dynamically discover pandas-ta indicators and their parameters."""
    # Get all available functions from pandas_ta
    for name, func in inspect.getmembers(ta, inspect.isfunction):
        try:
            # Skip helper functions
            if name.startswith('_') or name in ['plot', 'version']:
                continue
                
            # Get function signature
            sig = inspect.signature(func)
            
            # Extract parameter information
            parameters = []
            for param_name, param in sig.parameters.items():
                # Skip self, df, and other special parameters
                if param_name in ['self', 'df', 'open', 'high', 'low', 'close', 'volume']:
                    continue
                    
                # Get default value and type
                if param.default is not param.empty:
                    param_type = "int" if isinstance(param.default, int) else \
                                 "float" if isinstance(param.default, float) else \
                                 "bool" if isinstance(param.default, bool) else "str"
                    
                    default_val = str(param.default) if param.default is not None else ""
                    
                    parameters.append({
                        "name": param_name,
                        "default": default_val,
                        "type": param_type
                    })
            
            # Store indicator info if it has parameters
            if parameters:
                INDICATORS[name.upper()] = {
                    "name": name.upper(),
                    "description": name.replace('_', ' ').title(),
                    "parameters": parameters,
                    "function": func,
                    "source": "pandas_ta"
                }
        except Exception as e:
            logger.warning(f"Failed to process pandas-ta indicator {name}: {str(e)}")

def initialize_indicators() -> None:
    """Initialize and discover all available indicators."""
    if not INDICATORS:
        # Discover indicators from libraries
        discover_talib_indicators()
        discover_pandas_ta_indicators()
        
        logger.info(f"Discovered {len(INDICATORS)} indicators")

def get_indicator(name: str) -> Optional[Dict[str, Any]]:
    """Get indicator details by name."""
    initialize_indicators()
    
    # First, try direct match
    if name in INDICATORS:
        return INDICATORS[name]
    
    # Then try via mapping
    if name in INDICATOR_MAPPING and INDICATOR_MAPPING[name] in INDICATORS:
        return INDICATORS[INDICATOR_MAPPING[name]]
    
    return None

def calculate_indicator(df: pd.DataFrame, indicator_name: str, params: Dict[str, Any]) -> pd.DataFrame:
    """Calculate a specific indicator and add it to the dataframe."""
    indicator = get_indicator(indicator_name)
    if not indicator:
        logger.warning(f"Indicator {indicator_name} not found")
        return df
    
    # Prepare parameters
    func_params = {}
    for param in indicator['parameters']:
        if param['name'] in params:
            # Convert parameter to the appropriate type
            if param['type'] == 'int':
                func_params[param['name']] = int(params[param['name']])
            elif param['type'] == 'float':
                func_params[param['name']] = float(params[param['name']])
            elif param['type'] == 'bool':
                func_params[param['name']] = bool(params[param['name']])
            else:
                func_params[param['name']] = params[param['name']]
        elif param['default']:
            # Use default value if available
            if param['type'] == 'int':
                func_params[param['name']] = int(param['default'])
            elif param['type'] == 'float':
                func_params[param['name']] = float(param['default'])
            elif param['type'] == 'bool':
                func_params[param['name']] = param['default'].lower() == 'true'
            else:
                func_params[param['name']] = param['default']
    
    # Calculate the indicator
    source = indicator['source']
    name = indicator['name']
    
    try:
        # Calculate differently based on source
        if source == 'talib':
            func = indicator['function']
            # For TA-Lib, we need to specify the inputs
            result = func(
                df['open'], df['high'], df['low'], df['close'], df['volume'], **func_params
            )
            
            # Handle different return types
            if isinstance(result, pd.Series):
                df[f"{name}_{params_to_string(func_params)}"] = result
            elif isinstance(result, tuple):
                # For indicators that return multiple outputs
                for i, res in enumerate(result):
                    df[f"{name}_{i}_{params_to_string(func_params)}"] = res
        elif source == 'pandas_ta':
            # For pandas-ta, we use the extension method
            result = df.ta.__getattr__(name.lower())(**func_params)
            
            # If result is a DataFrame, merge it in
            if isinstance(result, pd.DataFrame):
                for column in result.columns:
                    df[column] = result[column]
            elif isinstance(result, pd.Series):
                df[result.name] = result
    except Exception as e:
        logger.error(f"Error calculating indicator {name}: {str(e)}")
    
    return df

def params_to_string(params: Dict[str, Any]) -> str:
    """Convert parameters to a string for column naming."""
    return '_'.join(f"{k}_{v}" for k, v in params.items())

def get_available_indicators() -> List[Dict[str, Any]]:
    """Return a list of all available indicators with their metadata."""
    initialize_indicators()
    
    # Format indicators for API response
    formatted_indicators = []
    
    id_counter = 1
    for name, indicator in INDICATORS.items():
        formatted_indicators.append({
            "id": str(id_counter),
            "name": indicator["name"],
            "description": indicator["description"],
            "parameters": [
                {
                    "name": param["name"],
                    "default": param["default"],
                    "type": param["type"],
                    "options": param.get("options", [])
                }
                for param in indicator["parameters"]
            ]
        })
        id_counter += 1
    
    return formatted_indicators