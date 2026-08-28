"""services/ml-svc/tests/test_batching.py"""

import asyncio
import time

from app.batching import MicroBatcher


async def test_groups_concurrent_calls_into_few_model_calls():
    calls: list[int] = []

    def double(items: list[int]) -> list[int]:
        calls.append(len(items))
        time.sleep(0.01)  # a model call is not instant
        return [i * 2 for i in items]

    batcher = MicroBatcher(double, max_batch_size=16, max_wait_ms=50, name="test")
    results = await asyncio.gather(*(batcher.submit(i) for i in range(20)))

    # The acceptance criterion: 20 concurrent calls, at most 2 model calls.
    assert len(calls) <= 2, calls
    assert sum(calls) == 20
    # And every caller got its own answer, not its neighbour's.
    assert results == [i * 2 for i in range(20)]
    await batcher.close()


async def test_flushes_on_the_wait_deadline():
    calls: list[int] = []

    async def echo(items: list[str]) -> list[str]:
        calls.append(len(items))
        return items

    batcher = MicroBatcher(echo, max_batch_size=64, max_wait_ms=20, name="test")
    started = time.perf_counter()
    assert await batcher.submit("only") == "only"
    elapsed = time.perf_counter() - started

    # One item must not wait for 63 friends that never come.
    assert calls == [1]
    assert elapsed < 1.0
    await batcher.close()


async def test_a_failed_batch_fails_only_its_own_callers():
    attempts: list[int] = []

    def explode_once(items: list[int]) -> list[int]:
        attempts.append(len(items))
        if len(attempts) == 1:
            raise RuntimeError("model fell over")
        return items

    batcher = MicroBatcher(explode_once, max_batch_size=4, max_wait_ms=10, name="test")

    results = await asyncio.gather(
        *(batcher.submit(i) for i in range(4)), return_exceptions=True
    )
    assert all(isinstance(r, RuntimeError) for r in results)

    # The worker survived, so the next caller is served normally.
    assert await batcher.submit(99) == 99


async def test_a_short_result_list_is_an_error_not_a_mismatch():
    batcher = MicroBatcher(lambda items: items[:-1], max_batch_size=8, max_wait_ms=10, name="test")
    results = await asyncio.gather(
        *(batcher.submit(i) for i in range(3)), return_exceptions=True
    )
    # Silently pairing the wrong result with the wrong caller is the one failure
    # mode a batcher must never have.
    assert all(isinstance(r, ValueError) for r in results)
