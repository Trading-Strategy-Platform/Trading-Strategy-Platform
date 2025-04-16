FROM continuumio/miniconda3:latest

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    curl \
    wget \
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

# Expose the application port
EXPOSE 5000

# Command to run the application with Gunicorn for production
CMD ["conda", "run", "--no-capture-output", "-n", "backtesting", "gunicorn", "-b", "0.0.0.0:5000", "--timeout", "120", "src.app:app"]