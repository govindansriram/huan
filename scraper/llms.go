package scraper

import (
	"context"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"huan/llm/messages"
	"huan/llm/model"
	"log"
	"math"
	"strings"
	"time"
)

type bot interface {
	Chat(convo messages.Conversation, ctx context.Context) (error, *bool, *messages.ChatCompletion)
	Validate(convo *messages.ConversationBuilder) error
}

func loadChatgptFromYML(
	modelSettings map[string]interface{},
	maxTokens uint16) (error, *model.ChatGpt) {

	cGpt := &struct {
		ApiKey      string   `yaml:"apiKey"`
		Model       string   `yaml:"model"`
		Temperature *float32 `yaml:"temperature"`
	}{}

	additionalSettings, err := yaml.Marshal(modelSettings)

	if err != nil {
		return err, nil
	}

	err = yaml.Unmarshal(additionalSettings, cGpt)

	if err != nil {
		return err, nil
	}

	maxTok := int(maxTokens)
	c := model.ChatGpt{
		Key:         cGpt.ApiKey,
		Model:       cGpt.Model,
		Temperature: cGpt.Temperature,
		MaxTokens:   &maxTok,
	}

	return nil, &c
}

func exponentialBackoff(
	parentCtx context.Context,
	model bot,
	maxWaitTime uint16,
	tryLimit uint8,
	conversation messages.Conversation,
	verbose bool) (error, *messages.AssistantMessage) {

	type retStruct struct {
		message    *messages.AssistantMessage
		error      error
		isWaitTime bool
	}

	logger := func(ms string) {
		if verbose {
			log.Println(ms)
		}
	}

	req := func(mod bot, ctx context.Context, c chan<- *retStruct) {

		err, bo, comp := mod.Chat(conversation, ctx)

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
		logger(fmt.Sprintf("executing chat request, on attempt %d", i))
		ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*time.Duration(maxWaitTime))
		channel := make(chan *retStruct)
		go req(model, ctx, channel)

		select {
		case val := <-channel:
			cancelFunc()
			if val.error != nil && !val.isWaitTime {
				logger(fmt.Sprintf("request has errored, cancelling request: %v \n", val.error))
				return val.error, nil
			} else if val.error != nil && val.isWaitTime {
				logger("rate limit hit, sleeping...")
				snooze(int(i), int(tryLimit))
			} else if val.error == nil {
				logger("response received")
				return nil, val.message
			}
		case <-ctx.Done():
			cancelFunc()
			logger("max request duration hit, sleeping...")
			snooze(int(i), int(tryLimit))
		case <-parentCtx.Done():
			cancelFunc()
			logger("global timeout hit cancelling")
			return parentCtx.Err(), nil
		}

		cancelFunc()
	}

	logger("try limit reached request has failed")
	return errors.New("reached try limit for chat request"), nil
}

/*
LanguageModel

a wrapper struct around a llm with easy chat requests and defaults
*/
type LanguageModel struct {
	tryLimit uint8
	duration uint16
	verbose  bool
	bot      bot
}

func (l *LanguageModel) Chat(ctx context.Context, convo *messages.Conversation) (error, *messages.AssistantMessage) {
	return exponentialBackoff(ctx, l.bot, l.duration, l.tryLimit, *convo, l.verbose)
}

func (l *LanguageModel) Validate(convo *messages.ConversationBuilder) error {
	err := l.bot.Validate(convo)
	return err
}

func InitLanguageModel(
	modelType string,
	settings map[string]interface{},
	tryLimit *uint8,
	maxTokens *uint16,
	duration *uint16,
	verbose bool) (error, *LanguageModel) {

	var tokenLimit uint16

	lang := &LanguageModel{
		verbose: verbose,
	}

	if duration == nil {
		lang.duration = 100
	} else {
		lang.duration = *duration
	}

	if tryLimit == nil {
		lang.tryLimit = 4
	} else {
		lang.tryLimit = *tryLimit
	}

	if maxTokens == nil {
		tokenLimit = 500
	} else {
		tokenLimit = *maxTokens
	}

	var b bot

	switch strings.ToLower(modelType) {
	case "openai":
		err, mod := loadChatgptFromYML(settings, tokenLimit)

		if err != nil {
			return err, nil
		}

		b = mod

	default:
		return fmt.Errorf("there is no llm type %s", modelType), nil
	}

	lang.bot = b
	return nil, lang
}
