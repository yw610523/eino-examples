package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file, err: %v", err)
		return
	}
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
	chatModel := createArtCharModel(ctx)
	generate(ctx, chatModel, messages)
}

func generate(ctx context.Context, chatModel *ark.ChatModel, messages []*schema.Message) {
	response, err := chatModel.Generate(ctx, messages)
	if err != nil {
		panic(err)
	}
	fmt.Println(response)
}

func createArtCharModel(ctx context.Context) *ark.ChatModel {
	arkApiKey := os.Getenv("ARK_API_KEY")
	arkModelID := os.Getenv("ARK_MODEL_ID")
	if arkApiKey == "" || arkModelID == "" {
		log.Fatal("请补充火山引擎的系统环境变量: ARK_API_KEY或ARK_MODEL_ID")
	}
	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey: arkApiKey,
		Model:  arkModelID,
	})
	if err != nil {
		log.Fatalf("创建火山大模型失败: %v", err)
	}
	return chatModel
}
