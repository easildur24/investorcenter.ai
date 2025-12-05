FROM python:3.11-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libpq-dev \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install Python dependencies
COPY reddit/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Create non-root user
RUN useradd -m -u 1000 appuser

# Copy the reddit package
COPY --chown=appuser:appuser reddit/ /app/reddit/

# Copy the main collector script
COPY --chown=appuser:appuser reddit_sentiment_collector.py /app/

# Set ownership of app directory
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Set Python path
ENV PYTHONPATH=/app
ENV PYTHONUNBUFFERED=1

# Default command
ENTRYPOINT ["python", "/app/reddit_sentiment_collector.py"]
CMD ["--subreddits", "wallstreetbets", "stocks", "options", "investing", "Daytrading", "--limit", "100"]
