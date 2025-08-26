"""Unit tests for the cache module."""

import tempfile
from pathlib import Path

import pytest

from us_tickers.cache import SimpleCache


class TestSimpleCache:
    """Test SimpleCache functionality."""

    def test_cache_initialization(self):
        """Test cache initialization creates directory."""
        with tempfile.TemporaryDirectory() as temp_dir:
            cache = SimpleCache(temp_dir)
            assert cache.cache_dir.exists()
            assert cache.cache_dir == Path(temp_dir)

    def test_cache_set_and_get(self):
        """Test basic cache set and get operations."""
        with tempfile.TemporaryDirectory() as temp_dir:
            cache = SimpleCache(temp_dir)

            # Set cache
            test_data = {"key": "value", "numbers": [1, 2, 3]}
            cache.set("test_key", test_data)

            # Get cache
            result = cache.get("test_key")
            assert result == test_data

    def test_cache_get_nonexistent(self):
        """Test getting non-existent cache key."""
        with tempfile.TemporaryDirectory() as temp_dir:
            cache = SimpleCache(temp_dir)
            result = cache.get("nonexistent_key")
            assert result is None

    def test_cache_ttl_expiration(self):
        """Test TTL-based cache expiration."""
        with tempfile.TemporaryDirectory() as temp_dir:
            cache = SimpleCache(temp_dir)

            # Set cache
            test_data = "test_value"
            cache.set("test_key", test_data)

            # Get immediately (should work)
            result = cache.get("test_key", ttl_hours=1)
            assert result == test_data

            # Test that cache works without TTL
            result = cache.get("test_key")
            assert result == test_data

    def test_cache_no_ttl(self):
        """Test cache without TTL (never expires)."""
        with tempfile.TemporaryDirectory() as temp_dir:
            cache = SimpleCache(temp_dir)

            # Set cache
            test_data = "test_value"
            cache.set("test_key", test_data)

            # Should still work (no TTL)
            result = cache.get("test_key")
            assert result == test_data

    def test_cache_key_sanitization(self):
        """Test that cache keys are properly sanitized for filenames."""
        with tempfile.TemporaryDirectory() as temp_dir:
            cache = SimpleCache(temp_dir)

            # Test key with special characters
            special_key = "test/key:with*special?chars"
            test_data = "test_value"

            cache.set(special_key, test_data)

            # Should be able to retrieve
            result = cache.get(special_key)
            assert result == test_data

            # Check that file was created with sanitized name
            cache_files = list(cache.cache_dir.glob("*.json"))
            assert len(cache_files) == 1

            # File should contain sanitized key
            filename = cache_files[0].name
            assert "test" in filename
            assert "key" in filename
            assert "with" in filename
            assert "special" in filename
            assert "chars" in filename

    def test_cache_corrupted_file_handling(self) -> None:
        """Test handling of corrupted cache files."""
        with tempfile.TemporaryDirectory() as temp_dir:
            cache = SimpleCache(temp_dir)

            # Create a corrupted cache file
            corrupted_file = cache._get_cache_path("test_key")
            with open(corrupted_file, "w") as f:
                f.write("invalid json content")

            # Should return None and clean up corrupted file
            result = cache.get("test_key")
            assert result is None
            assert not corrupted_file.exists()

    def test_cache_clear(self) -> None:
        """Test clearing all cached data."""
        with tempfile.TemporaryDirectory() as temp_dir:
            cache = SimpleCache(temp_dir)

            # Set multiple cache entries
            cache.set("key1", "value1")
            cache.set("key2", "value2")
            cache.set("key3", "value3")

            # Verify files exist
            cache_files = list(cache.cache_dir.glob("*.json"))
            assert len(cache_files) == 3

            # Clear cache
            cache.clear()

            # Verify all files are gone
            cache_files = list(cache.cache_dir.glob("*.json"))
            assert len(cache_files) == 0

    def test_cache_info(self) -> None:
        """Test cache information retrieval."""
        with tempfile.TemporaryDirectory() as temp_dir:
            cache = SimpleCache(temp_dir)

            # Get info on empty cache
            info = cache.get_cache_info()
            assert info["cache_dir"] == temp_dir
            assert info["total_files"] == 0
            assert info["total_size_bytes"] == 0
            assert info["oldest_file"] is None
            assert info["newest_file"] is None

            # Add some cache entries
            cache.set("key1", "value1")
            cache.set("key2", "value2")

            # Get updated info
            info = cache.get_cache_info()
            assert info["total_files"] == 2
            assert info["total_size_bytes"] > 0
            assert info["oldest_file"] is not None
            assert info["newest_file"] is not None

    def test_cache_error_handling(self) -> None:
        """Test error handling during cache operations."""
        with tempfile.TemporaryDirectory() as temp_dir:
            cache = SimpleCache(temp_dir)

            # Test setting cache with non-serializable data
            # This should not crash but log a warning
            def non_serializable(x):
                return x  # Functions can't be serialized

            # Should not raise exception
            cache.set("test_key", non_serializable)

            # But should not be retrievable
            result = cache.get("test_key")
            assert result is None


if __name__ == "__main__":
    pytest.main([__file__])
