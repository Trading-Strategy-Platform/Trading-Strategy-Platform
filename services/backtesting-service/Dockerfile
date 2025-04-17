FROM continuumio/miniconda3:latest

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    curl \
    wget \
    libpq-dev \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# Create conda environment
RUN conda create -n backtesting python=3.9 -y
SHELL ["conda", "run", "-n", "backtesting", "/bin/bash", "-c"]

# Install TA-Lib through conda (much more reliable than pip)
RUN conda install -c conda-forge ta-lib -y

# Copy requirements.txt
COPY requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy the application code
COPY ./src /app/src
COPY setup.py .

# Install the application
RUN pip install -e .

# Set Python path
ENV PYTHONPATH=/app

# Set database connection environment variables (these can be overridden at runtime)
ENV HISTORICAL_DB_HOST=historical-db
ENV HISTORICAL_DB_PORT=5432
ENV HISTORICAL_DB_USER=historical_service_user
ENV HISTORICAL_DB_PASSWORD=historical_service_password
ENV HISTORICAL_DB_NAME=historical_service

# Expose the application port
EXPOSE 5000

# Command to run the application with Gunicorn for production
CMD ["conda", "run", "--no-capture-output", "-n", "backtesting", "gunicorn", "-b", "0.0.0.0:5000", "--timeout", "120", "src.app:app"]