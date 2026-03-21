# AI Microservice Demo

A production-style AI microservice system built as a CV portfolio project. It demonstrates a **Go API gateway** routing requests over **gRPC** to a **Python AI service**, with full observability (Prometheus, Grafana, Jaeger) and Kubernetes deployment with auto-scaling.

## Architecture

```
                         ┌─────────────────────────────────────┐
                         │          Monitoring Stack           │
                         │  Prometheus  Grafana  Jaeger (OTel) │
                         └──────────┬──────────────────────────┘
                                    │ scrape / trace
             HTTP                   │
Client ──────────────▶  Go Gateway (Gin)  :8080
                              │
                              │  gRPC (protobuf)
                              ▼
                       Python AI Service  :50051
                        ├── Iris Classifier  (scikit-learn RandomForest)
                        └── LLM Inference    (Ollama / qwen2.5:1.5b)
```

## Tech Stack

| Layer | Technology |
|---|---|
| API Gateway | Go 1.25, Gin, gRPC client |
| AI Service | Python 3.12, gRPC server, scikit-learn, Ollama |
| IDL | Protocol Buffers v3, Buf |
| Metrics | Prometheus client, custom histograms & counters |
| Tracing | OpenTelemetry SDK → Jaeger (OTLP/gRPC) |
| Profiling | Go pprof (`:6060/debug/pprof`) |
| Container | Docker, Docker Compose |
| Orchestration | Kubernetes, HPA (custom metrics via Prometheus Adapter) |
| Load testing | k6 |

## Features

- **REST → gRPC gateway**: Gin HTTP handler translates JSON requests into protobuf messages forwarded over gRPC.
- **Iris flower classification**: RandomForestClassifier trained on the UCI Iris dataset. Three classes: *setosa*, *versicolor*, *virginica*.
- **LLM inference**: Streams a prompt to a locally running Ollama instance (`qwen2.5:1.5b`) and returns the generated text along with token counts and generation duration.
- **Prometheus metrics**: QPS, P99 latency histograms, gRPC call durations, AI token throughput, and GPU utilisation (via `nvidia_gpu_exporter`).
- **Distributed tracing**: End-to-end trace propagation from the Go gateway through to the Python service using W3C TraceContext headers.
- **Horizontal Pod Autoscaler**: Scales the gateway deployment (2–10 replicas) based on the P99 latency custom metric exposed by Prometheus Adapter.
- **Object pooling**: `sync.Pool` reuses proto request objects in the hot path to reduce GC pressure.
- **Health check endpoint**: `/healthz` for Kubernetes liveness / readiness probes.

## Project Layout

```
.
├── proto/                   # Protobuf IDL definitions
│   ├── iris/v1/iris.proto
│   └── model/v1/model.proto
├── service_go/              # Go API gateway
│   ├── main.go
│   ├── gen/                 # Generated Go gRPC stubs
│   └── Dockerfile
├── service_python/          # Python AI service
│   ├── main.py
│   ├── gen/                 # Generated Python gRPC stubs
│   └── Dockerfile
├── test/
│   └── test.js              # k6 load-test script
├── go-gateway.yaml          # K8s Deployment + Service
├── python-ai.yaml           # K8s Deployment + Service
├── go-gateway-hpa.yaml      # HorizontalPodAutoscaler
├── go-gateway-monitor.yaml  # Prometheus ServiceMonitor (kube-prometheus)
├── adapter-values.yaml      # Prometheus Adapter custom metrics rule
├── ollama-svc.yaml          # K8s ExternalName service pointing to Ollama
├── docker-compose.yml       # Local dev stack
├── prometheus.yml           # Prometheus scrape config
└── deploy.sh                # One-shot build + deploy helper
```

## Quick Start (Docker Compose)

> **Prerequisites**: Docker, Docker Desktop (or Docker Engine), Ollama running locally with `qwen2.5:1.5b` pulled.

```bash
# 1. Pull the LLM model (one-time)
ollama pull qwen2.5:1.5b

# 2. Start all services
docker compose up --build

# 3. Test the Iris endpoint
curl -X POST http://localhost:8080/predict/iris \
  -H 'Content-Type: application/json' \
  -d '{"sepal_length":6.0,"sepal_width":3.0,"petal_length":5.5,"petal_width":2.0}'
# → {"id":2,"result":"virginica","source":"Go Gateway -> Python AI (Iris)"}

# 4. Test the LLM endpoint
curl -X POST http://localhost:8080/predict/model \
  -H 'Content-Type: application/json' \
  -d '{"prompt":"What is machine learning in one sentence?"}'
# → {"metrics":{...},"model":"qwen2.5-1.5b","reply":"...","source":"..."}

# 5. View Prometheus metrics
open http://localhost:9090

# 6. View Grafana dashboards
open http://localhost:3000   # admin / admin

# 7. View Jaeger traces
open http://localhost:16686
```

## Kubernetes Deployment

> **Prerequisites**: `kubectl`, a local cluster (e.g., minikube / k3s), and images built & loaded.

```bash
# Full deploy (build images, load into cluster, apply manifests, port-forward)
./deploy.sh

# Or step by step:
./deploy.sh build    # docker build
./deploy.sh apply    # kubectl apply
./deploy.sh monitor  # start docker compose monitoring stack
./deploy.sh forward  # port-forward gateway → localhost:8080
./deploy.sh status   # show pod / HPA status
./deploy.sh logs     # tail logs
./deploy.sh reset    # tear everything down
```

## Regenerating Protobuf Stubs

```bash
# Install Buf CLI: https://buf.build/docs/installation
buf generate
```

## Load Testing

```bash
# Requires Docker
docker run --rm -i --network host grafana/k6 run - < test/test.js
```

The k6 script ramps from 10 → 30 VUs and asserts:
- Iris p95 latency < 500 ms
- LLM p95 latency < 30 s
- Overall HTTP error rate < 1 %

## API Reference

### `POST /predict/iris`

Classify an Iris flower sample.

**Request**
```json
{
  "sepal_length": 5.1,
  "sepal_width":  3.5,
  "petal_length": 1.4,
  "petal_width":  0.2
}
```

**Response**
```json
{
  "result": "setosa",
  "id":     0,
  "source": "Go Gateway -> Python AI (Iris)"
}
```

### `POST /predict/model`

Send a prompt to the local LLM.

**Request**
```json
{ "prompt": "Explain gradient descent in one sentence." }
```

**Response**
```json
{
  "reply":  "Gradient descent iteratively adjusts model parameters...",
  "model":  "qwen2.5-1.5b",
  "metrics": {
    "prompt_tokens": 12,
    "output_tokens": 38,
    "duration_sec":  1.42
  },
  "source": "Go Gateway -> Ollama (Qwen)"
}
```

### `GET /healthz`

Returns `200 OK` when the service is ready (used by Kubernetes probes).

### `GET /metrics`

Prometheus metrics endpoint (scraped every 15 s in K8s via ServiceMonitor).
