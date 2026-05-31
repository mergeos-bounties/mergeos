import asyncio

class BatchMerger:
    def __init__(self, max_batch=10):
        self.max_batch = max_batch
        self.pending = []
    
    async def add(self, pr):
        self.pending.append(pr)
        if len(self.pending) >= self.max_batch:
            return await self.flush()
        return None
    
    async def flush(self):
        batch = self.pending[:self.max_batch]
        self.pending = self.pending[self.max_batch:]
        return {"merged": len(batch), "prs": [p["number"] for p in batch]}