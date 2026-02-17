"""Shared test fixtures and environment setup for pipeline tests.

Sets LOG_DIR to a temp directory before any pipeline module is imported,
avoiding the OSError when modules try to create /app/logs/.
"""

import logging
import os
import tempfile

# Set LOG_DIR *before* any pipeline module is imported.
# risk_metrics_calculator and ic_score_calculator read LOG_DIR at module level.
_tmpdir = tempfile.mkdtemp(prefix="ic_test_logs_")
os.environ["LOG_DIR"] = _tmpdir

# Patch logging.FileHandler globally so modules that hardcode
# FileHandler('/app/logs/...') at module level don't fail.
_original_file_handler = logging.FileHandler


class _SafeFileHandler(logging.StreamHandler):
    """Drop-in replacement that writes to stderr instead of a file.

    Used during tests to avoid filesystem issues with /app/logs.
    """

    def __init__(self, filename=None, mode="a", encoding=None, delay=False):
        super().__init__()


logging.FileHandler = _SafeFileHandler  # type: ignore[misc]
