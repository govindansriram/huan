package model

import (
	"agent/llm/messages"
	"math"
	"strings"
	"testing"
)

func TestChatGpt_AppendStandardMessage(t *testing.T) {
	chat := ChatGpt{
		Model: "gpt-4-turbo",
		key:   "test-key",
	}

	tests := []struct {
		message messages.StandardMessage
		pass    bool
		name    string
	}{
		{
			message: messages.StandardMessage{Role: "user", Content: "hello world"},
			pass:    true, name: "valid standard message",
		},
		{
			message: messages.StandardMessage{Role: "assistant", Content: "bruh"},
			pass:    false, name: "assistant role with tool call",
		},
	}

	boolToInt := func(state bool) uint8 {
		var ret uint8
		if state {
			ret += 1
		}

		return ret
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (chat.AppendStandardMessage(&tt.message) != nil) == tt.pass || uint8(len(chat.Messages)) != boolToInt(tt.pass) {
				expected := func(state bool) string {
					if state {
						return "nil"
					}
					return "error"
				}
				t.Errorf(
					"expected TestChatGpt_AppendStandardMessage to return an %s but got %s",
					expected(tt.pass),
					expected(!tt.pass))
			}

			if len(chat.Messages) > 0 {
				var mess []llmMessages
				chat.Messages = mess
			}
		})
	}
}

func TestMultiModalBuilder_AddTextContent(t *testing.T) {
	mess := MultiModalBuilder{
		message: &messages.MultiModalMessage{Role: "user"},
	}

	text := "text"

	mess.AddTextContent(text)

	if len(mess.message.Content) != 1 {
		t.Fatal("content was not added")
	}
}

func TestMultiModalBuilder_AddImageUrl(t *testing.T) {
	mess := MultiModalBuilder{
		message: &messages.MultiModalMessage{Role: "user"},
	}

	text := "https://bench-ai.com/logo.jpg"
	det := "high"

	mess.AddImageUrl(text, &det)

	if len(mess.message.Content) != 1 {
		t.Fatal("content was not added")
	}
}

func TestMultiModalBuilder_AddImageB64(t *testing.T) {
	data := make([]byte, int64(math.Pow(1024, 1)*20)+1)
	detail := "auto"
	imageType := "jpeg"

	for range data {
		data = append(data, 1)
	}

	mess := (&MultiModalBuilder{
		message: &messages.MultiModalMessage{Role: "user"},
	}).AddImageB64(data, &detail, imageType)

	url := mess.message.Content[0].ImageUrl.Url

	pref := "data:image/jpeg;base64"

	if !strings.HasPrefix(url, pref) {
		t.Fatal("failed to gpt specific base44 prefix")
	}
}

func TestMultiModalBuilder_Build(t *testing.T) {
	chat := ChatGpt{
		Model: "gpt-4o",
		key:   "data",
	}

	mm := messages.MultiModalMessage{
		Role: "user",
	}

	data := make([]byte, int64(math.Pow(1024, 2)*20)+1)

	for idx := range data {
		data[idx] = 1
	}

	validData := make([]byte, 1000)
	copy(validData, data)

	builder := MultiModalBuilder{
		message: &mm,
		gpt:     &chat,
	}

	detail := "auto"
	badDetail := "medium"

	tests := []struct {
		pass        bool
		name        string
		typeSlice   []string
		byteSlice   [][]byte
		detailSlice []*string
	}{
		{typeSlice: []string{"png"}, byteSlice: [][]byte{validData}, detailSlice: []*string{nil}, pass: true, name: "valid 1-layer builder"},
		{typeSlice: []string{"svg"}, byteSlice: [][]byte{validData}, detailSlice: []*string{nil}, pass: false, name: "invalid type"},
		{typeSlice: []string{"jpeg"}, byteSlice: [][]byte{validData}, detailSlice: []*string{&badDetail}, pass: false, name: "invalid type"},
		{typeSlice: []string{"jpeg"}, byteSlice: [][]byte{data}, detailSlice: []*string{nil}, pass: false, name: "invalid byteData (too long)"},
		{typeSlice: []string{"jpeg", "png", "webp"}, byteSlice: [][]byte{validData, validData}, detailSlice: []*string{nil, &detail}, pass: true, name: "valid multi data"},
		{typeSlice: []string{"jpeg", "png"}, byteSlice: [][]byte{validData, data}, detailSlice: []*string{nil, &detail}, pass: false, name: "invalid multi data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			builder.imageBytes = tt.byteSlice
			builder.imageTypes = tt.typeSlice
			builder.details = tt.detailSlice

			if (builder.Build() != nil) == tt.pass {
				expected := func(state bool) string {
					if state {
						return "nil"
					}
					return "error"
				}
				t.Errorf(
					"expected TestChatGpt_AppendStandardMessage to return an %s but got %s",
					expected(tt.pass),
					expected(!tt.pass))
			}
		})
	}

	if len(chat.Messages) != 2 {
		t.Fatal("messages not added properly")
	}
}

func TestChatGpt_AppendAssistantMessage(t *testing.T) {
	cont := "add data"
	assist := messages.AssistantMessage{
		Role:    "assistant",
		Content: &cont,
	}

	assist2 := messages.AssistantMessage{
		Role:    "system",
		Content: &cont,
	}

	chat := ChatGpt{
		Model: "gpt-4o",
		key:   "data",
	}

	tests := []struct {
		pass    bool
		name    string
		message *messages.AssistantMessage
	}{
		{message: &assist, pass: true, name: "valid assistant message"},
		{message: &assist2, pass: false, name: "invalid assistant message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if (chat.AppendAssistantMessage(tt.message) != nil) == tt.pass || (tt.pass && len(chat.Messages) != 1) {
				expected := func(state bool) string {
					if state {
						return "nil"
					}
					return "error"
				}
				t.Errorf(
					"expected TestChatGpt_AppendStandardMessage to return an %s but got %s",
					expected(tt.pass),
					expected(!tt.pass))
			}
		})
	}
}

func TestChatGpt_PopMessage(t *testing.T) {
	chat := ChatGpt{
		Model: "gpt-4o",
		key:   "data",
	}

	for range 50 {
		_ = chat.AppendStandardMessage(&messages.StandardMessage{
			Role:    "user",
			Content: "test message",
		})
	}

	if err := chat.PopMessage(25); err != nil {
		t.Error(err)
	}

	if err := chat.PopMessage(100); err == nil {
		t.Errorf("deleted from invalid index")
	}

	for range chat.Messages {
		if err := chat.PopMessage(0); err != nil {
			t.Errorf("threw unnessecary error, %v", err)
		}
	}
}

func TestInitChatGpt(t *testing.T) {
	tests := []struct {
		key   string
		model string
		pass  bool
		name  string
	}{
		{key: "", model: "gpt4-o", pass: false, name: "valid key"},
		{key: "123", model: "", pass: false, name: "valid model"},
		{key: "123", model: "gpt4", pass: true, name: "valid model"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err, _ := InitChatGpt(tt.key, tt.model)
			if (err != nil) == tt.pass {
				expected := func(state bool) string {
					if state {
						return "nil"
					}
					return "error"
				}
				t.Errorf(
					"expected TestChatGpt_AppendStandardMessage to return an %s but got %s",
					expected(tt.pass),
					expected(!tt.pass))
			}
		})
	}

}

func TestValidateResponseFormat(t *testing.T) {

	validEngine := getEngineMap()["gpt-4o"]
	invalidEngine := engine{
		HasJsonMode: false,
		Name:        "test-llm",
	}

	failTable := []map[string]string{
		{
			"type":  "text",
			"other": "json_object",
		},
		{
			"type": "json_obje",
		},
		{
			"other": "json_obje",
		},
	}

	passTable := []map[string]string{
		{
			"type": "text",
		},
		{
			"type": "json_object",
		},
	}

	tests := []struct {
		engine         engine
		responseFormat map[string]string
		pass           bool
		name           string
	}{
		{engine: validEngine, responseFormat: failTable[0], pass: false, name: "both type and other"},
		{engine: validEngine, responseFormat: failTable[1], pass: false, name: "invalid type"},
		{engine: validEngine, responseFormat: failTable[2], pass: false, name: "missing type"},
		{engine: validEngine, responseFormat: passTable[0], pass: true, name: "valid type"},
		{engine: validEngine, responseFormat: passTable[1], pass: true, name: "valid type"},
		{engine: invalidEngine, responseFormat: passTable[1], pass: false, name: "invalid engine"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (validateResponseFormat(tt.responseFormat, tt.engine) != nil) == tt.pass {
				expected := func(state bool) string {
					if state {
						return "nil"
					}
					return "error"
				}
				t.Errorf(
					"expected TestChatGpt_AppendStandardMessage to return an %s but got %s",
					expected(tt.pass),
					expected(!tt.pass))
			}
		})
	}
}

func TestValidateTools(t *testing.T) {

	func1 := messages.ToolFunction{
		Name:       "test",
		Parameters: map[string]interface{}{"test": "test"},
	}

	t1 := messages.Tool{
		Type:     "function",
		Function: func1,
	}

	invalidEngine := engine{
		FunctionCalling: false,
		Name:            "test-llm",
	}

	engine := getEngineMap()["gpt-4o"]

	if err := validateTools(engine, t1); err != nil {
		t.Error("rejected function that was not invalid")
	}

	t1.Type = "f"

	if err := validateTools(engine, t1); err == nil {
		t.Error("failed to reject invalid tool type")
	}

	engine = getEngineMap()["gpt-3.5-turbo"]

	if err := validateTools(invalidEngine, t1); err == nil {
		t.Error("failed to reject invalid engine that has no function calling capabilities")
	}
}

func TestValidateToolChoice(t *testing.T) {

	toolChoicesStringPass := "auto"

	toolChoicesStringFail := "other"

	passData := map[string]interface{}{
		"type": "function",
		"function": map[string]string{
			"name": "my_func",
		},
	}

	failData := map[string]interface{}{
		"type": "function",
		"function": map[string]string{
			"nm": "my_func",
		},
	}

	if err := validateToolChoice(toolChoicesStringPass); err != nil {
		t.Errorf("rejected valid tool choice string, %v", err)
	}

	if err := validateToolChoice(toolChoicesStringFail); err == nil {
		t.Errorf("accpeted invalid tool choice, %v", err)
	}

	if err := validateToolChoice(passData); err != nil {
		t.Errorf("rejected valid tool choice string, %v", err)
	}

	if err := validateToolChoice(failData); err == nil {
		t.Errorf("accpeted invalid tool choice, %v", err)
	}
}

func TestValidateMessageType(t *testing.T) {
	testSlice := []string{
		"standard",
		"assistant",
		"multimodal",
		"assistant",
		"standard",
	}

	eng := getEngineMap()["gpt-4o"]

	err := validateMessageType(eng, testSlice)

	if err != nil {
		t.Error(err)
	}

	eng = engine{
		Name: "fail",
	}

	err = validateMessageType(eng, testSlice)

	if err == nil {
		t.Error("invalid engine succeeded")
	}

}
