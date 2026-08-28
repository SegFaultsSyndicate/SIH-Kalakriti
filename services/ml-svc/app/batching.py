"""services/ml-svc/app/batching.py

A micro-batcher: many concurrent callers, one model call. Callers await
`submit(item)`; the worker collects items until `max_batch_size` or `max_wait_ms`,
whichever comes first, runs the batch function once, and scatters the results
back to the coroutines that are waiting for them.
"""

from __future__ import annotations

import asyncio
import logging
from typing import Awaitable, Callable, Generic, TypeVar

Item = TypeVar("Item")
Result = TypeVar("Result")

log = logging.getLogger(__name__)


class MicroBatcher(Generic[Item, Result]):
    """Batches concurrent calls to one function.

    `fn` takes a list of items and must return a list of results of the same
    length, in the same order. It may be sync or async; a sync function is run in
    a worker thread so it cannot block the event loop.
    """

    def __init__(
        self,
        fn: Callable[[list[Item]], list[Result] | Awaitable[list[Result]]],
        *,
        max_batch_size: int,
        max_wait_ms: int,
        name: str = "batch",
    ) -> None:
        self._fn = fn
        self._max_batch = max(1, max_batch_size)
        self._max_wait = max_wait_ms / 1000
        self._name = name
        self._queue: asyncio.Queue[tuple[Item, asyncio.Future[Result]]] = asyncio.Queue()
        self._worker: asyncio.Task | None = None

    async def submit(self, item: Item) -> Result:
        """Queue one item and wait for its result."""
        if self._worker is None or self._worker.done():
            # Started here rather than in __init__ so constructing a batcher does
            # not require a running loop.
            self._worker = asyncio.create_task(self._run(), name=f"batcher-{self._name}")

        future: asyncio.Future[Result] = asyncio.get_running_loop().create_future()
        self._queue.put_nowait((item, future))
        return await future

    async def submit_many(self, items: list[Item]) -> list[Result]:
        """Queue several items from one caller; they may land in different batches."""
        return list(await asyncio.gather(*(self.submit(i) for i in items)))

    async def close(self) -> None:
        if self._worker is not None:
            self._worker.cancel()
            self._worker = None

    async def _collect(self) -> list[tuple[Item, asyncio.Future[Result]]]:
        batch = [await self._queue.get()]
        loop = asyncio.get_running_loop()
        deadline = loop.time() + self._max_wait

        while len(batch) < self._max_batch:
            timeout = deadline - loop.time()
            if timeout <= 0:
                break
            try:
                batch.append(await asyncio.wait_for(self._queue.get(), timeout))
            except asyncio.TimeoutError:
                break
        return batch

    async def _run(self) -> None:
        while True:
            batch = await self._collect()
            items = [item for item, _ in batch]
            futures = [f for _, f in batch]

            try:
                if asyncio.iscoroutinefunction(self._fn):
                    results = await self._fn(items)
                else:
                    # A sync model call would otherwise block every other RPC.
                    results = await asyncio.to_thread(self._fn, items)
                if len(results) != len(items):
                    raise ValueError(
                        f"{self._name} returned {len(results)} results for {len(items)} items"
                    )
            except Exception as err:  # one bad batch must not kill the worker
                log.exception("batch failed", extra={"batcher": self._name, "size": len(items)})
                for f in futures:
                    if not f.done():
                        f.set_exception(err)
                continue

            for f, r in zip(futures, results):
                if not f.done():
                    f.set_result(r)
