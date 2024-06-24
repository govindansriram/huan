package jsonparser

import (
	"encoding/json"
	"errors"
	"huan/helper"
	"strings"
)

/*
check if a string starts with "```json" & ends with "```"
- if true trim both

check if a string starts with '[' and ends with ']'
- if true check if json is valid
- else search for maps and see if you can extract them by doing a bracket stack
*/

/*
RemoveIdentifier

remove ```json & ``` if it is present in the string
*/
func RemoveIdentifier(json string) string {
	json = strings.TrimPrefix(json, "```json")
	json = strings.TrimSuffix(json, "```")
	json = strings.TrimRight(json, " ")
	json = strings.TrimLeft(json, " ")

	return json
}

/*
AttemptConversion

Attempts to convert json string to a valid json array
*/
func AttemptConversion(data string) (error, []map[string]interface{}) {
	dataBytes := []byte(data)

	var retSlice []map[string]interface{}

	if strings.HasPrefix(data, "[") && strings.HasSuffix(data, "]") {
		err := json.Unmarshal(dataBytes, &retSlice)
		if err != nil {
			return err, nil
		}

		return nil, retSlice
	} else if strings.HasPrefix(data, "{") && strings.HasSuffix(data, "}") {
		var retMap map[string]interface{}
		err := json.Unmarshal(dataBytes, &retMap)

		if err != nil {
			return err, nil
		}

		retSlice = append(retSlice, retMap)

		return nil, retSlice
	}

	return errors.New("failed to convert string"), nil
}

/*
ToJson

Converts a string to json data if possible
*/
func ToJson(data string) []map[string]interface{} {
	dataSlice := make([]map[string]interface{}, 0, 10)

	data = RemoveIdentifier(data)
	err, dataSlice := AttemptConversion(data)

	if err == nil {
		return dataSlice
	}

	bracketSlice := make([]struct{}, 0, 30)
	pop := helper.DeleteByIndex[struct{}]

	var inSequence bool
	var builder strings.Builder

	builder.Grow(200)

	for _, roon := range data {

		if inSequence {
			builder.WriteRune(roon)
		}

		if roon == '{' {
			bracketSlice = append(bracketSlice, struct{}{})

			if !inSequence {
				inSequence = true
				builder.WriteRune(roon)
			}
		}

		if roon == '}' {
			if len(bracketSlice) >= 1 {
				bracketSlice = pop(bracketSlice, uint(len(bracketSlice)-1))
				if len(bracketSlice) == 0 {
					val := builder.String()
					builder.Reset()
					inSequence = false
					builder.Grow(200)

					var sample map[string]interface{}

					if err := json.Unmarshal([]byte(val), &sample); err == nil {
						dataSlice = append(dataSlice, sample)
					}
				}
			}
		}
	}

	return dataSlice
}
