def handle_binary_conflict(base, ours, theirs):
    """For binary files, always use ours (cannot auto-merge)."""
    return {"strategy": "use_ours", "warning": "Binary conflict, using ours"}