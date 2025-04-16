#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Main Flask application for the backtesting service.
"""

import logging
from flask import Flask, request, jsonify

from src.backtest import run_backtest
from src.indicators import get_available_indicators
from src.strategies import validate_strategy

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
    """Health check endpoint."""
    return jsonify({"status": "healthy"})

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

if __name__ == "__main__":
    # Run the Flask app for development
    app.run(host='0.0.0.0', port=5000, debug=True)