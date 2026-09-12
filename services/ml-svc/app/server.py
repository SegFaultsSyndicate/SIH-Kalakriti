"""services/ml-svc/app/server.py

The gRPC surface. Protobuf is converted to plain Python here and nowhere else,
so the model backends never import a generated module and stay unit-testable.

Startup order matters: the server begins listening immediately but reports
NOT_SERVING until the registry has finished loading, so an orchestrator does not
route traffic at a half-loaded process.
"""

from __future__ import annotations

import asyncio
import json
import logging
import signal
import sys
import time

import grpc
from grpc_health.v1 import health, health_pb2, health_pb2_grpc
from grpc_reflection.v1alpha import reflection
from prometheus_client import Counter, Histogram, start_http_server

from app import models
from app.batching import MicroBatcher
from app.config import Config
from app.pb import common_pb2, inference_pb2, inference_pb2_grpc

SERVICE_NAME = "inference.v1.InferenceService"

RPC_LATENCY = Histogram(
    "ml_svc_rpc_seconds", "RPC wall time in seconds", ["method", "code"],
    buckets=(0.005, 0.025, 0.1, 0.25, 1, 2.5, 10, 30),
)
RPC_TOTAL = Counter("ml_svc_rpc_total", "RPCs served", ["method", "code"])
BATCH_SIZE = Histogram(
    "ml_svc_batch_size", "Items per model call", ["batcher"], buckets=(1, 2, 4, 8, 16, 32),
)

log = logging.getLogger("ml-svc")


class JSONFormatter(logging.Formatter):
    """One JSON object per line, the same shape pkg/logger emits on the Go side."""

    def format(self, record: logging.LogRecord) -> str:
        payload = {
            "time": self.formatTime(record, "%Y-%m-%dT%H:%M:%S%z"),
            "level": record.levelname,
            "msg": record.getMessage(),
            "logger": record.name,
        }
        # Anything passed as extra= rides along, which is how RPC timings get out.
        for key, value in record.__dict__.items():
            if key not in logging.LogRecord("", 0, "", 0, "", (), None).__dict__ and key != "taskName":
                payload[key] = value
        if record.exc_info:
            payload["error"] = self.formatException(record.exc_info)
        return json.dumps(payload, ensure_ascii=False)


def setup_logging(level: str) -> None:
    handler = logging.StreamHandler(sys.stdout)
    handler.setFormatter(JSONFormatter())
    logging.basicConfig(level=level, handlers=[handler], force=True)


class ObservabilityInterceptor(grpc.aio.ServerInterceptor):
    """Times every RPC, counts it by status, and logs one line per call."""

    async def intercept_service(self, continuation, handler_call_details):
        handler = await continuation(handler_call_details)
        if handler is None:
            return None

        method = handler_call_details.method.rsplit("/", 1)[-1]

        def observe(started: float, code: str) -> None:
            elapsed = time.perf_counter() - started
            RPC_LATENCY.labels(method, code).observe(elapsed)
            RPC_TOTAL.labels(method, code).inc()
            log.info("rpc", extra={"method": method, "code": code, "ms": round(elapsed * 1000, 2)})

        if handler.unary_unary:
            inner = handler.unary_unary

            async def unary(request, context):
                started = time.perf_counter()
                try:
                    response = await inner(request, context)
                except grpc.RpcError:
                    observe(started, "ERROR")
                    raise
                except Exception:
                    observe(started, "INTERNAL")
                    raise
                observe(started, str(context.code() or grpc.StatusCode.OK).rsplit(".", 1)[-1])
                return response

            return grpc.unary_unary_rpc_method_handler(
                unary,
                request_deserializer=handler.request_deserializer,
                response_serializer=handler.response_serializer,
            )

        if handler.unary_stream:
            inner_stream = handler.unary_stream

            async def stream(request, context):
                started = time.perf_counter()
                try:
                    async for item in inner_stream(request, context):
                        yield item
                finally:
                    observe(started, "OK")

            return grpc.unary_stream_rpc_method_handler(
                stream,
                request_deserializer=handler.request_deserializer,
                response_serializer=handler.response_serializer,
            )

        return handler


class InferenceServicer(inference_pb2_grpc.InferenceServiceServicer):
    """Converts protobuf to plain Python, calls the registry, converts back."""

    def __init__(self, registry: models.Models, cfg: Config) -> None:
        self._models = registry
        self._cfg = cfg
        # Embed and ExtractAttributes are the two RPCs that arrive in bursts —
        # one per listing image, one per search query — so they are the two that
        # go through the batcher.
        self._embed_batcher: MicroBatcher[str, list[float]] = MicroBatcher(
            self._embed_batch,
            max_batch_size=cfg.max_batch_size,
            max_wait_ms=cfg.max_wait_ms,
            name="embed",
        )
        self._extract_batcher: MicroBatcher[tuple, dict] = MicroBatcher(
            self._extract_batch,
            max_batch_size=cfg.max_batch_size,
            max_wait_ms=cfg.max_wait_ms,
            name="extract",
        )

    async def _embed_batch(self, texts: list[str]) -> list[list[float]]:
        BATCH_SIZE.labels("embed").observe(len(texts))
        return await self._models.embed(texts)

    async def _extract_batch(self, items: list[tuple]) -> list[dict]:
        BATCH_SIZE.labels("extract").observe(len(items))
        # The VLM takes one product's images at a time; batching here still wins,
        # because it is one scheduled call instead of N interleaved ones.
        return list(
            await asyncio.gather(*(self._models.extract_attributes(*item) for item in items))
        )

    async def close(self) -> None:
        await self._embed_batcher.close()
        await self._extract_batcher.close()

    # --- RPCs ---------------------------------------------------------------

    async def EnhanceImage(self, request, context):
        key, ops = await self._models.enhance_image(
            request.source.object_key,
            request.remove_background,
            request.auto_white_balance,
            request.upscale_factor or 1,
            request.correct_lighting,
        )
        enhanced = common_pb2.MediaRef()
        enhanced.CopyFrom(request.source)
        enhanced.object_key = key
        enhanced.mime_type = "image/webp"
        return inference_pb2.EnhanceImageResponse(
            enhanced=enhanced, operations_applied=ops, model_version=self._models.version
        )

    async def ExtractAttributes(self, request, context):
        attributes = await self._extract_batcher.submit(
            (
                [m.object_key for m in request.media],
                request.declared_craft_id or "",
                request.artisan_hint or "",
            )
        )
        return inference_pb2.ExtractAttributesResponse(
            attributes=_attributes_to_proto(attributes, self._models.version)
        )

    async def GenerateDescription(self, request, context):
        copy = await self._models.generate_description(
            _attributes_from_proto(request.attributes),
            request.craft_id,
            _language_name(request.language),
            request.artisan_note or "",
            request.max_chars or 0,
        )
        return inference_pb2.GenerateDescriptionResponse(
            title=copy["title"],
            description=copy["description"],
            highlights=copy["highlights"],
            keywords=copy["keywords"],
            attribute_keys_used=copy["attribute_keys_used"],
            language=request.language,
            model_version=self._models.version,
        )

    async def VerifyTechnique(self, request, context):
        verdict = await self._models.verify_technique(
            [m.object_key for m in request.media], request.claimed_technique, request.craft_id
        )
        return inference_pb2.VerifyTechniqueResponse(
            verdict=inference_pb2.TechniqueVerdict(
                claimed=verdict["claimed"],
                observed=verdict["observed"],
                matches=verdict["matches"],
                confidence=verdict["confidence"],
                explanation=verdict["explanation"],
                model_version=self._models.version,
            )
        )

    async def DetectHandloom(self, request, context):
        verdict = await self._models.detect_handloom(
            request.media.object_key, request.declared_thread_count or 0
        )
        return inference_pb2.DetectHandloomResponse(
            verdict=inference_pb2.LoomVerdict(
                is_handloom=verdict["is_handloom"],
                confidence=verdict["confidence"],
                fft_peak_ratio=verdict["fft_peak_ratio"],
                explanation=verdict["explanation"],
                model_version=self._models.version,
            )
        )

    async def Embed(self, request, context):
        # Media embedding shares the text tower via its caption-free object key,
        # so both arrive as strings and come back in request order.
        inputs = list(request.texts) + [m.object_key for m in request.media]
        vectors = await self._embed_batcher.submit_many(inputs)
        return inference_pb2.EmbedResponse(
            embeddings=[
                inference_pb2.Embedding(
                    values=v, dimensions=len(v), model_version=self._models.version
                )
                for v in vectors
            ]
        )

    async def Rerank(self, request, context):
        scored = await self._models.rerank(
            request.query, [(c.id, c.text) for c in request.candidates]
        )
        if request.top_k:
            scored = scored[: request.top_k]
        return inference_pb2.RerankResponse(
            results=[inference_pb2.RerankResult(id=cid, score=score) for cid, score in scored],
            model_version=self._models.version,
        )

    async def Transcribe(self, request, context):
        async for chunk in self._models.transcribe(
            request.audio.object_key, _language_name(request.language), request.interim_results
        ):
            yield inference_pb2.TranscribeResponse(
                text=chunk["text"],
                is_final=chunk["is_final"],
                start_ms=chunk["start_ms"],
                end_ms=chunk["end_ms"],
                confidence=chunk["confidence"],
                language=request.language,
            )


# --- protobuf conversion ----------------------------------------------------


def _language_name(language: int) -> str:
    """LANGUAGE_HINDI -> HINDI, which is what the models and the DB enum use."""
    return common_pb2.Language.Name(language).removeprefix("LANGUAGE_")


def _attributes_to_proto(attributes: dict, version: str) -> inference_pb2.AttributeSet:
    from google.protobuf.timestamp_pb2 import Timestamp

    now = Timestamp()
    now.GetCurrentTime()

    craft, craft_conf = attributes["craft_id"]
    material, material_conf = attributes["material"]
    technique, technique_conf = attributes["technique"]
    colours, colours_conf = attributes["colours"]
    motifs, motifs_conf = attributes["motifs"]
    dimensions, dimensions_conf = attributes["dimensions"]

    return inference_pb2.AttributeSet(
        craft_id=inference_pb2.ScoredString(value=craft, confidence=craft_conf),
        material=inference_pb2.ScoredString(value=material, confidence=material_conf),
        technique=inference_pb2.ScoredString(value=technique, confidence=technique_conf),
        colours=inference_pb2.ScoredStrings(values=colours, confidence=colours_conf),
        motifs=inference_pb2.ScoredStrings(values=motifs, confidence=motifs_conf),
        dimensions=inference_pb2.ScoredDimensions(
            value=common_pb2.Dimensions(**{k: v for k, v in (dimensions or {}).items()}),
            confidence=dimensions_conf,
        ),
        model_version=version,
        generated_at=now,
    )


def _attributes_from_proto(attributes: inference_pb2.AttributeSet) -> dict:
    return {
        "craft_id": (attributes.craft_id.value, attributes.craft_id.confidence),
        "material": (attributes.material.value, attributes.material.confidence),
        "technique": (attributes.technique.value, attributes.technique.confidence),
        "colours": (list(attributes.colours.values), attributes.colours.confidence),
        "motifs": (list(attributes.motifs.values), attributes.motifs.confidence),
        "dimensions": (
            {
                f.name: getattr(attributes.dimensions.value, f.name)
                for f in attributes.dimensions.value.DESCRIPTOR.fields
                if attributes.dimensions.value.HasField(f.name)
            },
            attributes.dimensions.confidence,
        ),
    }


# --- lifecycle --------------------------------------------------------------


async def serve(cfg: Config) -> None:
    registry = models.build(cfg)
    servicer = InferenceServicer(registry, cfg)

    server = grpc.aio.server(interceptors=[ObservabilityInterceptor()])
    inference_pb2_grpc.add_InferenceServiceServicer_to_server(servicer, server)

    health_servicer = health.aio.HealthServicer()
    health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)
    # NOT_SERVING until the weights are in memory: an orchestrator that routes to
    # a half-loaded process turns a slow start into a wave of failed RPCs.
    await health_servicer.set(SERVICE_NAME, health_pb2.HealthCheckResponse.NOT_SERVING)
    await health_servicer.set("", health_pb2.HealthCheckResponse.NOT_SERVING)

    reflection.enable_server_reflection(
        (SERVICE_NAME, health.SERVICE_NAME, reflection.SERVICE_NAME), server
    )

    server.add_insecure_port(f"[::]:{cfg.grpc_port}")
    await server.start()
    start_http_server(cfg.metrics_port)
    log.info(
        "listening",
        extra={"grpc_port": cfg.grpc_port, "metrics_port": cfg.metrics_port, "mock_mode": cfg.mock_mode},
    )

    started = time.perf_counter()
    await registry.load()
    await health_servicer.set(SERVICE_NAME, health_pb2.HealthCheckResponse.SERVING)
    await health_servicer.set("", health_pb2.HealthCheckResponse.SERVING)
    log.info("ready", extra={"load_seconds": round(time.perf_counter() - started, 3)})

    stop = asyncio.Event()
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, stop.set)
    await stop.wait()

    log.info("draining")
    await health_servicer.set(SERVICE_NAME, health_pb2.HealthCheckResponse.NOT_SERVING)
    await servicer.close()
    await server.stop(grace=10)


def main() -> None:
    cfg = Config.from_env()
    setup_logging(cfg.log_level)
    asyncio.run(serve(cfg))


if __name__ == "__main__":
    main()
