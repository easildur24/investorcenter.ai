"""Simple on-disk cache for downloaded ticker data."""

import json
import sys
import time
from pathlib import Path
from typing import Any, Dict, Optional


class SimpleCache:
    """Simple file-based cache with TTL support."""

    def __init__(self, cache_dir: str = ".cache"):
        self.cache_dir = Path(cache_dir)
        self.cache_dir.mkdir(exist_ok=True)

    def _get_cache_path(self, key: str) -> Path:
        """Get cache file path for a given key."""
        # Sanitize key for filename
        safe_key = "".join(c for c in key if c.isalnum() or c in "._-")
        return self.cache_dir / f"{safe_key}.json"

    def get(self, key: str, ttl_hours: Optional[int] = None) -> Optional[Any]:
        """
        Get cached value if it exists and hasn't expired.

        Args:
            key: Cache key
            ttl_hours: Time to live in hours (None = no expiration)

        Returns:
            Cached value or None if not found/expired
        """
        cache_path = self._get_cache_path(key)

        if not cache_path.exists():
            return None

        try:
            with open(cache_path, "r") as f:
                data = json.load(f)

            # Check TTL
            if ttl_hours is not None:
                cache_time = data.get("timestamp", 0)
                current_time = time.time()
                if current_time - cache_time > ttl_hours * 3600:
                    # Expired, remove file
                    cache_path.unlink()
                    return None

            value = data.get("value")

            # Handle DataFrame deserialization
            if isinstance(value, dict) and value.get("type") == "DataFrame":
                import pandas as pd

                df = pd.DataFrame(value["data"], columns=value["columns"])
                df.index = value["index"]
                return df

            return value

        except (json.JSONDecodeError, KeyError, OSError):
            # Corrupted cache file, remove it
            try:
                cache_path.unlink()
            except OSError:
                pass
            return None

    def set(self, key: str, value: Any) -> None:
        """
        Store value in cache.

        Args:
            key: Cache key
            value: Value to cache
        """
        cache_path = self._get_cache_path(key)

        try:
            # Handle pandas DataFrames specially
            if (
                hasattr(value, "__class__")
                and value.__class__.__name__ == "DataFrame"
            ):
                # Convert DataFrame to dict for JSON serialization
                serializable_value = {
                    "type": "DataFrame",
                    "data": value.to_dict("records"),
                    "columns": value.columns.tolist(),
                    "index": value.index.tolist(),
                }
            else:
                serializable_value = value

            data = {"timestamp": time.time(), "value": serializable_value}

            with open(cache_path, "w") as f:
                json.dump(data, f)

        except (OSError, TypeError) as e:
            # Log error but don't fail
            print(f"Warning: Failed to cache {key}: {e}", file=sys.stderr)

    def clear(self) -> None:
        """Clear all cached data."""
        try:
            for cache_file in self.cache_dir.glob("*.json"):
                cache_file.unlink()
        except OSError as e:
            print(f"Warning: Failed to clear cache: {e}", file=sys.stderr)

    def get_cache_info(self) -> Dict[str, Any]:
        """Get information about cached data."""
        info = {
            "cache_dir": str(self.cache_dir),
            "total_files": 0,
            "total_size_bytes": 0,
            "oldest_file": None,
            "newest_file": None,
        }

        try:
            files = list(self.cache_dir.glob("*.json"))
            info["total_files"] = len(files)

            if files:
                sizes = []
                times = []

                for f in files:
                    try:
                        stat = f.stat()
                        sizes.append(stat.st_size)
                        times.append(stat.st_mtime)
                    except OSError:
                        continue

                if sizes:
                    info["total_size_bytes"] = sum(sizes)
                    info["oldest_file"] = time.ctime(min(times))
                    info["newest_file"] = time.ctime(max(times))

        except OSError:
            pass

        return info
