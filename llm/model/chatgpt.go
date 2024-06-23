package model

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"huan/helper"
	"huan/llm/messages"
	"io"
	"net/http"
	"strings"
)

const (
	KILOBYTE = 1024
	MEGABYTE = KILOBYTE * KILOBYTE
)

/*
Engine

a struct representing the capabilities of a llm
*/
type Engine struct {
	ContextWindow   uint32
	HasJsonMode     bool
	Name            string
	Multimodal      bool
	FunctionCalling bool
}

/*
GetEngineMap

returns a map of all available chatgpt engines
*/
func GetEngineMap() map[string]Engine {
	gpt3turbo := Engine{
		16385,
		true,
		"gpt-3.5-turbo",
		false,
		true,
	}

	gpt4O := Engine{
		128000,
		true,
		"gpt-4o",
		true,
		true,
	}

	gpt4Turbo := Engine{
		ContextWindow:   128000,
		HasJsonMode:     true,
		Name:            "gpt-4-turbo",
		Multimodal:      true,
		FunctionCalling: true,
	}

	return map[string]Engine{
		gpt4Turbo.Name: gpt4Turbo,
		gpt3turbo.Name: gpt3turbo,
		gpt4O.Name:     gpt4O,
	}
}

/*
getEngineOptionList

A list of all the engines as a string
*/
func getEngineOptionList() string {

	engineString := ""
	for k := range GetEngineMap() {
		engineString += k + ", "
	}

	return engineString[:len(engineString)-2]
}

/*
ChatGpt

holds all the data to make requests with ChatGpt
*/
type ChatGpt struct {
	Model            string
	FrequencyPenalty *float32
	LogitBias        *map[string]int
	LogProbs         *bool
	TopLogprobs      *uint8
	MaxTokens        *int
	N                *int
	PresencePenalty  *float32
	ResponseFormat   *map[string]string
	Seed             *int
	Stop             *interface{}
	Stream           *bool
	Temperature      *float32
	TopP             *float32
	Tools            *[]messages.Tool
	ToolChoice       interface{}
	Key              string
}

/*
gptError

an error message provided by the chatbot
*/
type gptError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}

/*
gptRequestError

the error message provided with a corresponding status code, the status code makes it
easier to do rate limiting
*/
type gptRequestError struct {
	StatusCode int      `json:"statusCode"`
	Error      gptError `json:"error"`
}

/*
validateResponseFormat

validates that the response format you requested is valid, such as json_object
*/
func validateResponseFormat(responseFormat map[string]string, engine Engine) error {

	validSlice := []string{
		"json_object", "text",
	}

	containsString := helper.Contains[string]

	if len(responseFormat) > 1 {
		return fmt.Errorf("response format must only have one key, detected %d keys", len(responseFormat))
	}

	var v string
	var k string

	for k, v = range responseFormat {
		if k != "type" {
			return fmt.Errorf("response format must only have one key called type, detected key named %s", k)
		}

		if !containsString(validSlice, v) {
			return fmt.Errorf("%s is not a valid type", v)
		}
	}

	if v == "json_object" {
		if !engine.HasJsonMode {
			return errors.New("engine is not json mode capable")
		}
	}

	return nil
}

/*
validateToolChoice

validates the tools you provided can be used by the model
*/
func validateToolChoice(toolChoice interface{}) error {

	if val, ok := toolChoice.(string); ok {
		if !(val == "auto" || val == "none" || val == "required") {
			return fmt.Errorf("tool choice must be auto, none, or required found %s", val)
		} else {
			return nil
		}
	}

	val, ok := toolChoice.(map[string]interface{})

	if !ok {
		return errors.New("tool choice must be either a string or map")
	}

	valType, ok := val["type"]

	if !ok {
		return errors.New("tool choice map missing type key")
	}

	typeString, ok := valType.(string)

	if !ok {
		return errors.New("type must be a string")
	}

	optSlice := []string{
		"function",
	}

	if !helper.Contains[string](optSlice, typeString) {
		return fmt.Errorf("tool choice type must be function found %s", typeString)
	}

	functionInterface, ok := val[typeString]

	if !ok {
		return errors.New("tool choice is missing function definition object")
	}

	functionDefintion, ok := functionInterface.(map[string]string)

	if !ok {
		return errors.New("definition must be of type object")
	}

	if _, ok = functionDefintion["name"]; !ok {
		return errors.New("missing function name")
	}

	return nil
}

/*
checkB64

check if a string is a valid base64 with under 20mb of data
*/
func checkB64(str string) error {
	sDec, err := base64.StdEncoding.DecodeString(str)

	if err != nil {
		return err
	}

	if len(sDec) > 20*MEGABYTE {
		return errors.New("image exceeds 20MB")
	}

	return nil
}

/*
handleContent

checks if multimodal content is valid and if so alters it to work with a chatgpt request
*/
func handleContent(content *messages.MultimodalContent) error {

	//TODO: Please Refactor

	validExt := []string{
		"png", "jpeg", "jpg", "webp", "gif",
	}

	containsString := helper.Contains[string]

	if content.Type == "image_url" {
		if content.ImageUrl.Type == "" {
			if !strings.HasPrefix(content.ImageUrl.Url, "http") {
				return errors.New("image url is invalid")
			}
		} else if !containsString(validExt, content.ImageUrl.Type) {
			return fmt.Errorf("images of type %s are not accepted in chatgpt multimodal requests", content.Type)
		} else if !strings.HasPrefix(content.ImageUrl.Url, "data:image/") {
			if err := checkB64(content.ImageUrl.Url); err != nil {
				return err
			}

			content.ImageUrl.Url = fmt.Sprintf(
				`data:image/%s;base64,%s`,
				content.ImageUrl.Type,
				content.ImageUrl.Url)

		}
	}

	return nil
}

/*
adjustConversation

check the content for all multimodal messages and ensure they are compliant with gpt requirements
*/
func adjustConversation(conversation *messages.ConversationBuilder) error {
	for i := range conversation.Size() {
		if conversation.GetMessageType(i) == "multimodal" {
			mess := conversation.ConvertToMultiModal(i)
			for _, content := range mess.Content {
				if err := handleContent(&content); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

/*
checkIsEngineCapable

check if an engine is chatgpt engine is capable of fulfilling multimodal requests
*/
func checkIsEngineCapable(engine Engine, convo *messages.ConversationBuilder) error {
	for i := range convo.Size() {
		if convo.GetMessageType(i) == "multimodal" && !engine.Multimodal {
			return fmt.Errorf("engine: %s, lacks multimodal capabilities", engine.Name)
		}
	}

	return nil
}

func validateTools(engine Engine, tool messages.Tool) error {
	validSlice := []string{
		"function",
	}

	containsString := helper.Contains[string]

	if !containsString(validSlice, tool.Type) {
		return fmt.Errorf("tool can be only type function, found %s", tool.Type)
	}

	if tool.Type == "function" && !engine.FunctionCalling {
		return errors.New("engine is not function call capable")
	}

	return nil
}

/*
Validate

validates the gpt parameters are correct
*/
func (c *ChatGpt) Validate(convo *messages.ConversationBuilder) error {
	floatHelper := helper.IsBetween[float32]
	intHelper := helper.IsBetween[uint8]

	if c.Key == "" {
		return errors.New("openai settings received empty api key")
	}

	if c.Model == "" {
		return errors.New("openai settings received empty model name")
	}

	if c.MaxTokens != nil && *c.MaxTokens < 1 {
		return errors.New("max tokens must be greater than 1")
	}

	if err := adjustConversation(convo); err != nil {
		return err
	}

	if val, ok := GetEngineMap()[c.Model]; !ok {
		return fmt.Errorf("gpt model %s is not permitted", val.Name)
	}

	if c.FrequencyPenalty != nil && !floatHelper(-2.0, 2.0, *c.FrequencyPenalty, true, true) {
		return fmt.Errorf("frequency penalty must be between -2.0 and 2.0 got %f", *c.FrequencyPenalty)
	}

	if c.TopLogprobs != nil && !intHelper(0, 20, *c.TopLogprobs, true, true) {
		return fmt.Errorf("top log probs must be between 0 and 20 got %d", *c.TopLogprobs)
	}

	if c.PresencePenalty != nil && !floatHelper(-2.0, 2.0, *c.PresencePenalty, true, true) {
		return fmt.Errorf("presence penalty must be between -2.0 and 2.0 got %f", *c.PresencePenalty)
	}

	if c.Temperature != nil && !floatHelper(0.0, 2.0, *c.Temperature, true, true) {
		return fmt.Errorf("temperature must be between 0.0 and 2.0 got %f", *c.Temperature)
	}

	if c.TopP != nil && !floatHelper(0.0, 1.0, *c.TopP, true, true) {
		return fmt.Errorf("top p must be between 0.0 and 1.0 got %f", *c.TopP)
	}

	var engine Engine
	var ok bool

	if engine, ok = GetEngineMap()[c.Model]; !ok {
		return fmt.Errorf(
			"gpt has no integrated engine named %s, available options are %s",
			c.Model,
			getEngineOptionList())
	}

	if c.ResponseFormat != nil {
		err := validateResponseFormat(*c.ResponseFormat, engine)

		if err != nil {
			return err
		}
	}

	if c.Tools != nil {
		for _, i := range *c.Tools {
			if err := validateTools(engine, i); err != nil {
				return err
			}
		}
	}

	if err := checkIsEngineCapable(engine, convo); err != nil {
		return err
	}

	if c.ToolChoice != nil {
		if err := validateToolChoice(c.ToolChoice); err != nil {
			return err
		}
	}

	return nil
}

// TODO
// Add method to estimate context window For multimodal and regular requests

/*
Chat

makes a chat completion request to chatgpt, the settings and messages supplied will be used as request parameters

returns:
- an error representing if the request failed at any point
- a boolean pointer, it is nil if the request succeeded. it points to true if the request failed due to rate limiting
- a chat completion pointer: contains the request response
*/

func (c *ChatGpt) Chat(
	convo messages.Conversation,
	ctx context.Context) (error, *bool, *messages.ChatCompletion) {

	var isRateLimit bool

	type chatGpt struct {
		Model            string                `json:"model"`
		FrequencyPenalty *float32              `json:"frequency_penalty,omitempty"`
		Messages         messages.Conversation `json:"messages"`
		LogitBias        *map[string]int       `json:"logit_bias,omitempty"`
		LogProbs         *bool                 `json:"log_probs,omitempty"`
		TopLogprobs      *uint8                `json:"top_logprobs,omitempty"`
		MaxTokens        *int                  `json:"max_tokens,omitempty"`
		N                *int                  `json:"n,omitempty"`
		PresencePenalty  *float32              `json:"presence_penalty,omitempty"`
		ResponseFormat   *map[string]string    `json:"response_format,omitempty"`
		Seed             *int                  `json:"seed,omitempty"`
		Stop             *interface{}          `json:"stop,omitempty"`
		Stream           *bool                 `json:"stream,omitempty"`
		Temperature      *float32              `json:"temperature,omitempty"`
		TopP             *float32              `json:"top_p,omitempty"`
		Tools            *[]messages.Tool      `json:"tools,omitempty"`
		ToolChoice       interface{}           `json:"tool_choice"`
		key              string
	}

	n := 1

	chatSettings := chatGpt{
		Model:            c.Model,
		FrequencyPenalty: c.FrequencyPenalty,
		Messages:         convo,
		LogitBias:        c.LogitBias,
		LogProbs:         c.LogProbs,
		TopLogprobs:      c.TopLogprobs,
		MaxTokens:        c.MaxTokens,
		N:                &n,
		PresencePenalty:  c.PresencePenalty,
		ResponseFormat:   c.ResponseFormat,
		Seed:             c.Seed,
		Stop:             c.Stop,
		Stream:           c.Stream,
		Temperature:      c.Temperature,
		TopP:             c.TopP,
		Tools:            c.Tools,
		ToolChoice:       c.ToolChoice,
	}

	url := "https://api.openai.com/v1/chat/completions"
	jsonBytes, err := json.Marshal(chatSettings)

	if err != nil {
		return err, &isRateLimit, nil
	}

	var client http.Client
	reader := bytes.NewReader(jsonBytes)
	pRequest, err := http.NewRequestWithContext(ctx, "POST", url, reader)

	if err != nil {
		return err, &isRateLimit, nil
	}

	pRequest.Header.Set("Content-Type", "application/json")
	pRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Key))

	pResponse, err := client.Do(pRequest)

	if err != nil {
		return err, &isRateLimit, nil
	}

	defer func() {
		closeErr := pResponse.Body.Close()
		if closeErr != nil {
			panic(closeErr)
		}
	}()

	responseBytes, err := io.ReadAll(pResponse.Body)

	if err != nil {
		return err, &isRateLimit, nil
	}

	if pResponse.StatusCode == 200 {
		var gptResp messages.ChatCompletion
		if err = json.Unmarshal(responseBytes, &gptResp); err != nil {
			return err, &isRateLimit, nil
		}

		return nil, nil, &gptResp
	} else {
		var resp gptRequestError
		if err = json.Unmarshal(responseBytes, &resp); err != nil {
			return err, &isRateLimit, nil
		}

		err := errors.New(resp.Error.Message)

		if pResponse.StatusCode == 429 {
			isRateLimit = true
			return err, &isRateLimit, nil
		}

		return err, &isRateLimit, nil
	}
}
