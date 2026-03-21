import logging
import os
import sys
from concurrent import futures

# Fix import path so generated code under 'gen/' is discoverable.
sys.path.append(os.path.join(os.path.dirname(__file__), "gen"))
import grpc
import ollama
from iris.v1 import iris_pb2, iris_pb2_grpc
from model.v1 import model_pb2, model_pb2_grpc
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.grpc import GrpcInstrumentorServer
from opentelemetry.propagate import set_global_textmap
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator
from sklearn.datasets import load_iris
from sklearn.ensemble import RandomForestClassifier

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("python-ai")

# 1. Initialise tracer and export spans to Jaeger.
resource = Resource.create({"service.name": "python-ai"})
provider = TracerProvider(resource=resource)
jaeger_endpoint = os.getenv("JAEGER_ENDPOINT", "localhost:4317")
processor = BatchSpanProcessor(OTLPSpanExporter(endpoint=jaeger_endpoint, insecure=True))
provider.add_span_processor(processor)
trace.set_tracer_provider(provider)

# W3C TraceContext propagation for cross-service trace correlation.
set_global_textmap(TraceContextTextMapPropagator())

# 2. Auto-instrument all gRPC server calls.
instrumentor = GrpcInstrumentorServer()
instrumentor.instrument()

# Train Iris classifier once at startup.
iris = load_iris()
clf = RandomForestClassifier()
clf.fit(iris.data, iris.target)
logger.info("Iris RandomForestClassifier trained successfully")


class IrisPredictor(iris_pb2_grpc.IrisPredictorServicer):
    def IrisPredict(self, request, context):
        logger.info(
            "IrisPredict request: sepal_len=%.2f sepal_wid=%.2f "
            "petal_len=%.2f petal_wid=%.2f",
            request.sepal_length,
            request.sepal_width,
            request.petal_length,
            request.petal_width,
        )

        features = [
            [
                request.sepal_length,
                request.sepal_width,
                request.petal_length,
                request.petal_width,
            ]
        ]

        pred_idx = clf.predict(features)[0]
        class_name = iris.target_names[pred_idx]
        logger.info("IrisPredict result: class_id=%d class_name=%s", pred_idx, class_name)

        return iris_pb2.IrisPredictResponse(class_id=pred_idx, class_name=class_name)


# LLM inference via Ollama
class ModelPredictor(model_pb2_grpc.ModelPredictorServicer):
    def __init__(self):
        host = os.getenv("OLLAMA_HOST", "http://localhost:11434")
        self.client = ollama.Client(host=host)
        logger.info("ModelPredictor initialised with Ollama host: %s", host)

    def ModelPredict(self, request, context):
        logger.info("ModelPredict prompt: %.80s...", request.prompt)
        response = self.client.generate(model="qwen2.5:1.5b", prompt=request.prompt)
        logger.info(
            "ModelPredict done: eval_count=%s eval_duration=%sns",
            response.eval_count,
            response.eval_duration,
        )

        return model_pb2.ModelPredictResponse(
            response=response.response,
            model_name="qwen2.5-1.5b",
            prompt_eval_count=response.prompt_eval_count or 0,
            eval_count=response.eval_count or 0,
            eval_duration=response.eval_duration or 0,
        )


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    iris_pb2_grpc.add_IrisPredictorServicer_to_server(IrisPredictor(), server)
    model_pb2_grpc.add_ModelPredictorServicer_to_server(ModelPredictor(), server)
    server.add_insecure_port("[::]:50051")
    logger.info("Python AI Service (gRPC) listening on port 50051")
    server.start()
    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        logger.info("Shutting down gracefully...")
        server.stop(grace=5)
        provider.shutdown()
        logger.info("Server stopped.")


if __name__ == "__main__":
    serve()
