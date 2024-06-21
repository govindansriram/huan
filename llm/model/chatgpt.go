package model

import (
	"agent/helper"
	"agent/llm/messages"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const (
	KILOBYTE = 1024
	MEGABYTE = KILOBYTE * KILOBYTE
)

/*
llmMessages

an interface for all messages being passed into a llm to follow
*/
type llmMessages interface {
	Validate() error
}

/*
engine

a struct representing the capabilities of a llm
*/
type engine struct {
	ContextWindow   uint32
	HasJsonMode     bool
	Name            string
	Multimodal      bool
	FunctionCalling bool
}

/*
getEngineMap

returns a map of all available chatgpt engines
*/
func getEngineMap() map[string]engine {
	gpt3turbo := engine{16385, true, "gpt-3.5-turbo", false, true}
	gpt4O := engine{128000, true, "gpt-4o", true, true}
	gpt4Turbo := engine{ContextWindow: 128000, HasJsonMode: true, Name: "gpt-4-turbo", Multimodal: true, FunctionCalling: true}

	return map[string]engine{
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
	for k := range getEngineMap() {
		engineString += k + ", "
	}

	return engineString[:len(engineString)-2]
}

/*
ChatGpt

holds all the data to make requests with ChatGpt
*/
type ChatGpt struct {
	Model            string             `json:"model"`
	FrequencyPenalty *float32           `json:"frequency_penalty,omitempty"`
	Messages         []llmMessages      `json:"messages"`
	LogitBias        *map[string]int    `json:"logit_bias,omitempty"`
	LogProbs         *bool              `json:"log_probs,omitempty"`
	TopLogprobs      *uint8             `json:"top_logprobs,omitempty"`
	MaxTokens        *int               `json:"max_tokens,omitempty"`
	N                *int               `json:"n,omitempty"`
	PresencePenalty  *float32           `json:"presence_penalty,omitempty"`
	ResponseFormat   *map[string]string `json:"response_format,omitempty"`
	Seed             *int               `json:"seed,omitempty"`
	Stop             *interface{}       `json:"stop,omitempty"`
	Stream           *bool              `json:"stream,omitempty"`
	Temperature      *float32           `json:"temperature,omitempty"`
	TopP             *float32           `json:"top_p,omitempty"`
	Tools            *[]messages.Tool   `json:"tools,omitempty"`
	ToolChoice       interface{}        `json:"tool_choice"`
	key              string
	messageType      []string
	roleType         []string
}

/*
AppendStandardMessage

adds a standard message to the chatgpt message slice, this message can act as extra context, memory or the current
query
*/
func (c *ChatGpt) AppendStandardMessage(message *messages.StandardMessage) error {
	if err := message.Validate(); err != nil {
		return err
	}

	c.Messages = append(c.Messages, message)
	c.messageType = append(c.messageType, "standard")
	c.roleType = append(c.roleType, message.Role)
	return nil
}

/*
MultiModalBuilder

constructs a MultiModalMessage compatible with openai gpts
*/
type MultiModalBuilder struct {
	message    *messages.MultiModalMessage
	imageTypes []string
	imageBytes [][]byte
	details    []*string
	gpt        *ChatGpt
}

/*
AddTextContent

add text content to gpt multimodal message
*/
func (m *MultiModalBuilder) AddTextContent(text string) *MultiModalBuilder {
	m.message.AppendText(text)
	return m
}

/*
AddImageUrl

add text content to gpt multimodal message
*/
func (m *MultiModalBuilder) AddImageUrl(url string, detail *string) *MultiModalBuilder {
	m.message.AppendImageUrl(url, detail)
	m.details = append(m.details, detail)
	return m
}

/*
AddImageB64

add base64 image bytes to gpt, in preferred gpt standard
*/
func (m *MultiModalBuilder) AddImageB64(imageBytes []byte, detail *string, imageType string) *MultiModalBuilder {
	startPos := len(m.message.Content)
	m.message.AppendImageBytes(imageBytes, detail)
	m.message.Content[startPos].ImageUrl.Url = fmt.Sprintf(
		"data:image/%s;base64,%s",
		imageType,
		m.message.Content[startPos].ImageUrl.Url)

	m.imageTypes = append(m.imageTypes, imageType)
	m.details = append(m.details, detail)
	m.imageBytes = append(m.imageBytes, imageBytes)

	return m
}

/*
Build

appends the multimodal message to the message slice, if the message is valid
*/
func (m *MultiModalBuilder) Build() error {
	supported := []string{
		"png", "jpeg", "jpg", "webp", "gif",
	}

	detail := []string{
		"high", "low", "auto",
	}

	for _, det := range m.details {
		if det != nil {
			if !helper.Contains[string](detail, *det) {
				return fmt.Errorf("%s is not a valid detail", *det)
			}
		}
	}

	for _, imgTyp := range m.imageTypes {
		if !helper.Contains[string](supported, imgTyp) {
			return fmt.Errorf("%s is not a supported image type", imgTyp)
		}
	}

	for _, img := range m.imageBytes {
		if len(img) > 20*MEGABYTE {
			return errors.New("too many bytes in the image, the limit is 20MB")
		}
	}

	if err := m.message.Validate(); err != nil {
		return err
	}

	m.gpt.messageType = append(m.gpt.messageType, "multimodal")
	m.gpt.Messages = append(m.gpt.Messages, m.message)
	m.gpt.roleType = append(m.gpt.roleType, m.message.Role)

	return nil
}

/*
GetMultiModalMessage

gets a multimodal message builder, the builder upon building will be added to the message slice
*/
func (c *ChatGpt) GetMultiModalMessage(role string) *MultiModalBuilder {
	return &MultiModalBuilder{
		message: &messages.MultiModalMessage{Role: role},
		gpt:     c,
	}
}

func (c *ChatGpt) AppendAssistantMessage(message *messages.AssistantMessage) error {
	if err := message.Validate(); err != nil {
		return err
	}

	c.messageType = append(c.messageType, "assistant")
	c.Messages = append(c.Messages, message)
	c.roleType = append(c.roleType, message.Role)
	return nil
}

func (c *ChatGpt) PopMessage(index uint) {
	delMes := helper.DeleteByIndex[llmMessages]
	delStr := helper.DeleteByIndex[string]

	llmMess := delMes(c.Messages, index)
	mType := delStr(c.messageType, index)
	roleSlice := delStr(c.roleType, index)

	c.Messages = llmMess
	c.messageType = mType
	c.roleType = roleSlice
}

/*
InitChatGpt

initialize a ChatGpt instance with certain filled values
*/
func InitChatGpt(
	apiKey string,
	model string) (error, *ChatGpt) {

	if apiKey == "" {
		return errors.New("openai settings received empty api key"), nil
	}
	if model == "" {
		return errors.New("openai settings received empty model name"), nil
	}

	if val, ok := getEngineMap()[model]; !ok {
		return fmt.Errorf("gpt model %s is not permitted", val.Name), nil
	}

	c := ChatGpt{
		Model: model,
		key:   apiKey,
	}

	c.Messages = make([]llmMessages, 0, 10)
	c.messageType = make([]string, 0, len(c.Messages))
	c.roleType = make([]string, 0, len(c.Messages))

	return nil, &c
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
func validateResponseFormat(responseFormat map[string]string, engine engine) error {

	validSlice := []string{
		"json_object", "text",
	}

	containsString := helper.Contains[string]

	if len(responseFormat) > 1 {
		return fmt.Errorf("response format musto only have one key, detected %d keys", len(responseFormat))
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
validateMessageType

validates the messages provided are compatible with the model being used
*/
func validateMessageType(engine engine, messageTypeSlice []string) error {
	var isMM bool

	for _, typ := range messageTypeSlice {
		if typ == "multimodal" {
			isMM = true
			break
		}
	}

	if isMM && !engine.Multimodal {
		return fmt.Errorf("cannot use %s with mutimodal messages", engine.Name)
	}

	return nil
}

func validateMessages(messageSlice []llmMessages) error {
	for _, i := range messageSlice {
		if err := i.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func validateTools(engine engine, tool messages.Tool) error {
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
func (c *ChatGpt) Validate() error {
	floatHelper := helper.IsBetween[float32]
	intHelper := helper.IsBetween[uint8]

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

	var engine engine
	var ok bool

	if engine, ok = getEngineMap()[c.Model]; !ok {
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

	if err := validateMessages(c.Messages); err != nil {
		return err
	}

	if err := validateMessageType(engine, c.messageType); err != nil {
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

func (c *ChatGpt) Chat(ctx context.Context) (error, *bool, *messages.ChatCompletion) {
	var isRateLimit bool

	fmt.Printf("%p, %p \n", &c.Messages, c)
	fmt.Println(len(c.Messages))

	if c.roleType[len(c.roleType)-1] != "user" {
		return fmt.Errorf("the role for the last message must be user, got %s", c.roleType), &isRateLimit, nil
	}

	if err := c.Validate(); err != nil {
		return err, &isRateLimit, nil
	}

	url := "https://api.openai.com/v1/chat/completions"
	jsonBytes, err := json.Marshal(c)

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
	pRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.key))

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

func (c *ChatGpt) DeepCopy() *ChatGpt {

	//frq := *c.FrequencyPenalty
	//lb := *c.LogitBias
	//lp := *c.LogProbs
	//maxTok := *c.MaxTokens
	//n := *c.N
	//pr := *c.PresencePenalty
	//rp := *c.ResponseFormat
	//seed := *c.Seed
	//stop := *c.Stop
	//stream := *c.Stream
	//temp := *c.Temperature
	//topP := *c.TopP
	//tools := *c.Tools

	return &ChatGpt{
		Model:            c.Model,
		FrequencyPenalty: c.FrequencyPenalty,
		Messages:         c.Messages,
		LogitBias:        c.LogitBias,
		LogProbs:         c.LogProbs,
		MaxTokens:        c.MaxTokens,
		N:                c.N,
		PresencePenalty:  c.PresencePenalty,
		ResponseFormat:   c.ResponseFormat,
		Seed:             c.Seed,
		Stop:             c.Stop,
		Stream:           c.Stream,
		Temperature:      c.Temperature,
		TopP:             c.TopP,
		Tools:            c.Tools,
		ToolChoice:       c.ToolChoice,
		key:              c.key,
		messageType:      c.messageType,
		roleType:         c.roleType,
	}
}
