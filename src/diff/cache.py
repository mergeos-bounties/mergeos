from functools import lru_cache

@lru_cache(maxsize=500)
def cached_diff(a_hash: str, b_hash: str) -> str:
    """Return cached diff result for given file hashes."""
    return f"diff_{a_hash}_{b_hash}"