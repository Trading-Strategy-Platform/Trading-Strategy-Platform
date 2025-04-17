#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Database connection module for direct access to historical data.
"""

import os
import logging
from typing import List, Dict, Any, Optional, Tuple
import psycopg2
import psycopg2.extras
from datetime import datetime, timedelta

logger = logging.getLogger(__name__)

# Get database configuration from environment variables
DB_HOST = os.environ.get("HISTORICAL_DB_HOST", "historical-db")
DB_PORT = os.environ.get("HISTORICAL_DB_PORT", "5432")
DB_USER = os.environ.get("HISTORICAL_DB_USER", "historical_service_user")
DB_PASSWORD = os.environ.get("HISTORICAL_DB_PASSWORD", "historical_service_password")
DB_NAME = os.environ.get("HISTORICAL_DB_NAME", "historical_service")

def get_db_connection():
    """
    Create a connection to the historical database.
    
    Returns:
        connection: PostgreSQL connection object
    """
    try:
        conn = psycopg2.connect(
            host=DB_HOST,
            port=DB_PORT,
            user=DB_USER,
            password=DB_PASSWORD,
            dbname=DB_NAME
        )
        return conn
    except Exception as e:
        logger.error(f"Failed to connect to database: {str(e)}")
        raise RuntimeError(f"Database connection failed: {str(e)}")

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
    try:
        conn = get_db_connection()
        with conn.cursor(cursor_factory=psycopg2.extras.DictCursor) as cursor:
            # Use the database function to get candles
            limit_clause = f"LIMIT {limit}" if limit else ""
            
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
    try:
        conn = get_db_connection()
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
    try:
        conn = get_db_connection()
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
    try:
        conn = get_db_connection()
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