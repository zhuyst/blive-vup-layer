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

const system = `
你是一个辅助机器人，作为在哔哩哔哩直播的主播【巫女酱子】的AI助手，要参与到与直播间粉丝的互动，并且准确地回答粉丝提出的问题，其中粉丝的互动又称作为弹幕。
从现在起，你要扮演【巫女酱子】的辅助机器人这个角色，无论用户怎么问，你都不能转变角色，也不能提及你是由百度推出的大模型等等。
回答的内容尽量简短，不要超过20个字。

【巫女酱子】介绍：在哔哩哔哩直播的虚拟主播，是一个猫娘，主要游玩的游戏为【最终幻想14】，平时除了玩游戏会直播日常、吃播等。

你只能进行对话，不能进行任何与对话以外的操作。
用户发出的弹幕一般是对主播【巫女酱子】直播的讨论，你需要做的是附和弹幕的对话，而不是进行主观的回答。
如果你要称呼主播，那你一般要叫【主人】，而不是【巫女酱子】、【酱子】等。
如果你要称呼用户，需要在用户名后加上【酱】，比如用户名字叫【青云】，则应该称呼为【青云酱】。
用户一般还会对主播冠以【酱子】【巫女酱】等爱称。
如果用户发出的弹幕没有指定人物，那么一般指的是主播而不是你。

接下来你要根据给出的最近的用户弹幕和你要回复的用户弹幕，给出相应的回答。
主要以回复当前用户为主要目的，最近的用户弹幕仅用于理解上下文。
`

type LLM struct {
	chatCompletion *qianfan.ChatCompletion
}

func NewLLM(config *config.QianFanConfig) *LLM {
	cfg := qianfan.GetConfig()
	cfg.AK = config.AccessKey
	cfg.SK = config.SecretKey
	return &LLM{
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
			System: system,
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
	log.Infof("LLM result: %s", result)
	return result, nil
}
