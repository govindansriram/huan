package scraper

import (
	"agent/llm/messages"
	"context"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

/*
Session

configuration of the scraping session
*/
type Session struct {
	Settings Setting `yaml:"settings"`
	LLM      LLM     `yaml:"llm"`
	Fetch    *Fetch  `yaml:"fetch"`
}

/*
Setting

general settings for the session
*/
type Setting struct {
	Verbose bool `yaml:"verbose"`
}

type setting struct {
	verbose bool
}

/*
build

creates a setting struct with predefined defaults
*/
func (s *Setting) build() (error, *setting) {
	return nil, &setting{
		verbose: s.Verbose,
	}
}

/*
LLM

the settings to initialize the llm with
*/
type LLM struct {
	Type      string                 `yaml:"type"`                 // what type of llm is being used eg: openai, llama
	Settings  map[string]interface{} `yaml:"settings"`             // llm specific settings
	TryLimit  *uint8                 `yaml:"tryLimit"`             // how many times to retry a rate limited request
	MaxTokens *uint16                `yaml:"maxTokens"`            // the max tokens the chatbot should return
	Duration  *uint16                `yaml:"requestDurationLimit"` // Max wait time for a chat completion to request
}

/*
languageModel

a wrapper struct around a llm with easy chat requests and defaults
*/
type languageModel struct {
	tryLimit uint8
	duration uint16
	engine   llm
}

type Fetch struct {
	MaxRuntime      *uint32                `yaml:"maxRuntime"`      // max time a data collection session can run in seconds
	Headless        bool                   `yaml:"headless"`        // whether the scraping session should be visible
	MaxSamples      *uint16                `yaml:"maxSamples"`      // the max amount of samples to collect
	Url             string                 `yaml:"url"`             // the url to collect data from
	Task            string                 `yaml:"task"`            // the data collection task that needs to be done (extra context)
	SavePath        *string                `yaml:"savePath"`        // where the data will be saved
	SaveRejected    bool                   `yaml:"saveRejected"`    // whether to save all the samples that could not be processed
	ExampleTemplate map[string]interface{} `yaml:"exampleTemplate"` // an example of how the data should be collected
}

type fetch struct {
	maxRuntime      uint32
	headless        bool
	maxSamples      uint16
	url             string
	task            string
	savePath        string
	saveRejected    bool
	exampleTemplate map[string]interface{}
}

func (f Fetch) build() (error, *fetch) {

	if f.MaxRuntime != nil && *f.MaxRuntime == 0 {
		return errors.New("fetch setting: maxRunTime cannot be 0"), nil
	}

	if f.MaxRuntime == nil {
		runTime := uint32(16)
		f.MaxRuntime = &runTime //run for 16 minutes
	}

	if f.MaxSamples != nil && *f.MaxSamples == 0 {
		return errors.New("fetch setting: maxSamples cannot be 0"), nil
	}

	if f.MaxSamples == nil {
		samp := uint16(1_000)
		f.MaxSamples = &samp
	}

	if f.Url == "" || strings.HasPrefix("http", f.Url) {
		return errors.New("fetch setting: url is invalid"), nil
	}

	if f.Task == "" {
		return errors.New("fetch setting: task is blank"), nil
	}

	if f.SavePath != nil {
		info, err := os.Stat(*f.SavePath)
		if err != nil {
			return err, nil
		}

		if !info.IsDir() {
			return fmt.Errorf("fetch setting savePath: %s is not a directory", *f.SavePath), nil
		}
	}

	if f.SavePath == nil {
		here := "."
		f.SavePath = &here
	}

	if f.ExampleTemplate == nil {
		return errors.New("fetch setting exampleTemplate: not provided"), nil
	} else {
		if len(f.ExampleTemplate) == 0 {
			return errors.New("fetch setting exampleTemplate: contains no keys"), nil
		}
	}

	return nil, &fetch{
		maxRuntime:      *f.MaxRuntime,
		headless:        f.Headless,
		maxSamples:      *f.MaxSamples,
		url:             f.Url,
		task:            f.Task,
		savePath:        *f.SavePath,
		saveRejected:    f.SaveRejected,
		exampleTemplate: f.ExampleTemplate,
	}
}

func (l languageModel) chat(ctx context.Context) (error, *messages.AssistantMessage) {
	return exponentialBackoff(ctx, l.engine, l.duration, l.tryLimit)
}

func (l LLM) build() (error, *languageModel) {
	var model llm
	var err error

	if l.Type == "" {
		return errors.New("a type must be provided in the llm section"), nil
	}

	llmConfig, err := yaml.Marshal(l.Settings)
	if err != nil {
		return err, nil
	}

	if l.MaxTokens == nil {
		maxTok := uint16(500)
		l.MaxTokens = &maxTok
	}

	switch strings.ToLower(l.Type) {
	case "openai":
		err, model = loadCGptFromYaml(llmConfig, *l.MaxTokens)
	default:
		return fmt.Errorf("%s, is not a valid llm type", l.Type), nil
	}

	if l.Duration == nil {
		duration := uint16(100)
		l.Duration = &duration
	}

	if l.TryLimit == nil {
		tryLimit := uint8(5)
		l.TryLimit = &tryLimit
	}

	return nil, &languageModel{
		engine:   model,
		duration: *l.Duration,
		tryLimit: *l.TryLimit,
	}
}

func (s *Session) Start() error {

	err, model := s.LLM.build()

	if err != nil {
		return err
	}

	err, sett := s.Settings.build()

	if err != nil {
		return err
	}

	if s.Fetch != nil {
		err, fet := s.Fetch.build()

		err = collect(model, fet, sett)

		if err != nil {
			return err
		}
	}

	return err
}
