"""services/ml-svc/tests/test_server_mock.py

Every RPC over a real gRPC connection in mock mode. This is the contract the Go
side codes against: if these pass, a Go client gets valid protobuf back.

Skipped until `make proto` has generated pb/; nothing else in the suite needs it.
"""

import time

import pytest

grpc = pytest.importorskip("grpc")
pytest.importorskip("grpc_health.v1")
pb = pytest.importorskip("app.pb", reason="run `make proto` to generate pb/")

from grpc_health.v1 import health_pb2, health_pb2_grpc  # noqa: E402

from app.config import Config  # noqa: E402
from app.server import SERVICE_NAME, serve  # noqa: E402

common_pb2, inference_pb2, inference_pb2_grpc = pb.common_pb2, pb.inference_pb2, pb.inference_pb2_grpc


@pytest.fixture
async def stub():
    import asyncio
    import socket

    with socket.socket() as probe:
        probe.bind(("", 0))
        port = probe.getsockname()[1]

    cfg = Config(mock_mode=True, grpc_port=port, metrics_port=port + 1)
    started = time.perf_counter()
    task = asyncio.create_task(serve(cfg))

    channel = grpc.aio.insecure_channel(f"localhost:{port}")
    health = health_pb2_grpc.HealthStub(channel)
    for _ in range(200):
        try:
            response = await health.Check(health_pb2.HealthCheckRequest(service=SERVICE_NAME))
            if response.status == health_pb2.HealthCheckResponse.SERVING:
                break
        except grpc.RpcError:
            pass
        await asyncio.sleep(0.01)
    else:
        pytest.fail("ml-svc never reported SERVING")

    # The acceptance criterion: mock mode is up with no weights on disk in under
    # two seconds.
    assert time.perf_counter() - started < 2.0

    yield inference_pb2_grpc.InferenceServiceStub(channel)

    await channel.close()
    task.cancel()


def _media(key: str = "artisans/a/b.jpg") -> common_pb2.MediaRef:
    return common_pb2.MediaRef(
        id="018f0000-0000-7000-8000-0000000000aa",
        bucket="kalakriti-media",
        object_key=key,
        kind=common_pb2.MEDIA_KIND_IMAGE,
        mime_type="image/jpeg",
        size_bytes=4096,
        sha256_hex="4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865",
    )


async def test_enhance_image(stub):
    response = await stub.EnhanceImage(
        inference_pb2.EnhanceImageRequest(source=_media(), remove_background=True)
    )
    assert response.enhanced.object_key.endswith(".webp")
    assert "remove-background" in response.operations_applied
    assert response.model_version


async def test_extract_attributes(stub):
    response = await stub.ExtractAttributes(
        inference_pb2.ExtractAttributesRequest(
            media=[_media()], declared_craft_id="ajrakh-block-printing"
        )
    )
    attributes = response.attributes
    assert attributes.craft_id.value == "ajrakh-block-printing"
    assert 0.0 <= attributes.craft_id.confidence <= 1.0
    assert attributes.colours.values and attributes.motifs.values
    assert attributes.generated_at.seconds > 0


async def test_generate_description(stub):
    extracted = await stub.ExtractAttributes(
        inference_pb2.ExtractAttributesRequest(media=[_media()], declared_craft_id="blue-pottery")
    )
    response = await stub.GenerateDescription(
        inference_pb2.GenerateDescriptionRequest(
            attributes=extracted.attributes,
            craft_id="blue-pottery",
            language=common_pb2.LANGUAGE_ENGLISH,
            max_chars=240,
        )
    )
    assert response.title and response.description
    assert len(response.description) <= 240
    assert response.language == common_pb2.LANGUAGE_ENGLISH
    assert response.attribute_keys_used


async def test_verify_technique(stub):
    response = await stub.VerifyTechnique(
        inference_pb2.VerifyTechniqueRequest(
            media=[_media("clip.mp4")],
            claimed_technique="pit-loom-weaving",
            craft_id="assam-muga-weaving",
        )
    )
    assert response.verdict.claimed == "pit-loom-weaving"
    assert response.verdict.observed
    assert 0.0 <= response.verdict.confidence <= 1.0


async def test_detect_handloom(stub):
    response = await stub.DetectHandloom(
        inference_pb2.DetectHandloomRequest(media=_media(), declared_thread_count=120)
    )
    assert response.verdict.fft_peak_ratio > 0
    assert response.verdict.explanation
    assert response.verdict.is_handloom == (response.verdict.fft_peak_ratio < 4.0)


async def test_embed(stub):
    response = await stub.Embed(
        inference_pb2.EmbedRequest(
            texts=["ajrakh dupatta", "नीली मिट्टी के बर्तन"],
            media=[_media()],
            language=common_pb2.LANGUAGE_HINDI,
        )
    )
    assert len(response.embeddings) == 3  # texts first, then media
    for embedding in response.embeddings:
        assert embedding.dimensions == 768
        assert len(embedding.values) == 768


async def test_rerank(stub):
    response = await stub.Rerank(
        inference_pb2.RerankRequest(
            query="ajrakh stole",
            candidates=[
                inference_pb2.RerankCandidate(id="a", text="ajrakh cotton stole"),
                inference_pb2.RerankCandidate(id="b", text="brass lamp"),
            ],
            top_k=2,
        )
    )
    assert [r.id for r in response.results]
    assert response.results[0].score >= response.results[-1].score


async def test_transcribe_streams(stub):
    chunks = [
        chunk
        async for chunk in stub.Transcribe(
            inference_pb2.TranscribeRequest(
                audio=_media("note.m4a"), language=common_pb2.LANGUAGE_HINDI, interim_results=True
            )
        )
    ]
    assert chunks
    assert any(c.is_final for c in chunks)
    assert all(c.language == common_pb2.LANGUAGE_HINDI for c in chunks)


async def test_embed_batches_concurrent_callers(stub):
    import asyncio

    # Twenty concurrent single-text calls: the batcher should turn them into at
    # most two model calls, and every caller must still get its own vector.
    responses = await asyncio.gather(
        *(
            stub.Embed(inference_pb2.EmbedRequest(texts=[f"query {i}"]))
            for i in range(20)
        )
    )
    vectors = [tuple(r.embeddings[0].values) for r in responses]
    assert len(set(vectors)) == 20
