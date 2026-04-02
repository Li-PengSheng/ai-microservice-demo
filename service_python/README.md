# Python AI Service (`python-ai`)

The Python gRPC backend for the AI Gateway Monitor platform. It exposes two inference services over gRPC:

- **IrisPredictor** — classifies Iris flower species using scikit-learn (RandomForest).
- **ModelPredictor** — proxies text generation requests to a local Ollama instance, supporting both unary and server-side streaming responses.

## Structure

```
service_python/
├── main.py              # Entrypoint: wires predictors, starts gRPC server
├── server.py            # gRPC server creation and graceful shutdown
├── observability.py     # Structured logging and OpenTelemetry tracing setup
├── gpu_exporter.py      # Optional Prometheus GPU metrics exporter (port 9835)
├── models/
│   ├── iris_predictor.py    # IrisPredictor gRPC servicer (scikit-learn)
│   └── ollama_predictor.py  # ModelPredictor gRPC servicer (Ollama)
├── gen/                 # Generated gRPC stubs (iris/v1, model/v1)
├── Dockerfile
└── pyproject.toml
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `OLLAMA_HOST` | `http://localhost:11434` | Ollama API base URL |
| `MODEL_NAME` | `qwen2.5:1.5b` | Ollama model to serve |
| `IRIS_MODEL_PATH` | _(unset)_ | Optional path to a pre-trained Iris pickle model. Trains in-memory if not set. |
| `JAEGER_ENDPOINT` | `localhost:4317` | OTLP gRPC endpoint for distributed trace export |

## Running locally

```bash
# Install dependencies with uv
uv sync

# Start the gRPC server (requires Ollama running on localhost:11434)
uv run main.py
```

The server listens on `[::]:50051`.

## Regenerating gRPC stubs

Stubs are generated from the shared proto definitions at the repo root:

```bash
# From the repo root
buf generate
```
