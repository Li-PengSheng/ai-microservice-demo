# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code (including generated protobuf stubs)
COPY . .

# Compile to a single binary named 'gateway'
RUN go build -o gateway main.go

# Run stage
FROM alpine:latest

WORKDIR /root/
# Copy the binary from the build stage
COPY --from=builder /app/gateway .

# Expose the Gin HTTP port
EXPOSE 8080

CMD ["./gateway"]
