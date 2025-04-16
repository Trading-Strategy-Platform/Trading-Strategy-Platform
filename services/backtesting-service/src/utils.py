#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Utility functions for the backtesting service.
"""

import logging
import json
from typing import Dict, List, Any, Union, Optional
import pandas as pd
from datetime import datetime, timezone

logger = logging.getLogger(__name__)

def parse_iso_datetime(dt_str: str) -> Optional[datetime]:
    """
    Parse ISO datetime string into datetime object.
    Handles different ISO formats.
    
    Args:
        dt_str: ISO datetime string
        
    Returns:
        datetime object or None if parsing fails
    """
    formats = [
        "%Y-%m-%dT%H:%M:%S.%fZ",  # ISO format with microseconds and Z
        "%Y-%m-%dT%H:%M:%S.%f%z",  # ISO format with microseconds and timezone
        "%Y-%m-%dT%H:%M:%S.%f",    # ISO format with microseconds
        "%Y-%m-%dT%H:%M:%SZ",      # ISO format with Z
        "%Y-%m-%dT%H:%M:%S%z",     # ISO format with timezone
        "%Y-%m-%dT%H:%M:%S",       # ISO format
        "%Y-%m-%d %H:%M:%S.%f",    # Common format with microseconds
        "%Y-%m-%d %H:%M:%S",       # Common format
        "%Y-%m-%d",                # Date only
    ]
    
    for fmt in formats:
        try:
            dt = datetime.strptime(dt_str, fmt)
            # Make timezone-aware if it's naive
            if dt.tzinfo is None:
                dt = dt.replace(tzinfo=timezone.utc)
            return dt
        except ValueError:
            continue
    
    logger.warning(f"Failed to parse datetime string: {dt_str}")
    return None

def safe_json_serialize(obj: Any) -> Any:
    """
    Safely serialize an object to JSON, handling non-serializable types.
    
    Args:
        obj: Object to serialize
        
    Returns:
        JSON-serializable version of the object
    """
    if isinstance(obj, (datetime,)):
        return obj.isoformat()
    elif isinstance(obj, pd.DataFrame):
        return obj.to_dict(orient='records')
    elif isinstance(obj, pd.Series):
        return obj.to_dict()
    elif isinstance(obj, (int, float, str, bool, type(None))):
        return obj
    elif isinstance(obj, (list, tuple)):
        return [safe_json_serialize(item) for item in obj]
    elif isinstance(obj, dict):
        return {str(k): safe_json_serialize(v) for k, v in obj.items()}
    else:
        try:
            # Try to convert to a string representation
            return str(obj)
        except:
            return f"<Unserializable object of type {type(obj).__name__}>"

def format_error_response(error: Union[str, Exception]) -> Dict[str, str]:
    """
    Format an error response for API endpoints.
    
    Args:
        error: Error message or exception
        
    Returns:
        Formatted error response dictionary
    """
    if isinstance(error, Exception):
        return {"error": str(error), "type": type(error).__name__}
    else:
        return {"error": str(error)}

def get_time_component(dt: datetime, component: str) -> int:
    """
    Extract a specific time component from a datetime object.
    
    Args:
        dt: Datetime object
        component: Component to extract ('year', 'month', 'day', 'hour', 'minute', 'second')
        
    Returns:
        Value of the requested component
    """
    if component == 'year':
        return dt.year
    elif component == 'month':
        return dt.month
    elif component == 'day':
        return dt.day
    elif component == 'hour':
        return dt.hour
    elif component == 'minute':
        return dt.minute
    elif component == 'second':
        return dt.second
    else:
        raise ValueError(f"Unknown time component: {component}")

def group_by_time(data: List[Dict[str, Any]], time_key: str, component: str) -> Dict[str, List[Dict[str, Any]]]:
    """
    Group data by a time component.
    
    Args:
        data: List of dictionaries containing data
        time_key: Key in dictionaries that contains the datetime
        component: Time component to group by ('year', 'month', 'day', 'hour', 'minute')
        
    Returns:
        Dictionary with time component values as keys and lists of matching data as values
    """
    result = {}
    
    for item in data:
        if time_key not in item:
            continue
            
        dt_value = item[time_key]
        if isinstance(dt_value, str):
            dt = parse_iso_datetime(dt_value)
            if dt is None:
                continue
        elif isinstance(dt_value, datetime):
            dt = dt_value
        else:
            continue
            
        # Get the component value
        comp_value = get_time_component(dt, component)
        
        # Format key based on component
        if component == 'month':
            key = dt.strftime('%Y-%m')
        elif component == 'day':
            key = dt.strftime('%Y-%m-%d')
        else:
            key = str(comp_value)
            
        # Add to result
        if key not in result:
            result[key] = []
        result[key].append(item)
    
    return result