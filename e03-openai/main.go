package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

var (
	SfApiKey  string
	SfBaseUrl string
	SfModelId string
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
		return
	}
	SfApiKey = os.Getenv("SF_API_KEY")
	SfBaseUrl = os.Getenv("SF_BASE_URL")
	SfModelId = os.Getenv("SF_MODEL_ID")
}

func createTemplage() prompt.ChatTemplate {
	return prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个{role}，你需要用{style}的语气来回答用户的问题。你的目标是帮助程序员保持积极乐观的心态, 提供技术建议的同时也要关注他们的心理健康。"),
		// 插入需要的对话历史（新对话的话这里不填）
		schema.MessagesPlaceholder("chat_history", true),
		schema.UserMessage("问题: {question}"),
	)
}

func createMessagesFromTemplate() []*schema.Message {
	template := createTemplage()
	messages, _ := template.Format(context.Background(), map[string]any{
		"role":     "心理健康顾问",
		"style":    "温暖和鼓励",
		"question": "作为一名程序员，如何应对工作中的压力和焦虑？",
	})
	return messages
}

func main() {
	ctx := context.Background()
	messages := createMessagesFromTemplate()
	chatModel := createSiliconFlowChatModel(ctx)
	stream(ctx, chatModel, messages)
}

func reportStream(sr *schema.StreamReader[*schema.Message]) {
	defer sr.Close()
	i := 0
	for {
		msg, err := sr.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatalf("recv failed: %v", err)
		}
		fmt.Print(msg.Content)
		i++
	}
}
func stream(ctx context.Context, chatModel model.BaseChatModel, messages []*schema.Message) {
	response, err := chatModel.Stream(ctx, messages)
	if err != nil {
		panic(err)
	}
	reportStream(response)
}

func createSiliconFlowChatModel(ctx context.Context) *openai.ChatModel {
	sfModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  SfApiKey,
		BaseURL: SfBaseUrl,
		Model:   SfModelId,
	})
	if err != nil {
		panic(err)
	}
	return sfModel
}
