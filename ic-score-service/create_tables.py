#!/usr/bin/env python3
"""Create IC Score database tables without Alembic."""

import os
from sqlalchemy import create_engine, text

# Get database connection from environment
DB_USER = os.getenv('DB_USER', 'investorcenter')
DB_PASSWORD = os.getenv('DB_PASSWORD')
DB_HOST = os.getenv('DB_HOST', 'postgres-simple-service')
DB_PORT = os.getenv('DB_PORT', '5432')
DB_NAME = os.getenv('DB_NAME', 'investorcenter_db')

# Create connection string
DATABASE_URL = f"postgresql://{DB_USER}:{DB_PASSWORD}@{DB_HOST}:{DB_PORT}/{DB_NAME}"

print(f"Connecting to database at {DB_HOST}:{DB_PORT}/{DB_NAME}...")
engine = create_engine(DATABASE_URL, echo=True)

with engine.connect() as conn:
    # Execute the upgrade migration from migrations/versions/001_initial_schema.py
    from migrations.versions.import_001_initial_schema import upgrade

    print("Running database migrations...")
    upgrade()

    print("✅ Database tables created successfully!")
