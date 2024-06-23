package model

import (
	"encoding/base64"
	"huan/llm/messages"
	"strings"
	"testing"
)

func Test_validateResponseFormat(t *testing.T) {

	validEngine := GetEngineMap()["gpt-4o"]
	invalidEngine := Engine{
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
		engine         Engine
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

func Test_validateTools(t *testing.T) {

	func1 := messages.ToolFunction{
		Name:       "test",
		Parameters: map[string]interface{}{"test": "test"},
	}

	t1 := messages.Tool{
		Type:     "function",
		Function: func1,
	}

	invalidEngine := Engine{
		FunctionCalling: false,
		Name:            "test-llm",
	}

	engine := GetEngineMap()["gpt-4o"]

	if err := validateTools(engine, t1); err != nil {
		t.Error("rejected function that was not invalid")
	}

	t1.Type = "f"

	if err := validateTools(engine, t1); err == nil {
		t.Error("failed to reject invalid tool type")
	}

	engine = GetEngineMap()["gpt-3.5-turbo"]

	if err := validateTools(invalidEngine, t1); err == nil {
		t.Error("failed to reject invalid engine that has no function calling capabilities")
	}
}

func Test_validateToolChoice(t *testing.T) {

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

func Test_checkB64(t *testing.T) {
	t.Run("test invalid base64 string", func(t *testing.T) {
		str := "false"
		if checkB64(str) == nil {
			t.Error("could not catch invalid b64 string")
		}
	})

	t.Run("test large base64", func(t *testing.T) {
		bt := make([]byte, 20*MEGABYTE+1)

		for index := range 20*MEGABYTE + 1 {
			bt[index] = 'a'
		}

		enc := base64.StdEncoding.EncodeToString(bt)

		if checkB64(enc) == nil {
			t.Error("could not catch invalid b64 string")
		}
	})

	t.Run("valid base64", func(t *testing.T) {
		bt := make([]byte, 10)

		for index := range 10 {
			bt[index] = 'a'
		}

		enc := base64.StdEncoding.EncodeToString(bt)

		if checkB64(enc) != nil {
			t.Error("failed valid b64")
		}
	})
}

func Test_handleContent(t *testing.T) {

	t.Run("invalid url", func(t *testing.T) {
		invalidUrl := messages.MultiModalMessage{
			Role: "user",
		}

		invalidUrl.AppendImageUrl("asdasdas", nil)

		if err := handleContent(&invalidUrl.Content[0]); err == nil {
			t.Errorf("should not accept invalid url")
		}

		invalidUrl.Content[0].ImageUrl.Url = "https://data.com/img.jpg"
		if err := handleContent(&invalidUrl.Content[0]); err != nil {
			t.Errorf("should accept invalid url")
		}
	})

	t.Run("invalid extension", func(t *testing.T) {
		invalidUrl := messages.MultiModalMessage{
			Role: "user",
		}

		invalidUrl.AppendImageBytes([]byte{}, nil, "svg")

		if err := handleContent(&invalidUrl.Content[0]); err == nil {
			t.Errorf("invalid extenstion was accepted")
		}
	})

	t.Run("invalid extension", func(t *testing.T) {
		invalidUrl := messages.MultiModalMessage{
			Role: "user",
		}

		imageBytes := make([]byte, 10_000)
		invalidUrl.AppendImageBytes(imageBytes, nil, "png")

		if err := handleContent(&invalidUrl.Content[0]); err != nil {
			t.Error(err)
		}

		if !strings.HasPrefix(invalidUrl.Content[0].ImageUrl.Url, "data:image/") {
			t.Errorf("base64 string was never modified")
		}
	})

}
