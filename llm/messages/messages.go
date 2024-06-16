package messages

import (
	"agent/helper"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

/*
containsRole

checks if a role is present in a slice of strings
*/
func containsRole(roleSlice []string, role string) bool {
	contains := helper.Contains[string]
	return contains(roleSlice, role)
}

/*
invalidRoleError

formats invalid roles for errors in the most optimal way
*/
func invalidRoleError(roleSlice []string, role string) error {
	secondHalf := fmt.Sprintf("you provided %s", role)
	s := strings.Builder{} // strings are immutable, to optimize for fewer reallocations use string builder

	length := len(secondHalf)
	for _, i := range roleSlice {
		length += len(i) + 2
	}

	length += len("are valid roles") // calculates total length needed for the string
	s.Grow(length)                   // preallocate space

	for idx, i := range roleSlice {

		s.WriteString(i)
		if idx != len(roleSlice)-1 {
			s.WriteString(",")
		}

		s.WriteString(" ")
	}

	s.WriteString("are valid roles")
	s.WriteString(" ")
	s.WriteString(secondHalf)

	return errors.New(s.String())
}

/*
StandardMessage

a standard message chat request that only contains text
*/
type StandardMessage struct {
	Role    string  `json:"role"`
	Content string  `json:"message"`
	Name    *string `json:"name,omitempty"`
}

/*
Validate

ensure the message is valid for llm ingestion
*/
func (s StandardMessage) Validate() error {
	roles := [2]string{
		"system", "user",
	}

	if !containsRole(roles[:], s.Role) {
		return invalidRoleError(roles[:], s.Role)
	}

	if s.Content == "" {
		return errors.New("message is empty")
	}

	return nil
}

/*
imageContent

represents the image that would be passed into multimodal requests. The image could be a base64 string, or it can be
an url to an image
*/
type imageContent struct {
	Url    string  `json:"url"`
	Detail *string `json:"detail,omitempty"`
}

/*
validate

ensures a image is not empty
*/
func (i *imageContent) validate() error {
	err := errors.New("image message is invalid")

	if strings.HasPrefix(i.Url, "data:image/") || strings.HasPrefix(i.Url, "http") {
		return nil
	}

	return err
}

/*
multimodalContent

what a multimodal message is composed of
it can contain one of two Types image_url, or text
based on what category it is the relevant section should be filled
*/
type multimodalContent struct {
	Type     string        `json:"type"`
	Text     *string       `json:"text,omitempty"`
	ImageUrl *imageContent `json:"image_url,omitempty"`
}

/*
validate

ensures multimodal message contains usable valid data
*/
func (mc *multimodalContent) validate() error {

	if mc.Type == "image_url" {
		if mc.ImageUrl != nil {
			return mc.ImageUrl.validate()
		} else {
			return errors.New("when using Type image_url, ImageUrl cannot be nil")
		}
	} else if mc.Type == "text" {
		if mc.Text == nil {
			return errors.New("when using type text, Text cannot be nil")
		}

		if *mc.Text == "" {
			return errors.New("when using type text, Text cannot be empty")
		}
	} else {
		return fmt.Errorf("the Type cannot be %s, valid types are image_url or text", mc.Type)
	}

	return nil
}

/*
MultiModalMessage

the multimodal message being sent to the llm
*/
type MultiModalMessage struct {
	Role    string              `json:"role"`
	Content []multimodalContent `json:"message"`
	Name    *string             `json:"name,omitempty"`
}

/*
AppendImageUrl

adds an image url to the message array
*/
func (m *MultiModalMessage) AppendImageUrl(url string, detail *string) {
	m.Content = append(m.Content, multimodalContent{
		Type: "image_url",
		ImageUrl: &imageContent{
			Url:    url,
			Detail: detail,
		},
	})
}

/*
AppendImageBytes

converts bytes to base64 and adds then to the message array
*/
func (m *MultiModalMessage) AppendImageBytes(imageBytes []byte, detail *string) {
	encodedStr := base64.StdEncoding.EncodeToString(imageBytes)
	//b64 := fmt.Sprintf("data:image/%s;base64,%s", imageType, encodedStr)
	m.Content = append(m.Content, multimodalContent{
		Type: "image_url",
		ImageUrl: &imageContent{
			Url:    encodedStr,
			Detail: detail,
		},
	})
}

/*
AppendText

appends text to the multimodal message
*/
func (m *MultiModalMessage) AppendText(text string) {
	m.Content = append(m.Content, multimodalContent{
		Type: "text",
		Text: &text,
	})
}

/*
Validate

ensures the multimodal message is compliant
*/
func (m *MultiModalMessage) Validate() error {
	roles := [2]string{
		"system", "user",
	}

	if !containsRole(roles[:], m.Role) {
		return invalidRoleError(roles[:], m.Role)
	}

	for index, content := range m.Content {
		if err := content.validate(); err != nil {
			return fmt.Errorf("at message %d. received error: %v", index, err)
		}
	}

	return nil
}

/*
ToolCall

a struct detailing a tool that the chatbot can call
*/
type ToolCall struct {
	Id       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

/*
AssistantMessage

a message generated by the assistant, details the response through the message
along with the appropriate tool calls
*/
type AssistantMessage struct {
	Content   *string     `json:"message,omitempty"`
	Role      string      `json:"role,omitempty"`
	Name      *string     `json:"name,omitempty"`
	ToolCalls *[]ToolCall `json:"tool_calls,omitempty"`
}

/*
Validate

assures an AssistantMessage is compliant
*/
func (g AssistantMessage) Validate() error {
	roles := [2]string{
		"assistant",
	}

	if !containsRole(roles[:], g.Role) {
		return invalidRoleError(roles[:], g.Role)
	}

	state := false

	if g.Content != nil {
		state = true

		if *g.Content == "" {
			return errors.New("AssistantMessage Content cannot be empty")
		}
	}

	if g.ToolCalls != nil {
		state = !state
	}

	if !state {
		return errors.New("either ToolCalls or Content must be provided but not both")
	}

	return nil
}

/*
ToolFunction

a struct for tool/function calls
*/
type ToolFunction struct {
	Description *string                `json:"description,omitempty"`
	Name        string                 `json:"name"`
	Parameters  map[string]interface{} `json:"parameters"`
}

/*
Tool

a tool call with its tool data, in the future tool calls will go beyond function calling
*/
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

/*
Message

the message returned by the chatbot
*/
type Message struct {
	Content   *string    `json:"message"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Role      string     `json:"role"`
}

/*
LogprobContent

the log probabilities generated by the chatbot
*/
type LogprobContent struct {
	Token   string   `json:"token"`
	Logprob int32    `json:"logprob"`
	Bytes   *[]int32 `json:"bytes"`
}

/*
FullLogprobContent

the log probabilities generated by the chatbot along with Top Log Probabilities
*/
type FullLogprobContent struct {
	LogprobContent
	TopLogprobs []LogprobContent `json:"top_logprobs"`
}

/*
Choice

the response choices provided by the chatbot
*/
type Choice struct {
	FinishReason string              `json:"finish_reason"`
	Index        int32               `json:"index"`
	Message      Message             `json:"message"`
	Logprobs     *FullLogprobContent `json:"logprobs,omitempty"`
}

/*
ChatCompletion

the response provided by the chatbot
*/
type ChatCompletion struct {
	Id                string   `json:"id"`
	Created           int64    `json:"created"`
	Choices           []Choice `json:"choices"`
	Model             string   `json:"model"`
	SystemFingerprint string   `json:"system_fingerprint"`
	Object            string   `json:"object"`
	Usage             struct {
		PromptTokens     int32 `json:"prompt_tokens"`
		CompletionTokens int32 `json:"completion_tokens"`
		TotalTokens      int32 `json:"total_tokens"`
	} `json:"usage"`
}

/*
ToAssistant

converts chat completions to a slice of assistant messages
*/
func (c *ChatCompletion) ToAssistant() []AssistantMessage {

	retSlice := make([]AssistantMessage, len(c.Choices))

	for index, choice := range c.Choices {
		retSlice[index] = AssistantMessage{
			Role:      "assistant",
			Content:   choice.Message.Content,
			ToolCalls: &choice.Message.ToolCalls,
		}
	}

	return retSlice
}
