package scraper

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"os"
	"strings"
)

/*
Session

configuration of the scraping session
*/
type Session struct {
	Settings struct {
		Verbose     bool    `yaml:"verbose"`
		SessionName *string `yaml:"sessionName"`
	} `yaml:"settings"`

	LlmConfig struct {
		Type      string                 `yaml:"type"`            // what type of llm is being used eg: openai, llama
		Settings  map[string]interface{} `yaml:"settings"`        // llm specific settings
		TryLimit  *uint8                 `yaml:"tryLimit"`        // how many times to retry a rate limited request
		MaxTokens *uint16                `yaml:"maxTokens"`       // the max tokens the chatbot should return
		Duration  *uint16                `yaml:"requestDuration"` // Max wait time for a chat completion to request
	} `yaml:"llmConfig"`

	Fetch *struct {
		MaxRuntime      *uint32                `yaml:"maxRuntime"`      // max time a data collection session can run in seconds
		Headless        bool                   `yaml:"headless"`        // whether the scraping session should be visible
		MaxSamples      *uint16                `yaml:"maxSamples"`      // the max amount of samples to collect
		Url             string                 `yaml:"url"`             // the url to collect data from
		Task            string                 `yaml:"task"`            // the data collection task that needs to be done (extra context)
		SavePath        *string                `yaml:"savePath"`        // where the data will be saved
		ExampleTemplate map[string]interface{} `yaml:"exampleTemplate"` // an example of how the data should be collected
	} `yaml:"fetch"`
}

type Settings struct {
	Verbose     bool
	SessionName string
}

/*
BuildSettings

creates a settings struct with predefined defaults
*/
func (s *Session) BuildSettings() (error, *Settings) {
	var sessionName string

	if s.Settings.SessionName == nil {
		sessionName = uuid.New().String()
	} else {
		sessionName = *s.Settings.SessionName
	}

	return nil, &Settings{
		Verbose:     s.Settings.Verbose,
		SessionName: sessionName,
	}
}

type Fetch struct {
	MaxRuntime      uint32
	Headless        bool
	MaxSamples      uint16
	Url             string
	Task            string
	SavePath        string
	ExampleTemplate map[string]interface{}
}

func (s *Session) BuildFetchSettings() (error, *Fetch) {

	if s.Fetch.MaxRuntime != nil && *s.Fetch.MaxRuntime == 0 {
		return errors.New("the Fetch setting: maxRunTime cannot be 0"), nil
	}

	if s.Fetch.MaxRuntime == nil {
		runTime := uint32(16)
		s.Fetch.MaxRuntime = &runTime //run for 16 minutes
	}

	if s.Fetch.MaxSamples != nil && *s.Fetch.MaxSamples == 0 {
		return errors.New("the Fetch setting: maxSamples cannot be 0"), nil
	}

	if s.Fetch.MaxSamples == nil {
		samp := uint16(1_000)
		s.Fetch.MaxSamples = &samp
	}

	if s.Fetch.Url == "" || strings.HasPrefix("http", s.Fetch.Url) {
		return errors.New("the Fetch setting: url is invalid"), nil
	}

	if s.Fetch.Task == "" {
		return errors.New("the Fetch setting: task is blank"), nil
	}

	if s.Fetch.SavePath != nil {
		info, err := os.Stat(*s.Fetch.SavePath)
		if err != nil {
			return err, nil
		}

		if !info.IsDir() {
			return fmt.Errorf("the Fetch settings savePath: %s is not a directory", *s.Fetch.SavePath), nil
		}
	}

	if s.Fetch.SavePath == nil {
		here := "."
		s.Fetch.SavePath = &here
	}

	if s.Fetch.ExampleTemplate == nil {
		return errors.New("the Fetch setting exampleTemplate: not provided"), nil
	} else {
		if len(s.Fetch.ExampleTemplate) == 0 {
			return errors.New("the Fetch settings exampleTemplate: contains no keys"), nil
		}
	}

	return nil, &Fetch{
		MaxRuntime:      *s.Fetch.MaxRuntime,
		Headless:        s.Fetch.Headless,
		MaxSamples:      *s.Fetch.MaxSamples,
		Url:             s.Fetch.Url,
		Task:            s.Fetch.Task,
		SavePath:        *s.Fetch.SavePath,
		ExampleTemplate: s.Fetch.ExampleTemplate,
	}
}
