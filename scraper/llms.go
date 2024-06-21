package scraper

import (
	"agent/llm/messages"
	"agent/llm/model"
	"context"
	"errors"
	"gopkg.in/yaml.v3"
	"log"
	"math"
	"time"
)

type Chatgpt struct {
	chat *model.ChatGpt
}

func (c *Chatgpt) AppendAssistant(message *messages.AssistantMessage) error {
	return c.chat.AppendAssistantMessage(message)
}

func (c *Chatgpt) AppendStandard(message *messages.StandardMessage) error {
	return c.chat.AppendStandardMessage(message)
}

func (c *Chatgpt) AppendMultiModal(imageBytes []byte, role string, detail *string, imageType string) error {
	mess := c.chat.GetMultiModalMessage(role)
	mess.AddImageB64(imageBytes, detail, imageType)
	return mess.Build()
}

func (c *Chatgpt) Pop(index uint) {
	c.chat.PopMessage(index)
}

func (c *Chatgpt) Chat(ctx context.Context) (error, *bool, *messages.ChatCompletion) {
	return c.chat.Chat(ctx)
}

func (c *Chatgpt) deepCopy() llm {
	//c.chat = c.chat.DeepCopy()
	return &Chatgpt{
		chat: c.chat.DeepCopy(),
	}
}

func loadCGptFromYaml(data []byte, maxTokens uint16) (error, *Chatgpt) {
	cGpt := &struct {
		ApiKey      string   `yaml:"apiKey"`
		Model       string   `yaml:"model"`
		Temperature *float32 `yaml:"temperature"`
	}{}

	err := yaml.Unmarshal(data, cGpt)

	if err != nil {
		return err, nil
	}

	err, chat := model.InitChatGpt(cGpt.ApiKey, cGpt.Model)

	if err != nil {
		return err, nil
	}

	mt := int(maxTokens)

	chat.MaxTokens = &(mt)
	chat.Temperature = cGpt.Temperature

	return nil, &Chatgpt{
		chat: chat,
	}
}

type llm interface {
	AppendAssistant(message *messages.AssistantMessage) error
	AppendStandard(message *messages.StandardMessage) error
	AppendMultiModal(imageBytes []byte, role string, detail *string, imageType string) error
	Pop(index uint)
	Chat(ctx context.Context) (error, *bool, *messages.ChatCompletion)
	deepCopy() llm
}

func exponentialBackoff(
	parentCtx context.Context,
	model llm,
	maxWaitTime uint16,
	tryLimit uint8) (error, *messages.AssistantMessage) {

	type retStruct struct {
		message    *messages.AssistantMessage
		error      error
		isWaitTime bool
	}

	req := func(mod llm, ctx context.Context, c chan<- *retStruct) {

		err, bo, comp := mod.Chat(ctx)

		var mess *messages.AssistantMessage
		if comp != nil {
			mess = &comp.ToAssistant()[0]
		}

		r := &retStruct{
			message: mess,
			error:   err,
			isWaitTime: func() bool {
				if bo == nil {
					return false
				}
				return *bo
			}(),
		}

		c <- r
	}

	snooze := func(index int, tryLimit int) {
		if index != tryLimit-1 {
			time.Sleep(time.Second * time.Duration(math.Pow(2.0, float64(index))))
		}
	}

	for i := range tryLimit {
		log.Printf("executing chat request, on attempt %d", i)
		ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*time.Duration(maxWaitTime))
		channel := make(chan *retStruct)
		go req(model, ctx, channel)

		select {
		case val := <-channel:
			cancelFunc()
			if val.error != nil && !val.isWaitTime {
				log.Printf("request has errored, cancelling request: %v \n", val.error)
				return val.error, nil
			} else if val.error != nil && val.isWaitTime {
				log.Println("rate limit hit, sleeping...")
				snooze(int(i), int(tryLimit))
			} else if val.error == nil {
				log.Println("response received")
				return nil, val.message
			}
		case <-ctx.Done():
			cancelFunc()
			log.Println("max request duration hit, sleeping...")
			snooze(int(i), int(tryLimit))
		case <-parentCtx.Done():
			cancelFunc()
			log.Println("global timeout hit cancelling")
			return parentCtx.Err(), nil
		}

		cancelFunc()
	}

	log.Println("try limit reached request has failed")
	return errors.New("reached try limit for chat request"), nil
}
