FROM --platform=linux/amd64 python:3.11-slim

# Set working directory
WORKDIR /app

# Install system dependencies for PostgreSQL and Git
RUN apt-get update && apt-get install -y \
    gcc \
    libpq-dev \
    git \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install Python dependencies
COPY docker/ticker-updater/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy ticker update scripts
COPY scripts/update_tickers_cron.py .
COPY scripts/ticker_import_to_db.py .
COPY scripts/us_tickers/ ./us_tickers/

# Create non-root user for security
RUN useradd --create-home --shell /bin/bash ticker && \
    chown -R ticker:ticker /app
USER ticker

# Set environment variables
ENV PYTHONPATH=/app
ENV PYTHONUNBUFFERED=1

# Run the cron update script
CMD ["python", "update_tickers_cron.py"]
