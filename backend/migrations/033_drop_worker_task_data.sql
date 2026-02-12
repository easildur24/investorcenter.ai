-- Migration 033: Drop worker_task_data table
-- Worker-collected data is now stored in S3 (bucket: claw-treasure, prefix: worker-data/)
-- instead of PostgreSQL for better scalability and cost efficiency.

DROP TABLE IF EXISTS worker_task_data;
