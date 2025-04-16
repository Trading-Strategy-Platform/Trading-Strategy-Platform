#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Backtesting engine using backtesting.py.
Executes backtest runs based on provided data and strategy configuration.
"""

import logging
import pandas as pd
import numpy as np
from typing import Dict, List, Any, Tuple
from datetime import datetime
from backtesting import Backtest

from src.models import (
    candles_to_dataframe, BacktestParameters, BacktestMetrics,
    TradeResult, BacktestResult
)
from src.strategies import build_strategy

logger = logging.getLogger(__name__)

def run_backtest(
    candles: List[Dict[str, Any]],
    strategy: Dict[str, Any],
    params: Dict[str, Any]
) -> Dict[str, Any]:
    """
    Run a backtest using the provided candles and strategy configuration.
    
    Args:
        candles: List of candle data (OHLCV)
        strategy: Strategy configuration from the frontend
        params: Backtest parameters
    
    Returns:
        Dict containing backtest results
    """
    try:
        # Convert parameters
        backtest_params = BacktestParameters.from_dict(params)
        
        # Convert candles to DataFrame
        df = candles_to_dataframe(candles)
        
        # Ensure required columns are present
        required_columns = ['open', 'high', 'low', 'close', 'volume']
        for col in required_columns:
            if col not in df.columns:
                logger.error(f"Required column {col} not found in data")
                raise ValueError(f"Required column {col} not found in data")
        
        # Rename columns to match backtesting.py expected format
        df = df.rename(columns={
            'open': 'Open',
            'high': 'High',
            'low': 'Low',
            'close': 'Close',
            'volume': 'Volume'
        })
        
        # Remove any NaN values
        df = df.dropna()
        
        # Build the strategy class
        strategy_class = build_strategy(strategy, params)
        
        # Run the backtest
        bt = Backtest(
            df,
            strategy_class,
            cash=backtest_params.initial_capital,
            commission=backtest_params.commission_rate/100,
            exclusive_orders=True
        )
        
        # Run the backtest
        result = bt.run()
        
        # Process results
        metrics = process_backtest_metrics(result, backtest_params.initial_capital)
        trades = process_backtest_trades(result, backtest_params.symbol_id)
        equity_curve, equity_times = extract_equity_curve(result)
        
        # Create result object
        backtest_result = BacktestResult(
            trades=trades,
            equity_curve=equity_curve,
            equity_times=[t.isoformat() for t in equity_times],
            metrics=metrics
        )
        
        # Convert to dict for JSON serialization
        return backtest_result.to_dict()
    except Exception as e:
        logger.exception(f"Error running backtest: {str(e)}")
        raise RuntimeError(f"Failed to run backtest: {str(e)}")

def process_backtest_metrics(result: pd.Series, initial_capital: float) -> BacktestMetrics:
    """
    Process backtest results to generate performance metrics.
    
    Args:
        result: Results from backtesting.py
        initial_capital: Initial capital amount
    
    Returns:
        BacktestMetrics object with calculated performance metrics
    """
    # Extract metrics from backtesting.py result
    total_trades = len(result['_trades'])
    winning_trades = sum(1 for t in result['_trades'] if t.PnL > 0)
    losing_trades = total_trades - winning_trades
    
    # Calculate win rate
    win_rate = (winning_trades / total_trades * 100) if total_trades > 0 else 0
    
    # Calculate profit factor
    total_profit = sum(t.PnL for t in result['_trades'] if t.PnL > 0)
    total_loss = sum(-t.PnL for t in result['_trades'] if t.PnL <= 0)
    profit_factor = total_profit / total_loss if total_loss > 0 else float('inf')
    
    # Get final equity
    final_capital = result['Equity'][-1]
    
    # Calculate total return
    total_return = (final_capital / initial_capital - 1) * 100
    
    # Calculate annualized return (assuming 252 trading days per year)
    days = len(result['Equity'])
    annualized_return = ((1 + total_return / 100) ** (252 / max(days, 1)) - 1) * 100
    
    # Calculate Sharpe ratio
    returns = np.diff(result['Equity']) / result['Equity'][:-1]
    sharpe_ratio = (np.mean(returns) / np.std(returns)) * np.sqrt(252) if np.std(returns) > 0 else 0
    
    # Calculate max drawdown
    if 'Max. Drawdown [%]' in result:
        max_drawdown = result['Max. Drawdown [%]']
    else:
        # Calculate manually if not provided
        peak = result['Equity'][0]
        max_drawdown = 0
        for equity in result['Equity']:
            if equity > peak:
                peak = equity
            drawdown = (peak - equity) / peak * 100
            max_drawdown = max(max_drawdown, drawdown)
    
    # Calculate average trade values
    average_trade = (final_capital - initial_capital) / total_trades if total_trades > 0 else 0
    average_win = total_profit / winning_trades if winning_trades > 0 else 0
    average_loss = total_loss / losing_trades if losing_trades > 0 else 0
    
    # Get largest win/loss
    largest_win = max([t.PnL for t in result['_trades']], default=0)
    largest_loss = min([t.PnL for t in result['_trades']], default=0)
    
    # Create metrics object
    return BacktestMetrics(
        total_trades=total_trades,
        winning_trades=winning_trades,
        losing_trades=losing_trades,
        win_rate=win_rate,
        profit_factor=profit_factor,
        sharpe_ratio=sharpe_ratio,
        max_drawdown=max_drawdown,
        final_capital=final_capital,
        total_return=total_return,
        annualized_return=annualized_return,
        average_trade=average_trade,
        average_win=average_win,
        average_loss=average_loss,
        largest_win=largest_win,
        largest_loss=largest_loss
    )

def process_backtest_trades(result: pd.Series, symbol_id: int) -> List[TradeResult]:
    """
    Process trades from backtesting.py results.
    
    Args:
        result: Results from backtesting.py
        symbol_id: Symbol ID for the backtest
    
    Returns:
        List of TradeResult objects
    """
    trades = []
    
    # Process each trade
    for bt_trade in result['_trades']:
        # Convert to our trade model
        trade = TradeResult(
            symbol_id=symbol_id,
            entry_time=bt_trade.EntryTime,
            exit_time=bt_trade.ExitTime if hasattr(bt_trade, 'ExitTime') else None,
            position_type='long' if bt_trade.Size > 0 else 'short',
            entry_price=bt_trade.EntryPrice,
            exit_price=bt_trade.ExitPrice if hasattr(bt_trade, 'ExitPrice') else None,
            quantity=abs(bt_trade.Size),
            profit_loss=bt_trade.PnL,
            profit_loss_percent=bt_trade.PnL / (bt_trade.EntryPrice * abs(bt_trade.Size)) * 100,
            exit_reason=bt_trade.ExitReason if hasattr(bt_trade, 'ExitReason') else None
        )
        
        trades.append(trade)
    
    return trades

def extract_equity_curve(result: pd.Series) -> Tuple[List[float], List[datetime]]:
    """
    Extract equity curve and times from backtest results.
    
    Args:
        result: Results from backtesting.py
    
    Returns:
        Tuple of (equity values, equity times)
    """
    # Get equity curve from results
    equity_curve = result['Equity'].tolist()
    
    # Get times for the equity curve
    if hasattr(result, 'index'):
        equity_times = result.index.tolist()
    else:
        # If times are not available, generate sequential times
        equity_times = [datetime.now()] * len(equity_curve)
    
    return equity_curve, equity_times