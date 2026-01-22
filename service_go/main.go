package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	pb "my-go-gateway/gen"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.Dial(os.Getenv("AI_SERVICE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("无法连接 AI 服务: %v", err)
	}
	defer conn.Close()

	client := pb.NewIrisPredictorClient(conn)

	r := gin.Default()
	r.POST("/predict", func(c *gin.Context) {
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
	})

	slog.Info("Go Gateway running on http://localhost:8080")
	r.Run(":8080")
}
