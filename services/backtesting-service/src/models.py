#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Data models for the backtesting service.
"""

import pandas as pd
from typing import Dict, List, Any, Optional, Union
from dataclasses import dataclass
from datetime import datetime

@dataclass
class CandleData:
    """Represents OHLCV candle data."""
    time: datetime
    open: float
    high: float
    low: float
    close: float
    volume: float
    symbol_id: int

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'CandleData':
        """Create a CandleData instance from a dictionary."""
        # Handle various time formats
        if isinstance(data['time'], str):
            time = pd.to_datetime(data['time'])
        elif isinstance(data['time'], (int, float)):
            time = pd.to_datetime(data['time'], unit='ms')
        else:
            time = data['time']
            
        return cls(
            time=time,
            open=float(data['open']),
            high=float(data['high']),
            low=float(data['low']),
            close=float(data['close']),
            volume=float(data['volume']),
            symbol_id=int(data.get('symbol_id', 0))
        )

@dataclass
class BacktestParameters:
    """Parameters for backtest configuration."""
    symbol_id: int
    initial_capital: float
    market_type: str = 'spot'  # 'spot' or 'futures'
    leverage: float = 1.0
    commission_rate: float = 0.1  # In percentage
    slippage_rate: float = 0.05  # In percentage
    position_sizing: str = 'fixed'  # 'fixed', 'percentage', 'risk_based'
    risk_percentage: float = 2.0  # In percentage
    stop_loss: float = 0.0  # In percentage
    take_profit: float = 0.0  # In percentage
    trailing_stop: float = 0.0  # In percentage
    allow_short: bool = False

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'BacktestParameters':
        """Create a BacktestParameters instance from a dictionary."""
        return cls(
            symbol_id=int(data.get('symbol_id', 0)),
            initial_capital=float(data.get('initial_capital', 10000.0)),
            market_type=data.get('market_type', 'spot'),
            leverage=float(data.get('leverage', 1.0)),
            commission_rate=float(data.get('commission_rate', 0.1)),
            slippage_rate=float(data.get('slippage_rate', 0.05)),
            position_sizing=data.get('position_sizing', 'fixed'),
            risk_percentage=float(data.get('risk_percentage', 2.0)),
            stop_loss=float(data.get('stop_loss', 0.0)),
            take_profit=float(data.get('take_profit', 0.0)),
            trailing_stop=float(data.get('trailing_stop', 0.0)),
            allow_short=bool(data.get('allow_short', False))
        )

@dataclass
class TradeResult:
    """Represents a single trade result."""
    symbol_id: int
    entry_time: datetime
    exit_time: Optional[datetime]
    position_type: str  # 'long' or 'short'
    entry_price: float
    exit_price: Optional[float]
    quantity: float
    profit_loss: Optional[float]
    profit_loss_percent: Optional[float]
    exit_reason: Optional[str]

@dataclass
class BacktestMetrics:
    """Performance metrics from a backtest."""
    total_trades: int
    winning_trades: int
    losing_trades: int
    win_rate: float
    profit_factor: float
    sharpe_ratio: float
    max_drawdown: float
    final_capital: float
    total_return: float
    annualized_return: float
    average_trade: float
    average_win: float
    average_loss: float
    largest_win: float
    largest_loss: float

@dataclass
class BacktestResult:
    """Complete result of a backtest run."""
    trades: List[TradeResult]
    equity_curve: List[float]
    equity_times: List[str]
    metrics: BacktestMetrics
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert the result to a dictionary for JSON serialization."""
        return {
            'trades': [vars(trade) for trade in self.trades],
            'equity_curve': self.equity_curve,
            'equity_times': self.equity_times,
            'metrics': vars(self.metrics)
        }

def candles_to_dataframe(candles: List[Dict[str, Any]]) -> pd.DataFrame:
    """Convert a list of candle dictionaries to a pandas DataFrame."""
    # Convert to CandleData objects for consistent processing
    candle_objs = [CandleData.from_dict(candle) for candle in candles]
    
    # Convert to dataframe
    df = pd.DataFrame([vars(c) for c in candle_objs])
    
    # Set time as index and sort
    if 'time' in df.columns:
        df.set_index('time', inplace=True)
        df.sort_index(inplace=True)
    
    return df