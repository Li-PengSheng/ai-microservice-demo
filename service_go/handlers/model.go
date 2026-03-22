// service_go/handlers/model.go
package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"

	modelv1 "my-go-gateway/gen/model/v1"
	"my-go-gateway/config"
	"my-go-gateway/metrics"
)

type ModelHandler struct {
	client modelv1.ModelPredictorClient
	cfg    *config.Config
}

func NewModelHandler(client modelv1.ModelPredictorClient, cfg *config.Config) *ModelHandler {
	return &ModelHandler{client: client, cfg: cfg}
}

func (h *ModelHandler) Predict(c *gin.Context) {
	timer := prometheus.NewTimer(metrics.HTTPDuration.WithLabelValues("/predict/model"))
	defer timer.ObserveDuration()

	var body struct {
		Prompt string `json:"prompt" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		metrics.HTTPRequestsTotal.WithLabelValues("/predict/model", "400").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(body.Prompt) > h.cfg.MaxPromptLen {
		metrics.HTTPRequestsTotal.WithLabelValues("/predict/model", "400").Inc()
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "prompt exceeds maximum length of 2000 characters",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.cfg.ModelTimeout)
	defer cancel()

	grpcStart := time.Now()
	resp, err := h.client.ModelPredict(ctx, &modelv1.ModelPredictRequest{
		Prompt: body.Prompt,
	})
	grpcStatus := "ok"
	if err != nil {
		grpcStatus = "error"
	}
	metrics.GRPCRequestDuration.WithLabelValues("ModelPredict", grpcStatus).
		Observe(time.Since(grpcStart).Seconds())

	if err != nil {
		metrics.HTTPRequestsTotal.WithLabelValues("/predict/model", "500").Inc()
		slog.Error("model service call failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if resp.EvalCount > 0 {
		metrics.AITokensTotal.WithLabelValues(resp.ModelName).Add(float64(resp.EvalCount))
		metrics.AIGenerationDuration.WithLabelValues(resp.ModelName).
			Observe(float64(resp.EvalDuration) / 1e9)
	}

	metrics.HTTPRequestsTotal.WithLabelValues("/predict/model", "200").Inc()
	c.JSON(http.StatusOK, gin.H{
		"reply": resp.Response,
		"model": resp.ModelName,
		"metrics": gin.H{
			"prompt_tokens": resp.PromptEvalCount,
			"output_tokens": resp.EvalCount,
			"duration_sec":  float64(resp.EvalDuration) / 1e9,
		},
		"source": "Go Gateway -> Ollama (Qwen)",
	})
}