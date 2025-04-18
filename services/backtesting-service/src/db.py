#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Database connection module for direct access to historical and strategy data.
"""

import os
import logging
from typing import List, Dict, Any, Optional, Tuple
import psycopg2
import psycopg2.extras
from datetime import datetime, timedelta

logger = logging.getLogger(__name__)

# Database configuration classes
class DBConfig:
    """Base class for database configurations"""
    def __init__(self, prefix):
        self.host = os.environ.get(f"{prefix}_DB_HOST", f"{prefix.lower()}-db")
        self.port = os.environ.get(f"{prefix}_DB_PORT", "5432")
        self.user = os.environ.get(f"{prefix}_DB_USER", f"{prefix.lower()}_service_user")
        self.password = os.environ.get(f"{prefix}_DB_PASSWORD", f"{prefix.lower()}_service_password")
        self.name = os.environ.get(f"{prefix}_DB_NAME", f"{prefix.lower()}_service")
    
    def get_connection_string(self) -> str:
        """Get connection string for this database"""
        return f"host={self.host} port={self.port} dbname={self.name} user={self.user} password={self.password}"

# Global database configurations
HISTORICAL_DB = DBConfig("HISTORICAL")
STRATEGY_DB = DBConfig("STRATEGY")

def get_db_connection(db_config: DBConfig, connect_timeout: int = 10):
    """
    Create a connection to a database.
    
    Args:
        db_config: Database configuration object
        connect_timeout: Connection timeout in seconds
        
    Returns:
        connection: PostgreSQL connection object
    """
    try:
        conn = psycopg2.connect(
            db_config.get_connection_string(),
            connect_timeout=connect_timeout
        )
        return conn
    except Exception as e:
        logger.error(f"Failed to connect to database {db_config.name}: {str(e)}")
        raise RuntimeError(f"Database connection failed: {str(e)}")

def check_db_connection(db_config: DBConfig, timeout: int = 3) -> Dict[str, Any]:
    """
    Check database connection health.
    
    Args:
        db_config: Database configuration object
        timeout: Connection timeout in seconds
        
    Returns:
        Dict with connection status information
    """
    status = {
        "status": "unknown",
        "host": db_config.host,
        "port": db_config.port
    }
    
    try:
        # Quick connection test with short timeout
        conn = psycopg2.connect(
            db_config.get_connection_string(),
            connect_timeout=timeout
        )
        cursor = conn.cursor()
        cursor.execute("SELECT 1")
        cursor.close()
        conn.close()
        status["status"] = "connected"
    except Exception as e:
        logger.warning(f"DB connection failed: {str(e)}")
        status["status"] = "error"
        status["message"] = str(e)
    
    return status

# Historical database functions

def get_historical_db_connection(connect_timeout: int = 10):
    """Get a connection to the historical database"""
    return get_db_connection(HISTORICAL_DB, connect_timeout)

def get_candles(
    symbol_id: int,
    timeframe: str,
    start_time: datetime,
    end_time: datetime,
    limit: Optional[int] = None
) -> List[Dict[str, Any]]:
    """
    Get candle data directly from the database.
    
    Args:
        symbol_id: Symbol ID
        timeframe: Timeframe (e.g., '1m', '5m', '1h')
        start_time: Start time
        end_time: End time
        limit: Optional limit on number of candles
        
    Returns:
        List of candle dictionaries
    """
    conn = None
    try:
        conn = get_historical_db_connection()
        with conn.cursor(cursor_factory=psycopg2.extras.DictCursor) as cursor:
            # Use the database function to get candles
            query = """
                SELECT * FROM get_candles(%s, %s, %s, %s, %s)
                ORDER BY candle_time ASC
            """
            
            cursor.execute(
                query,
                (symbol_id, timeframe, start_time, end_time, limit)
            )
            
            # Convert to list of dictionaries
            candles = []
            for row in cursor.fetchall():
                candle = {
                    'symbol_id': row['symbol_id'],
                    'time': row['candle_time'].isoformat(),
                    'open': float(row['open']),
                    'high': float(row['high']),
                    'low': float(row['low']),
                    'close': float(row['close']),
                    'volume': float(row['volume'])
                }
                candles.append(candle)
                
            return candles
    except Exception as e:
        logger.error(f"Failed to get candles: {str(e)}")
        raise RuntimeError(f"Failed to get candles: {str(e)}")
    finally:
        if conn:
            conn.close()
            
def get_symbol_by_id(symbol_id: int) -> Dict[str, Any]:
    """
    Get symbol information by ID.
    
    Args:
        symbol_id: Symbol ID
        
    Returns:
        Symbol information
    """
    conn = None
    try:
        conn = get_historical_db_connection()
        with conn.cursor(cursor_factory=psycopg2.extras.DictCursor) as cursor:
            query = """
                SELECT id, symbol, name, asset_type, exchange, is_active, data_available
                FROM symbols
                WHERE id = %s
            """
            
            cursor.execute(query, (symbol_id,))
            row = cursor.fetchone()
            
            if not row:
                return None
                
            symbol = {
                'id': row['id'],
                'symbol': row['symbol'],
                'name': row['name'],
                'asset_type': row['asset_type'],
                'exchange': row['exchange'],
                'is_active': row['is_active'],
                'data_available': row['data_available']
            }
            
            return symbol
    except Exception as e:
        logger.error(f"Failed to get symbol: {str(e)}")
        raise RuntimeError(f"Failed to get symbol: {str(e)}")
    finally:
        if conn:
            conn.close()
            
def save_backtest_result(
    backtest_run_id: int,
    total_trades: int,
    winning_trades: int,
    losing_trades: int,
    profit_factor: float,
    sharpe_ratio: float,
    max_drawdown: float,
    final_capital: float,
    total_return: float,
    annualized_return: float,
    results_json: Dict[str, Any]
) -> int:
    """
    Save backtest results directly to the database.
    
    Args:
        backtest_run_id: Backtest run ID
        total_trades: Total number of trades
        winning_trades: Number of winning trades
        losing_trades: Number of losing trades
        profit_factor: Profit factor
        sharpe_ratio: Sharpe ratio
        max_drawdown: Maximum drawdown
        final_capital: Final capital
        total_return: Total return
        annualized_return: Annualized return
        results_json: Detailed results as JSON
        
    Returns:
        Result ID
    """
    conn = None
    try:
        conn = get_historical_db_connection()
        with conn.cursor() as cursor:
            query = """
                SELECT save_backtest_result(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            """
            
            cursor.execute(
                query,
                (
                    backtest_run_id,
                    total_trades,
                    winning_trades,
                    losing_trades,
                    profit_factor,
                    sharpe_ratio,
                    max_drawdown,
                    final_capital,
                    total_return,
                    annualized_return,
                    psycopg2.extras.Json(results_json)
                )
            )
            
            result_id = cursor.fetchone()[0]
            conn.commit()
            
            return result_id
    except Exception as e:
        if conn:
            conn.rollback()
        logger.error(f"Failed to save backtest result: {str(e)}")
        raise RuntimeError(f"Failed to save backtest result: {str(e)}")
    finally:
        if conn:
            conn.close()
            
def add_backtest_trade(
    backtest_run_id: int,
    symbol_id: int,
    entry_time: datetime,
    exit_time: Optional[datetime],
    position_type: str,
    entry_price: float,
    exit_price: Optional[float],
    quantity: float,
    profit_loss: Optional[float],
    profit_loss_percent: Optional[float],
    exit_reason: Optional[str]
) -> int:
    """
    Add a backtest trade directly to the database.
    
    Args:
        backtest_run_id: Backtest run ID
        symbol_id: Symbol ID
        entry_time: Entry time
        exit_time: Exit time
        position_type: Position type (long/short)
        entry_price: Entry price
        exit_price: Exit price
        quantity: Quantity
        profit_loss: Profit/loss
        profit_loss_percent: Profit/loss percentage
        exit_reason: Exit reason
        
    Returns:
        Trade ID
    """
    conn = None
    try:
        conn = get_historical_db_connection()
        with conn.cursor() as cursor:
            query = """
                SELECT add_backtest_trade(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            """
            
            cursor.execute(
                query,
                (
                    backtest_run_id,
                    symbol_id,
                    entry_time,
                    exit_time,
                    position_type,
                    entry_price,
                    exit_price,
                    quantity,
                    profit_loss,
                    profit_loss_percent,
                    exit_reason
                )
            )
            
            trade_id = cursor.fetchone()[0]
            conn.commit()
            
            return trade_id
    except Exception as e:
        if conn:
            conn.rollback()
        logger.error(f"Failed to add backtest trade: {str(e)}")
        raise RuntimeError(f"Failed to add backtest trade: {str(e)}")
    finally:
        if conn:
            conn.close()

# Strategy database functions

def get_strategy_db_connection(connect_timeout: int = 10):
    """Get a connection to the strategy database"""
    return get_db_connection(STRATEGY_DB, connect_timeout)

def sync_indicators_to_strategy_db(indicators_data: List[Dict[str, Any]]) -> Dict[str, Any]:
    """
    Sync indicators to the strategy service database.
    
    Args:
        indicators_data: List of indicator data
        
    Returns:
        Dict with sync status and count
    """
    conn = None
    cursor = None
    try:
        # Connect to strategy service database
        conn = get_strategy_db_connection()
        cursor = conn.cursor()
        
        # Get existing indicators for comparison
        cursor.execute("SELECT id, name FROM indicators")
        existing_indicators = {name: id for id, name in cursor.fetchall()}
        
        # Start transaction
        conn.autocommit = False
        
        sync_count = 0
        for indicator in indicators_data:
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