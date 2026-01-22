package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	// --- 链路追踪相关核心依赖 ---

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	// 拦截器与插件
	pb "my-go-gateway/gen" // 确保路径与你的 go.mod 一致

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

var (
	// 记录请求总数 (用于计算 QPS)
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"path", "status"},
	)
	// 记录请求耗时 (用于计算 P99 延迟)
	httpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of response latency for HTTP requests.",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"path"},
	)
)

func init() {
	// 注册指标
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpDuration)
}

// 初始化 OpenTelemetry 追踪器
func initTracer() (*sdktrace.TracerProvider, error) {
	ctx := context.Background()
	// 连接到 Docker Compose 中的 Jaeger 服务
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("jaeger:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("go-gateway"),
		)),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

func main() {
	// 1. 初始化链路追踪
	tp, err := initTracer()
	if err != nil {
		slog.Error("failed to initialize tracer", "error", err)
		os.Exit(1)
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()

	conn, err := grpc.Dial(os.Getenv("AI_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	if err != nil {
		slog.Error("failed to connect to AI service", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewIrisPredictorClient(conn)

	r := gin.Default()

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.POST("/predict", func(c *gin.Context) {

		timer := prometheus.NewTimer(httpDuration.WithLabelValues("/predict"))
		defer timer.ObserveDuration()

		var reqJson struct {
			SepalLength float32 `json:"sepal_length"`
			SepalWidth  float32 `json:"sepal_width"`
			PetalLength float32 `json:"petal_length"`
			PetalWidth  float32 `json:"petal_width"`
		}
		// 绑定 JSON
		if err := c.ShouldBindJSON(&reqJson); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		resp, err := client.Predict(ctx, &pb.PredictRequest{
			SepalLength: reqJson.SepalLength,
			SepalWidth:  reqJson.SepalWidth,
			PetalLength: reqJson.PetalLength,
			PetalWidth:  reqJson.PetalWidth,
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 服务调用失败: " + err.Error()})
			return
		}
		httpRequestsTotal.WithLabelValues("/predict", "200").Inc()

		// 4. 返回最终结果
		c.JSON(http.StatusOK, gin.H{
			"result": resp.ClassName,
			"id":     resp.ClassId,
			"source": "Go Gateway -> Python AI",
		})

	})

	slog.Info("Go Gateway running on http://localhost:8080")
	r.Run(":8080")
}

//go get go.opentelemetry.io/otel go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc
