"""Integration tests for IC Score data pipelines.

These tests run pipelines against a real PostgreSQL + TimescaleDB database.
They are skipped unless INTEGRATION_TEST_DB=true is set (CI provides this).
"""
