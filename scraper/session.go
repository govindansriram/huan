package scraper

import (
	"encoding/json"
	"errors"
)

/*
Session

settings to guide the scraping session
*/
type Session struct {
	maxRunTime uint32                 // max time a data collection session can run in ms
	headless   bool                   // whether the scraping session should be visible
	tryLimit   uint8                  // how many times to retry a rate limited chat request
	maxTokens  uint16                 // the max tokens the chatbot should return
	modelName  string                 // the name of the model being used
	maxData    uint16                 // the max amount of data points that can be collected
	template   map[string]interface{} // how the data should be formatted
}

/*
SessionBuilder

helper struct useful for validating settings, and handling optional parameters
*/
type SessionBuilder struct {
	maxRunTime *uint32
	tryLimit   *uint8
	maxTokens  *uint16
	maxData    *uint16
}

func (b *SessionBuilder) AddMaxRunTime(maxRunTime uint32) *SessionBuilder {
	b.maxRunTime = &maxRunTime
	return b
}

func (b *SessionBuilder) SetTryLimit(tryLimit uint8) *SessionBuilder {
	b.tryLimit = &tryLimit
	return b
}

func (b *SessionBuilder) SetMaxTokens(maxTokens uint16) *SessionBuilder {
	b.maxTokens = &maxTokens
	return b
}

func (b *SessionBuilder) SetMaxData(maxData uint16) *SessionBuilder {
	b.maxData = &maxData
	return b
}

func (b *SessionBuilder) Build(headless bool, modelName string, template string) (error, *Session) {
	if b.maxRunTime == nil {
		maxTime := uint32(60 * 5 * 1000)
		b.maxRunTime = &maxTime
	}

	if *b.maxRunTime == 0 {
		return errors.New("maxRunTime must be at least 1ms"), nil
	}

	if b.tryLimit == nil {
		tryLimit := uint8(1)
		b.tryLimit = &tryLimit
	}

	if *b.tryLimit == 0 {
		return errors.New("tryLimit must be at least 1"), nil
	}

	if b.maxTokens == nil {
		maxTok := uint16(400)
		b.maxTokens = &maxTok
	}

	if *b.maxTokens == 0 {
		return errors.New("maxTokens must be at least 1"), nil
	}

	if b.maxData == nil {
		maxData := uint16(400)
		b.maxData = &maxData
	}

	if *b.maxData == 0 {
		return errors.New("maxData must be at least 1"), nil
	}

	byteTemplate := []byte(template)

	structure := make(map[string]interface{}, 10)

	err := json.Unmarshal(byteTemplate, &structure)

	if err != nil {
		return err, nil
	}

	return nil, &Session{
		maxRunTime: *b.maxRunTime,
		headless:   headless,
		modelName:  modelName,
		maxTokens:  *b.maxTokens,
		tryLimit:   *b.tryLimit,
		maxData:    *b.maxData,
		template:   structure,
	}
}
