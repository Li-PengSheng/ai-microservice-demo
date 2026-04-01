# service_python/observability.py
import logging
import os

from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.grpc import GrpcInstrumentorServer
from opentelemetry.propagate import set_global_textmap
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator


def setup_logging() -> logging.Logger:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
        datefmt="%Y-%m-%dT%H:%M:%S",
    )
    return logging.getLogger("python-ai")


def setup_tracing() -> TracerProvider:
    resource = Resource.create({"service.name": "python-ai"})
    provider = TracerProvider(resource=resource)

    jaeger_endpoint = os.getenv("JAEGER_ENDPOINT", "localhost:4317")
    processor = BatchSpanProcessor(
        OTLPSpanExporter(endpoint=jaeger_endpoint, insecure=True)
    )
    provider.add_span_processor(processor)
    trace.set_tracer_provider(provider)

    set_global_textmap(TraceContextTextMapPropagator())

    GrpcInstrumentorServer().instrument()

    return provider
