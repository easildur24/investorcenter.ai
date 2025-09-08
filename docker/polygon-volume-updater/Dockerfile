FROM --platform=linux/amd64 python:3.11-slim

# Set working directory
WORKDIR /app

# Install system dependencies for PostgreSQL
RUN apt-get update && apt-get install -y \
    gcc \
    libpq-dev \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
RUN pip install --no-cache-dir \
    psycopg2-binary \
    requests \
    python-dotenv

# Copy the volume update script
COPY scripts/polygon_volume_update.py .

# Create non-root user for security
RUN useradd --create-home --shell /bin/bash ticker && \
    chown -R ticker:ticker /app
USER ticker

# Set environment variables
ENV PYTHONPATH=/app
ENV PYTHONUNBUFFERED=1

# Default to updating 100 tickers, can be overridden
ENTRYPOINT ["python", "polygon_volume_update.py"]
CMD ["100"]