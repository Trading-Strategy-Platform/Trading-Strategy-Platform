#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Main Flask application for the backtesting service.
"""

import logging
from datetime import datetime
from flask import Flask, request, jsonify

from src.backtest import run_backtest
from src.indicators import get_available_indicators, sync_indicators
from src.strategies import validate_strategy
import src.db as db

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Create Flask app
app = Flask(__name__)

@app.route('/health', methods=['GET'])
def health_check():
    """Enhanced health check endpoint that verifies database connections."""
    health_status = {
        "status": "healthy",
        "service": "backtesting-service",
        "timestamp": datetime.now().isoformat(),
        "connections": {}
    }
    
    # Check historical database connection using the db module
    historical_db_status = db.check_db_connection(db.HISTORICAL_DB)
    health_status["connections"]["historical_db"] = historical_db_status
    
    # Check strategy database connection using the db module
    strategy_db_status = db.check_db_connection(db.STRATEGY_DB)
    health_status["connections"]["strategy_db"] = strategy_db_status
    
    # Set overall health status based on connection statuses
    if historical_db_status["status"] != "connected" or strategy_db_status["status"] != "connected":
        health_status["status"] = "degraded"
    
    # Add indicators count
    try:
        indicators = get_available_indicators()
        health_status["indicators_count"] = len(indicators)
    except Exception as e:
        health_status["indicators_error"] = str(e)
        health_status["status"] = "degraded"
    
    # Return 200 even if degraded, to not trigger container restarts
    # The status field in the response body will indicate the true health
    return jsonify(health_status)

@app.route('/backtest', methods=['POST'])
def backtest():
    """Run a backtest with provided strategy and data."""
    try:
        data = request.json
        if not data:
            return jsonify({"error": "No data provided"}), 400
            
        # Extract data, strategy, and parameters
        candles = data.get('candles', [])
        strategy = data.get('strategy', {})
        params = data.get('params', {})
        
        # Validate inputs
        if not candles:
            return jsonify({"error": "No candle data provided"}), 400
        if not strategy:
            return jsonify({"error": "No strategy provided"}), 400
            
        # Run the backtest
        result = run_backtest(candles, strategy, params)
        
        return jsonify(result)
    except Exception as e:
        logger.exception(f"Error running backtest: {str(e)}")
        return jsonify({"error": f"Failed to run backtest: {str(e)}"}), 500

@app.route('/backtest/db', methods=['POST'])
def backtest_from_db():
    """Run a backtest with data fetched directly from the database."""
    try:
        data = request.json
        if not data:
            return jsonify({"error": "No data provided"}), 400
            
        # Extract parameters
        symbol_id = data.get('symbol_id')
        timeframe = data.get('timeframe')
        start_date_str = data.get('start_date')
        end_date_str = data.get('end_date')
        strategy = data.get('strategy', {})
        params = data.get('params', {})
        backtest_run_id = data.get('backtest_run_id')
        
        # Validate inputs
        if not symbol_id:
            return jsonify({"error": "Symbol ID is required"}), 400
        if not timeframe:
            return jsonify({"error": "Timeframe is required"}), 400
        if not start_date_str:
            return jsonify({"error": "Start date is required"}), 400
        if not end_date_str:
            return jsonify({"error": "End date is required"}), 400
        if not strategy:
            return jsonify({"error": "No strategy provided"}), 400
            
        # Parse dates
        try:
            start_date = datetime.fromisoformat(start_date_str.replace('Z', '+00:00'))
            end_date = datetime.fromisoformat(end_date_str.replace('Z', '+00:00'))
        except ValueError:
            return jsonify({"error": "Invalid date format. Use ISO 8601 format (YYYY-MM-DDTHH:MM:SSZ)"}), 400
            
        # Verify symbol exists
        symbol = db.get_symbol_by_id(symbol_id)
        if not symbol:
            return jsonify({"error": f"Symbol with ID {symbol_id} not found"}), 404
            
        logger.info(f"Fetching candles for symbol {symbol_id} from {start_date} to {end_date}")
        
        # Fetch candles directly from the database
        candles = db.get_candles(
            symbol_id=symbol_id,
            timeframe=timeframe,
            start_time=start_date,
            end_time=end_date
        )
        
        if not candles:
            return jsonify({"error": f"No data found for symbol {symbol_id} in the specified time range"}), 404
            
        logger.info(f"Fetched {len(candles)} candles for backtest")
        
        # Run the backtest with the candles
        result = run_backtest(candles, strategy, params)
        
        # If backtest_run_id is provided, save results to the database
        if backtest_run_id:
            metrics = result.get('metrics', {})
            
            # Save backtest results
            result_id = db.save_backtest_result(
                backtest_run_id=backtest_run_id,
                total_trades=metrics.get('total_trades', 0),
                winning_trades=metrics.get('winning_trades', 0),
                losing_trades=metrics.get('losing_trades', 0),
                profit_factor=metrics.get('profit_factor', 0),
                sharpe_ratio=metrics.get('sharpe_ratio', 0),
                max_drawdown=metrics.get('max_drawdown', 0),
                final_capital=metrics.get('final_capital', 0),
                total_return=metrics.get('total_return', 0),
                annualized_return=metrics.get('annualized_return', 0),
                results_json={
                    'equity_curve': result.get('equity_curve', []),
                    'equity_times': result.get('equity_times', [])
                }
            )
            
            logger.info(f"Saved backtest results with ID {result_id}")
            
            # Save trades
            for trade in result.get('trades', []):
                # Convert string timestamps to datetime objects
                entry_time = datetime.fromisoformat(trade['entry_time']) if isinstance(trade['entry_time'], str) else trade['entry_time']
                exit_time = datetime.fromisoformat(trade['exit_time']) if trade.get('exit_time') and isinstance(trade['exit_time'], str) else trade.get('exit_time')
                
                trade_id = db.add_backtest_trade(
                    backtest_run_id=backtest_run_id,
                    symbol_id=symbol_id,
                    entry_time=entry_time,
                    exit_time=exit_time,
                    position_type=trade['position_type'],
                    entry_price=trade['entry_price'],
                    exit_price=trade.get('exit_price'),
                    quantity=trade['quantity'],
                    profit_loss=trade.get('profit_loss'),
                    profit_loss_percent=trade.get('profit_loss_percent'),
                    exit_reason=trade.get('exit_reason')
                )
            
            logger.info(f"Saved {len(result.get('trades', []))} trades")
        
        return jsonify(result)
    except Exception as e:
        logger.exception(f"Error running backtest from DB: {str(e)}")
        return jsonify({"error": f"Failed to run backtest: {str(e)}"}), 500

@app.route('/validate-strategy', methods=['POST'])
def validate():
    """Validate a strategy structure."""
    try:
        data = request.json
        if not data:
            return jsonify({"error": "No data provided"}), 400
            
        strategy = data.get('strategy', {})
        
        # Validate strategy format
        if not strategy:
            return jsonify({"error": "No strategy provided"}), 400
            
        # Validate the strategy
        valid, message = validate_strategy(strategy)
        
        return jsonify({
            "valid": valid,
            "message": message
        })
    except Exception as e:
        logger.exception(f"Error validating strategy: {str(e)}")
        return jsonify({
            "valid": False,
            "error": f"Failed to validate strategy: {str(e)}"
        }), 500

@app.route('/indicators', methods=['GET'])
def indicators():
    """Return list of supported indicators."""
    try:
        # Get available indicators dynamically
        indicators = get_available_indicators()
        return jsonify(indicators)
    except Exception as e:
        logger.exception(f"Error fetching indicators: {str(e)}")
        return jsonify({"error": f"Failed to fetch indicators: {str(e)}"}), 500
    
@app.route('/sync-indicators', methods=['POST'])
def sync_indicators_endpoint():
    """Endpoint to sync indicators to the strategy service database."""
    try:
        # Call the sync function
        result = sync_indicators()
        
        # Return the result as JSON
        return jsonify(result)
    except Exception as e:
        logger.exception(f"Error syncing indicators: {str(e)}")
        return jsonify({"error": f"Failed to sync indicators: {str(e)}"}), 500

if __name__ == "__main__":
    # Run the Flask app for development
    app.run(host='0.0.0.0', port=5000, debug=True)