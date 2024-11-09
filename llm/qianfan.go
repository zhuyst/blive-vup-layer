package llm

import (
	"blive-vup-layer/config"
	"context"
	"errors"
	"fmt"
	"github.com/baidubce/bce-qianfan-sdk/go/qianfan"
	log "github.com/sirupsen/logrus"
	"strings"
)

type LLM struct {
	cfg            *config.QianFanConfig
	chatCompletion *qianfan.ChatCompletion
}

func NewLLM(config *config.QianFanConfig) *LLM {
	cfg := qianfan.GetConfig()
	cfg.AK = config.AccessKey
	cfg.SK = config.SecretKey
	return &LLM{
		cfg: config,
		chatCompletion: qianfan.NewChatCompletion(
			qianfan.WithModel("ERNIE-4.0-Turbo-8K"),
		),
	}
}

type ChatMessage struct {
	User    string
	Message string
}

func (msg *ChatMessage) String() string {
	return fmt.Sprintf("用户【%s】说：%s", msg.User, msg.Message)
}

func (llm *LLM) ChatWithLLM(ctx context.Context, messages []*ChatMessage) (string, error) {
	if len(messages) == 0 {
		return "", errors.New("no messages")
	}
	contentSb := strings.Builder{}
	if len(messages) > 1 {
		contentSb.WriteString("以下是历史用户发言：\n")
		for _, msg := range messages[:len(messages)-1] {
			contentSb.WriteString(msg.String() + "\n")
		}
	}
	currentMsg := messages[len(messages)-1]
	contentSb.WriteString("以下是当前用户发言：\n")
	contentSb.WriteString(currentMsg.String())

	content := contentSb.String()
	log.Infof("LLM content: %s", content)

	resp, err := llm.chatCompletion.Do(
		ctx,
		&qianfan.ChatCompletionRequest{
			System:      llm.cfg.Prompt,
			Temperature: 0.5,
			TopP:        0.5,
			Messages: []qianfan.ChatCompletionMessage{
				qianfan.ChatCompletionUserMessage(content),
			},
		},
	)
	if err != nil {
		log.Errorf("LLM err: %v", err)
		return "", err
	}

	result := resp.Result
	result = strings.ReplaceAll(result, "喔~", "喵 ")
	result = strings.ReplaceAll(result, "~", " ")
	result = strings.TrimSpace(result)
	log.Infof("LLM result: %s", result)
	return result, nil
}
