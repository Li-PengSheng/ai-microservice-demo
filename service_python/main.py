import os
import sys
import time
from concurrent import futures

# --- 关键：解决 Import 路径问题 ---
# 把 'gen' 目录加入到 Python 搜索路径，这样就能直接 import 生成的代码了
sys.path.append(os.path.join(os.path.dirname(__file__), "gen"))
import grpc
import iris_pb2
import iris_pb2_grpc
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.grpc import GrpcInstrumentorServer
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from sklearn.datasets import load_iris
from sklearn.ensemble import RandomForestClassifier

# 1. 初始化追踪器，发送到 Jaeger
# 配置追踪数据发送到 Jaeger:4317
resource = trace.Resource.create({"service.name": "python-ai"})
provider = TracerProvider(resource=resource)
processor = BatchSpanProcessor(OTLPSpanExporter(endpoint="jaeger:4317", insecure=True))
provider.add_span_processor(processor)
trace.set_tracer_provider(provider)

# 2. 自动拦截所有 gRPC 请求
instrumentor = GrpcInstrumentorServer()
instrumentor.instrument()

# 1. 模拟加载/训练模型 (实际项目中你会 load 一个 .pkl 文件)
iris = load_iris()
clf = RandomForestClassifier()
clf.fit(iris.data, iris.target)

class IrisPredictorServicer(iris_pb2_grpc.IrisPredictorServicer):
    def Predict(self, request, context):
        print(f"收到请求: sepal_len={request.sepal_length}, sepal_wid={request.sepal_width}...")

        features = [[
                    request.sepal_length,
                    request.sepal_width,
                    request.petal_length,
                    request.petal_width
                ]]

        pred_idx = clf.predict(features)[0]
        class_name = iris.target_names[pred_idx]

        # class_id = 0
        # class_name = "Setosa"

        # if request.petal_length > 5.0:
        #     class_id = 2
        #     class_name = "Virginica"
        # elif request.petal_length > 3.0:
        #     class_id = 1
        #     class_name = "Versicolor"

        return iris_pb2.PredictResponse(class_id=pred_idx, class_name=class_name)

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    iris_pb2_grpc.add_IrisPredictorServicer_to_server(IrisPredictorServicer(), server)
    server.add_insecure_port('[::]:50051')
    print("Python AI Service (gRPC) is running on port 50051...")
    server.start()
    server.wait_for_termination()

if __name__ == '__main__':
    serve()

# curl -X POST http://localhost:8080/predict -H "Content-Type: application/json" -d '{"sepal_length": 6.0, "sepal_width": 3.0, "petal_length": 5.5, "petal_width": 2.0}'
