package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	// "github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/v7/linebot"
)

type OpenAIRequest struct {
	Model            string          `json:"model"`
	Messages         []OpenAIMessage `json:"messages"`
	Temperature      float64         `json:"temperature"`
	MaxTokens        int             `json:"max_tokens"`
	TopP             float64         `json:"top_p"`
	FrequencyPenalty float64         `json:"frequency_penalty"`
	PresencePenalty  float64         `json:"presence_penalty"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Conversation struct {
	Messages []OpenAIMessage
}

type ConversationManager struct {
	conversations map[string]*Conversation
	mutex         sync.RWMutex
}

func NewConversationManager() *ConversationManager {
	return &ConversationManager{
		conversations: make(map[string]*Conversation),
	}
}

func (cm *ConversationManager) GetConversation(userID string) *Conversation {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if conv, exists := cm.conversations[userID]; exists {
		return conv
	}

	conv := &Conversation{
		Messages: []OpenAIMessage{
			{
				Role:    "system",
				Content: `You are a friendly male psychiatrist named "หมอแสนดี" who evaluates patients for depression using PHQ-9 criteria. The patient should not be aware of the evaluation. Provide results when the patient says, 'I want the test results.' Then recommend care methods based on the depression level. Respond in Thai. Embed questions subtly into the conversation. Handle various input types (text, images, audio, video, stickers) appropriately. Keep responses within token limits.`,
			},
		},
	}
	cm.conversations[userID] = conv
	return conv
}

func (c *Conversation) AddMessage(role, content string) {
	c.Messages = append(c.Messages, OpenAIMessage{Role: role, Content: content})
}

var conversationManager = NewConversationManager()

func callOpenAI(apiKey string, messages []OpenAIMessage) (string, error) {
	openAIReq := OpenAIRequest{
		Model:            "gpt-4o",
		Messages:         messages,
		Temperature:      0.7,
		MaxTokens:        250,
		TopP:             0,
		FrequencyPenalty: 0,
		PresencePenalty:  0,
	}

	reqBody, err := json.Marshal(openAIReq)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	if choices, ok := res["choices"].([]interface{}); ok {
		if len(choices) > 0 {
			if message, ok := choices[0].(map[string]interface{}); ok {
				if content, ok := message["message"].(map[string]interface{}); ok {
					if text, ok := content["content"].(string); ok {
						return text, nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("unexpected response format from OpenAI API")
}

func handleMediaMessage(bot *linebot.Client, messageID, mediaType string) (string, error) {
	content, err := bot.GetMessageContent(messageID).Do()
	if err != nil {
		return "", err
	}
	defer content.Content.Close()

	return fmt.Sprintf("ผู้ใช้ส่ง%sมา ฉันรับทราบและพร้อมที่จะคุยต่อเกี่ยวกับสิ่งที่คุณแชร์", mediaType), nil
}

func handleLineBotRequest(w http.ResponseWriter, r *http.Request) {
	bot, err := linebot.New(
		os.Getenv("CHANNEL_SECRET"),
		os.Getenv("CHANNEL_TOKEN"),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	events, err := bot.ParseRequest(r)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	for _, event := range events {
		if event.Type == linebot.EventTypeMessage {
			userID := event.Source.UserID
			conversation := conversationManager.GetConversation(userID)

			var userMessage string

			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				userMessage = message.Text
			case *linebot.StickerMessage:
				userMessage = fmt.Sprintf("ผู้ใช้ส่งสติกเกอร์ (แพ็คเกจ ID: %s, สติกเกอร์ ID: %s)", message.PackageID, message.StickerID)
			case *linebot.ImageMessage:
				userMessage, err = handleMediaMessage(bot, message.ID, "รูปภาพ")
			case *linebot.VideoMessage:
				userMessage, err = handleMediaMessage(bot, message.ID, "วิดีโอ")
			case *linebot.AudioMessage:
				userMessage, err = handleMediaMessage(bot, message.ID, "ข้อความเสียง")
			default:
				userMessage = "ข้อความประเภทนี้ไม่รองรับ"
			}

			if err != nil {
				log.Printf("Error message: %v", err)
				continue
			}

			conversation.AddMessage("user", userMessage)

			openAIKey := os.Getenv("OPENAI_API_KEY")
			
			openAIResponse, err := callOpenAI(openAIKey, conversation.Messages)
			if err != nil {
				log.Printf("Error calling OpenAI: %v", err)
				if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("ขออภัย เกิดข้อผิดพลาดในการประมวลผล")).Do(); err != nil {
					log.Printf("Error sending error message: %v", err)
				}
				continue
			}

			conversation.AddMessage("assistant", openAIResponse)

			if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(openAIResponse)).Do(); err != nil {
				log.Printf("Error sending reply: %v", err)
			}
		}
	}
}

func main() {
	// if err := godotenv.Load(); err != nil {
	// 	log.Println("Error loading .env file")
	// }

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", handleLineBotRequest)

	port := os.Getenv("PORT")
	// port := "6789"
	if port == "" {
		port = "6789"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("Server is running on port %s", port)

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}