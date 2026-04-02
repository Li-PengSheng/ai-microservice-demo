# Lightweight Python base image
FROM python:3.12-slim

# Install uv package manager
COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /bin/

# Set working directory
WORKDIR /app

# Copy dependency files and install
COPY pyproject.toml uv.lock ./
RUN uv sync --frozen

# Copy generated protobuf stubs and application source
COPY gen/ ./gen/ 
COPY models/ ./models/
COPY main.py observability.py server.py gpu_exporter.py ./

# Ensure the gen directory is on PYTHONPATH
ENV PYTHONPATH="/app:/app/gen"

# Expose the gRPC port
EXPOSE 50051

# Start the gRPC server
CMD ["uv", "run", "main.py"]
