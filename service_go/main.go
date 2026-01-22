package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	pb "my-go-gateway/gen"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

func main() {
	conn, err := grpc.Dial(os.Getenv("AI_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("无法连接 AI 服务: %v", err)
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

		// 4. 返回最终结果
		c.JSON(http.StatusOK, gin.H{
			"result": resp.ClassName,
			"id":     resp.ClassId,
			"source": "Go Gateway -> Python AI",
		})

		httpRequestsTotal.WithLabelValues("/predict", "200").Inc()
	})

	slog.Info("Go Gateway running on http://localhost:8080")
	r.Run(":8080")
}
