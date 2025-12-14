package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
)

type requestBody struct {
	Query       string   `json:"query"`
	System      string   `json:"system"`
	Temperature *float32 `json:"temperature"`
	MaxTokens   *int     `json:"max_tokens"`
	Stream      *bool    `json:"stream"`
}

func newArkChatModel(ctx context.Context) (interface {
	Generate(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error)
	Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error)
}, error) {
	timeout := 60 * time.Second

	apiKey := os.Getenv("ARK_API_KEY")
	modelID := os.Getenv("ARK_MODEL_ID")
	if apiKey == "" {
		panic("请设置ARK_API_KEY和ARK_MODEL_ID环境变量")
	}
	if modelID == "" {
		modelID = "doubao-seed-1-6-251015"
	}
	accessKey := os.Getenv("ARK_ACCESS_KEY")
	secretKey := os.Getenv("ARK_SECRET_KEY")
	baseURL := os.Getenv("ARK_BASE_URL")
	if baseURL == "" {
		baseURL = "https://ark.cn-beijing.volces.com/api/v3"
	}
	region := os.Getenv("ARK_REGION")
	if region == "" {
		region = "cn-beijing"
	}

	cfg := &ark.ChatModelConfig{
		Timeout:   &timeout,
		BaseURL:   baseURL,
		Region:    region,
		APIKey:    apiKey,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Model:     modelID,
	}

	return ark.NewChatModel(ctx, cfg)
}
func streamHandler(c *gin.Context) {
	var req requestBody
	if err := c.ShouldBindJSON(&req); err != nil || req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	ctx := c.Request.Context()
	cm, err := newArkChatModel(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "init model failed"})
		return
	}

	msgs := make([]*schema.Message, 0, 2)
	if req.System != "" {
		msgs = append(msgs, schema.SystemMessage(req.System))
	}
	msgs = append(msgs, schema.UserMessage(req.Query))

	// 只处理流式请求
	sr, err := cm.Stream(ctx, msgs)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "stream failed"})
		return
	}
	defer sr.Close()

	// ----- SSE 头部 -----
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	w := c.Writer
	flush := func() {
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

	// ----- 逐块转发 -----
	for {
		chunk, recvErr := sr.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			// 向客户端发送一个错误事件后断开
			c.SSEvent("error", recvErr.Error())
			flush()
			return
		}
		c.SSEvent("data", chunk.Content)
		c.Writer.Flush()
	}

	c.SSEvent("data", "[DONE]")
	c.Writer.Flush()
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.POST("/chat", streamHandler)

	addr := ":2345"
	log.Printf("listening %s http://localhost%v\n", addr, addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}