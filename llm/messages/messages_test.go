package messages

import (
	"encoding/base64"
	"testing"
)

// TestInvalidRoleError
/*
unit test
*/
func TestInvalidRoleError(t *testing.T) {
	err := invalidRoleError([]string{"user", "assistant"}, "system")

	requiredString := `user, assistant are valid roles you provided system`
	if err == nil {
		t.Fatal("invalidRoleError should never return nil")
	}

	if err.Error() != requiredString {
		t.Fatalf(`expected error message: "%s", received: "%s"`, requiredString, err.Error())
	}
}

// TestStandardMessage_Validate
/*
unit test
*/
func TestStandardMessage_Validate(t *testing.T) {
	tests := []struct {
		message StandardMessage
		pass    bool
		name    string
	}{
		{message: StandardMessage{Role: "", Content: ""}, pass: false, name: "missing role and message"},
		{message: StandardMessage{Role: "user", Content: ""}, pass: false, name: "missing message"},
		{message: StandardMessage{Role: "assistant", Content: "hello"}, pass: false, name: "invalid role"},
		{message: StandardMessage{Role: "system", Content: "hello"}, pass: true, name: "testing valid role system"},
		{message: StandardMessage{Role: "user", Content: "hello"}, pass: true, name: "testing valid role user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (tt.message.Validate() != nil) == tt.pass {
				expected := func(state bool) string {
					if state {
						return "nil"
					}
					return "error"
				}
				t.Errorf(
					"expected TestStandardMessage_Validate to return an %s but got %s",
					expected(tt.pass),
					expected(!tt.pass))
			}
		})
	}
}

func TestImageContent_Validate(t *testing.T) {
	tests := []struct {
		content imageContent
		pass    bool
		name    string
	}{
		{content: imageContent{Url: ""}, pass: false, name: "empty url"},
		{content: imageContent{Url: "invalid url"}, pass: false, name: "invalid url"},
		{content: imageContent{Url: "https://somewebsite.net/test.jpg"}, pass: true, name: "valid url"},
		{content: imageContent{Url: "data:image/"}, pass: true, name: "valid base64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (tt.content.validate() != nil) == tt.pass {
				expected := func(state bool) string {
					if state {
						return "nil"
					}
					return "error"
				}
				t.Errorf(
					"expected TestImageContent_Validate to return an %s but got %s",
					expected(tt.pass),
					expected(!tt.pass))
			}
		})
	}
}

func TestMultiModalContent_Validate(t *testing.T) {

	testText := "hello"
	tests := []struct {
		content multimodalContent
		pass    bool
		name    string
	}{
		{content: multimodalContent{Type: "image_url", ImageUrl: &imageContent{Url: "https://bench-ai.com/test.png"}}, pass: true, name: "valid image message"},
		{content: multimodalContent{Type: "text", Text: &testText}, pass: true, name: "valid text message"},
		{content: multimodalContent{Type: "data"}, pass: false, name: "invalid type"},
		{content: multimodalContent{Type: "image_url", ImageUrl: nil, Text: &testText}, pass: false, name: "nil message for image_url"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (tt.content.validate() != nil) == tt.pass {
				expected := func(state bool) string {
					if state {
						return "nil"
					}
					return "error"
				}
				t.Errorf(
					"expected TestImageContent_Validate to return an %s but got %s",
					expected(tt.pass),
					expected(!tt.pass))
			}
		})
	}
}

func TestMultiModalMessage_AppendImageBytes(t *testing.T) {
	mm := MultiModalMessage{Role: "user"}
	bytes := []byte("this is a test string")
	intBytes := make([]byte, 1024)

	for i := range intBytes {
		intBytes[i] = 1
	}

	mm.AppendImageBytes(bytes, nil)
	mm.AppendImageBytes(intBytes, nil)

	byteSlice := make([]*[]byte, 2)

	byteSlice[0] = &bytes
	byteSlice[1] = &intBytes

	for i, currBytes := range byteSlice {
		firstContent := mm.Content[i]
		b64 := firstContent.ImageUrl.Url

		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			t.Fatal(err)
		}

		for idx, byt := range data {
			if byt != (*currBytes)[idx] {
				t.Fatal("bytes are not the same")
			}
		}
	}
}

func TestMultiModalMessage_AppendImageUrl(t *testing.T) {
	mm := MultiModalMessage{Role: "user"}
	det := "auto"
	mm.AppendImageUrl("https://data.png", &det)

	if len(mm.Content) != 1 {
		t.Fatal("failed to append the ImageUrl")
	}

	if mm.Content[0].ImageUrl.Detail != &det {
		t.Fatal("detail was not saved")
	}
}

func TestMultiModalMessage_AppendText(t *testing.T) {
	mm := MultiModalMessage{Role: "user"}

	prompt := "hello, how are you?"
	mm.AppendText(prompt)

	if len(mm.Content) != 1 {
		t.Fatal("failed to append the ImageUrl")
	}

	if *mm.Content[0].Text != prompt {
		t.Fatal("text was not saved")
	}
}

func TestMultiModalMessage_Validate(t *testing.T) {
	tests := []struct {
		message MultiModalMessage
		pass    bool
		name    string
	}{
		{message: MultiModalMessage{Role: "user"}, pass: true, name: "user role"},
		{message: MultiModalMessage{Role: "system"}, pass: true, name: "system role"},
		{message: MultiModalMessage{Role: "..."}, pass: false, name: "invalid role"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (tt.message.Validate() != nil) == tt.pass {
				expected := func(state bool) string {
					if state {
						return "nil"
					}
					return "error"
				}
				t.Errorf(
					"expected TestImageContent_Validate to return an %s but got %s",
					expected(tt.pass),
					expected(!tt.pass))
			}
		})
	}
}

func TestAssistantMessage_Validate(t *testing.T) {
	content := "I am doing well how are you"
	var emptyContent string

	toolCall := ToolCall{
		Type: "function",
		Function: struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		}{Name: "data", Arguments: "data"},
	}

	tests := []struct {
		message AssistantMessage
		pass    bool
		name    string
	}{
		{message: AssistantMessage{Role: "assistant", Content: &content}, pass: true, name: "assistant role with content"},
		{message: AssistantMessage{Role: "assistant", ToolCalls: &[]ToolCall{toolCall}}, pass: true, name: "assistant role with tool call"},
		{message: AssistantMessage{Role: "user"}, pass: false, name: "invalid role"},
		{message: AssistantMessage{Role: "assistant", ToolCalls: &[]ToolCall{toolCall}, Content: &content}, pass: false, name: "both responses provided"},
		{message: AssistantMessage{Role: "assistant"}, pass: false, name: "valid role with no responses"},
		{message: AssistantMessage{Role: "assistant", Content: &emptyContent}, pass: false, name: "valid role with empty content"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (tt.message.Validate() != nil) == tt.pass {
				expected := func(state bool) string {
					if state {
						return "nil"
					}
					return "error"
				}
				t.Errorf(
					"expected TestImageContent_Validate to return an %s but got %s",
					expected(tt.pass),
					expected(!tt.pass))
			}
		})
	}
}

func TestChatCompletion_ToAssistant(t *testing.T) {

	cont := "test content"
	messageSlice := []Message{
		{Content: &cont},
		{Content: &cont},
		{Content: &cont},
		{Content: &cont},
	}

	choiceSlice := make([]Choice, len(messageSlice))

	for idx, message := range messageSlice {
		choiceSlice[idx] = Choice{
			Message: message,
		}
	}

	completion := ChatCompletion{
		Choices: choiceSlice,
	}

	assistantSlice := completion.ToAssistant()

	if len(assistantSlice) != len(messageSlice) {
		t.Fatal("assistant messages were not all converted")
	}
}
