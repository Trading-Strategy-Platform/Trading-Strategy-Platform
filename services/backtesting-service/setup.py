#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Setup script for the backtesting service package.
"""

from setuptools import setup, find_packages

setup(
    name="backtesting_service",
    version="1.0.0",
    packages=find_packages(),
    install_requires=[
        "flask>=2.0.1",
        "gunicorn>=20.1.0",
        "numpy>=1.21.2",
        "pandas>=1.3.3",
        "pandas-ta>=0.3.14b0",
        "backtesting>=0.3.3",
        "python-dateutil>=2.8.2",
        "pytz>=2021.3",
    ],
    author="Trading System Team",
    author_email="tradingsystem@example.com",
    description="Backtesting service for trading strategy evaluation",
    keywords="trading, backtesting, finance",
    classifiers=[
        "Development Status :: 3 - Alpha",
        "Intended Audience :: Financial and Insurance Industry",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.9",
    ],
    python_requires=">=3.9",
)