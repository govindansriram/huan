package messages

import (
	"encoding/base64"
	"testing"
)

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

func TestStandardMessage_validate(t *testing.T) {
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
			if (tt.message.validate() != nil) == tt.pass {
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

func TestConversationBuilder_AddStandardMessage(t *testing.T) {
	mess := StandardMessage{
		Role:    "user",
		Content: "testing access",
	}

	builder := (&ConversationBuilder{}).AddStandardMessage(&mess)

	t.Run("standard message added", func(t *testing.T) {
		if builder.conversation[0] != &mess {
			t.Errorf("standard message was not added")
		}
	})

	t.Run("standard message role added", func(t *testing.T) {
		if builder.roles[0] != "user" {
			t.Errorf("standard message role was not added")
		}
	})

	t.Run("standard message type added", func(t *testing.T) {
		if builder.messageTypes[0] != "standard" {
			t.Errorf("standard message was not added")
		}
	})
}

func TestMultiModalContent_validate(t *testing.T) {

	testText := "hello"
	var invalidText string
	tests := []struct {
		content MultimodalContent
		pass    bool
		name    string
	}{
		{
			content: MultimodalContent{
				Type: "image_url",
				ImageUrl: &imageContent{
					Url: "https://bench-ai.com/test.png"},
			},
			pass: true,
			name: "valid image message",
		},
		{
			content: MultimodalContent{
				Type: "text",
				Text: &testText,
			},
			pass: true,
			name: "valid text message",
		},
		{
			content: MultimodalContent{
				Type: "text",
				Text: nil,
			},
			pass: false,
			name: "nil text",
		},
		{
			content: MultimodalContent{
				Type: "text",
				Text: &invalidText,
			},
			pass: false,
			name: "empty text",
		},
		{
			content: MultimodalContent{
				Type: "data",
			},
			pass: false,
			name: "invalid type",
		},
		{
			content: MultimodalContent{
				Type:     "image_url",
				ImageUrl: nil,
				Text:     &testText,
			},
			pass: false,
			name: "nil message for image_url",
		},
		{
			content: MultimodalContent{
				Type:     "image_url",
				ImageUrl: &imageContent{},
				Text:     &testText,
			},
			pass: false,
			name: "nil url in image url",
		},
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

	mm.AppendImageBytes(bytes, nil, "png")
	mm.AppendImageBytes(intBytes, nil, "png")

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
			if (tt.message.validate() != nil) == tt.pass {
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
			if (tt.message.validate() != nil) == tt.pass {
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

func TestConversationBuilder_AddMultimodalMessage(t *testing.T) {
	mess := MultiModalMessage{
		Role: "user",
	}

	detail := "low"
	mess.AppendImageUrl("https://test.com/img.png", &detail)

	bb := &ConversationBuilder{}
	bb.AddMultimodalMessage(&mess)

	t.Run("mm message added", func(t *testing.T) {
		if bb.conversation[0] != &mess {
			t.Errorf("standard message was not added")
		}
	})

	t.Run("mm message role added", func(t *testing.T) {
		if bb.roles[0] != "user" {
			t.Errorf("mm message role was not added")
		}
	})

	t.Run("mm message type added", func(t *testing.T) {
		if bb.messageTypes[0] != "multimodal" {
			t.Errorf("mm message was not added")
		}
	})
}

func TestConversationBuilder_AddAssistantMessage(t *testing.T) {
	content := "test content"
	mess := AssistantMessage{
		Role:    "assistant",
		Content: &content,
	}

	bb := &ConversationBuilder{}
	bb.AddAssistantMessage(&mess)

	t.Run("assistant message added", func(t *testing.T) {
		if bb.conversation[0] != &mess {
			t.Errorf("assistant message was not added")
		}
	})

	t.Run("assistant role added", func(t *testing.T) {
		if bb.roles[0] != "assistant" {
			t.Errorf("assistant role was not added")
		}
	})

	t.Run("assistant message type added", func(t *testing.T) {
		if bb.messageTypes[0] != "assistant" {
			t.Errorf("assistant message was not added")
		}
	})
}

func getBuilder() *ConversationBuilder {
	builder := &ConversationBuilder{}

	builder.AddStandardMessage(&StandardMessage{
		Role:    "user",
		Content: "testing content",
	})

	mm := &MultiModalMessage{
		Role: "user",
	}
	mm.AppendText("testing content")
	builder.AddMultimodalMessage(mm)

	data := "testing content"
	as := &AssistantMessage{}
	as.Content = &data
	as.Role = "assistant"
	builder.AddAssistantMessage(as)

	return builder
}

func TestConversationBuilder_ConvertToAssistant(t *testing.T) {
	builder := getBuilder()
	t.Run("got assistant message", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("recovery was unnessecary")
			}
		}()

		as := builder.ConvertToAssistant(2)

		if as != builder.conversation[2] {
			t.Errorf("data is not equivalent")
		}
	})
}

func TestConversationBuilder_ConvertToStandard(t *testing.T) {
	builder := getBuilder()
	t.Run("got standard message", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("recovery was unnessecary")
			}
		}()

		stan := builder.ConvertToStandard(0)

		if stan != builder.conversation[0] {
			t.Errorf("data is not equivalent")
		}
	})
}

func TestConversationBuilder_ConvertToMultiModal(t *testing.T) {
	builder := getBuilder()
	t.Run("got mm message", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("recovery was unnessecary")
			}
		}()

		stan := builder.ConvertToMultiModal(1)

		if stan != builder.conversation[1] {
			t.Errorf("data is not equivalent")
		}
	})
}

func TestConversationBuilder_Pop(t *testing.T) {
	bb := getBuilder()

	for range 50 {
		bb.AddStandardMessage(&StandardMessage{
			Role:    "user",
			Content: "test message",
		})
	}

	t.Run("delete middle element", func(t *testing.T) {
		start := bb.Size()
		bb.Pop(25)

		if bb.Size() != start-1 {
			t.Errorf("improper deletion")
		}

		if bb.GetMessageType(bb.Size()-1) != "standard" {
			t.Errorf("improper deletion")
		}
	})

	t.Run("delete invalid index", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("recovery was nessecary")
			}
		}()

		bb.Pop(1000)
	})

	t.Run("delete all indices", func(t *testing.T) {
		sz := bb.Size()

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("recovery was unnessecary")
			}
		}()

		for range sz {
			bb.Pop(bb.Size() - 1)

			if bb.Size() != 0 {
				tp := bb.GetMessageType(bb.Size() - 1)

				switch tp {
				case "standard":
					_ = bb.ConvertToStandard(bb.Size() - 1)
				case "multimodal":
					_ = bb.ConvertToMultiModal(bb.Size() - 1)
				case "assistant":
					_ = bb.ConvertToAssistant(bb.Size() - 1)
				}
			}
		}
	})
}

func TestConversationBuilder_Build(t *testing.T) {

	t.Run("invalid last message role", func(t *testing.T) {
		bb := getBuilder()
		err, _ := bb.Build()

		if err == nil {
			t.Errorf("conversation should have failed to build")
		}
	})

	t.Run("invalid messages", func(t *testing.T) {
		bb := getBuilder()
		bb.AddStandardMessage(&StandardMessage{
			Role:    "user",
			Content: "",
		})

		err, _ := bb.Build()

		if err == nil {
			t.Errorf("conversation should have failed to build")
		}
	})

	t.Run("valid messages", func(t *testing.T) {
		bb := getBuilder()
		bb.AddStandardMessage(&StandardMessage{
			Role:    "user",
			Content: "test",
		})

		err, _ := bb.Build()

		if err != nil {
			t.Errorf("conversation should have built")
		}
	})

	t.Run("conversations are copies", func(t *testing.T) {
		bb := getBuilder()
		bb.AddStandardMessage(&StandardMessage{
			Role:    "user",
			Content: "test",
		})
		err, convo1 := bb.Build()

		bb.AddStandardMessage(&StandardMessage{
			Role:    "user",
			Content: "test",
		})
		err2, convo2 := bb.Build()

		if err != nil || err2 != nil {
			t.Errorf("conversation should have built")
		}

		if len(convo1) == len(convo2) {
			t.Errorf("conversation should be copies but share same address")
		}

	})
}
