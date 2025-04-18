#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Dynamic indicator discovery and management.
Automatically discovers indicators from TA-Lib and pandas-ta.
"""

import os
import inspect
import logging
import pandas as pd
import pandas_ta as ta
import psycopg2
from typing import Dict, List, Any, Callable, Optional
import talib
from talib import abstract
from src import db

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

def sync_indicators():
    """
    Sync indicators from backtesting service to strategy service database.
    Uses environment variables for database connection.
    
    Returns:
        dict: Result of the sync operation
    """
    # Get indicators from backtesting service
    indicators = get_available_indicators()
    
    # Get database connection parameters from environment variables
    db_host = os.environ.get("STRATEGY_DB_HOST", "strategy-db")
    db_port = os.environ.get("STRATEGY_DB_PORT", "5432")
    db_name = os.environ.get("STRATEGY_DB_NAME", "strategy_service")
    db_user = os.environ.get("STRATEGY_DB_USER", "strategy_service_user")
    db_pass = os.environ.get("STRATEGY_DB_PASSWORD", "strategy_service_password")
    
    # Connect to strategy service database using environment variables
    db_conn_string = f"host={db_host} port={db_port} dbname={db_name} user={db_user} password={db_pass}"
    logger.info(f"Connecting to database: host={db_host} port={db_port} dbname={db_name}")
    
    conn = None
    cursor = None
    try:
        # Connect to database
        conn = psycopg2.connect(db_conn_string)
        cursor = conn.cursor()
        
        # Get existing indicators for comparison
        cursor.execute("SELECT id, name FROM indicators")
        existing_indicators = {name: id for id, name in cursor.fetchall()}
        
        # Start transaction
        conn.autocommit = False
        
        sync_count = 0
        for indicator in indicators:
            indicator_name = indicator['name']
            
            if indicator_name in existing_indicators:
                # Update existing indicator
                indicator_id = existing_indicators[indicator_name]
                cursor.execute(
                    "UPDATE indicators SET description = %s, updated_at = NOW() WHERE id = %s",
                    (indicator['description'], indicator_id)
                )
            else:
                # Insert new indicator
                cursor.execute(
                    """INSERT INTO indicators 
                       (name, description, category, created_at, updated_at) 
                       VALUES (%s, %s, %s, NOW(), NOW()) RETURNING id""",
                    (indicator_name, indicator['description'], categorize_indicator(indicator_name))
                )
                indicator_id = cursor.fetchone()[0]
            
            # Get existing parameters for this indicator
            cursor.execute("SELECT id, parameter_name FROM indicator_parameters WHERE indicator_id = %s", 
                          (indicator_id,))
            existing_params = {param_name: param_id for param_id, param_name in cursor.fetchall()}
            
            # Process parameters
            for param in indicator.get('parameters', []):
                param_name = param['name']
                param_type = param.get('type', 'string')
                default_value = param.get('default', '')
                
                # Determine if this is an enum parameter
                has_options = 'options' in param and len(param['options']) > 0
                if has_options:
                    param_type = 'enum'
                
                if param_name in existing_params:
                    # Update existing parameter
                    param_id = existing_params[param_name]
                    cursor.execute(
                        """UPDATE indicator_parameters 
                           SET parameter_type = %s, default_value = %s 
                           WHERE id = %s""",
                        (param_type, default_value, param_id)
                    )
                else:
                    # Insert new parameter
                    cursor.execute(
                        """INSERT INTO indicator_parameters 
                           (indicator_id, parameter_name, parameter_type, default_value, is_required) 
                           VALUES (%s, %s, %s, %s, %s) RETURNING id""",
                        (indicator_id, param_name, param_type, default_value, True)
                    )
                    param_id = cursor.fetchone()[0]
                
                # Handle enum values if present
                if has_options:
                    # Get existing enum values
                    cursor.execute("SELECT id, enum_value FROM parameter_enum_values WHERE parameter_id = %s",
                                  (param_id,))
                    existing_enums = {enum_val: enum_id for enum_id, enum_val in cursor.fetchall()}
                    
                    # Process each option
                    for option in param['options']:
                        option_str = str(option)
                        if option_str not in existing_enums:
                            cursor.execute(
                                """INSERT INTO parameter_enum_values 
                                   (parameter_id, enum_value, display_name) 
                                   VALUES (%s, %s, %s)""",
                                (param_id, option_str, option_str)
                            )
            
            sync_count += 1
        
        # Commit all changes if successful
        conn.commit()
        logger.info(f"Successfully synced {sync_count} indicators")
        return {"status": "success", "indicators_synced": sync_count}
        
    except Exception as e:
        # Rollback on error
        if conn:
            conn.rollback()
        logger.error(f"Error syncing indicators: {str(e)}")
        return {"status": "error", "message": str(e)}
    
    finally:
        if cursor:
            cursor.close()
        if conn:
            conn.close()

def categorize_indicator(name):
    """
    Categorize an indicator based on its name.
    
    Args:
        name (str): The indicator name
        
    Returns:
        str: The category of the indicator
    """
    # Default category
    category = "Other"
    
    # Check for trend indicators
    trend_indicators = ["MA", "EMA", "SMA", "MACD", "ADX", "DEMA", "TEMA", "TRIMA", "Ichimoku"]
    for indicator in trend_indicators:
        if indicator.upper() in name.upper():
            return "Trend"
    
    # Check for momentum indicators
    momentum_indicators = ["RSI", "CCI", "Stochastic", "TRIX", "ROC", "MOM", "Momentum", "Williams", "TSI"]
    for indicator in momentum_indicators:
        if indicator.upper() in name.upper():
            return "Momentum"
    
    # Check for volatility indicators
    volatility_indicators = ["Bollinger", "ATR", "Keltner", "Standard Deviation", "Donchian"]
    for indicator in volatility_indicators:
        if indicator.upper() in name.upper():
            return "Volatility"
    
    # Check for volume indicators
    volume_indicators = ["Volume", "OBV", "Money Flow", "Accumulation", "CMF", "Volume Profile"]
    for indicator in volume_indicators:
        if indicator.upper() in name.upper():
            return "Volume"
    
    return category