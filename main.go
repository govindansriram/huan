package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"huan/llm/messages"
	"huan/scraper"
	"huan/scraper/fetch"
	"log"
	"os"
)

/*
buildFromYaml
convert bytes from yaml file, and convert it into Session
*/
func buildFromYaml(bytes []byte) (error, *scraper.Session) {
	config := &scraper.Session{}
	err := yaml.Unmarshal(bytes, config)
	return err, config
}

func Start(s *scraper.Session) {

	err, model := scraper.InitLanguageModel(
		s.LlmConfig.Type,
		s.LlmConfig.Settings,
		s.LlmConfig.TryLimit,
		s.LlmConfig.MaxTokens,
		s.LlmConfig.Duration,
		s.Settings.Verbose,
		s.LlmConfig.Workers)

	lg := func(message string) {
		if s.Settings.Verbose {
			log.Println(message)
		}
	}

	if err != nil {
		lg(fmt.Sprintf("could not initialize language model due to error: %v", err))
		return
	}

	err, sett := s.BuildSettings()

	if err != nil {
		lg(fmt.Sprintf("could not initialize settings model due to error: %v", err))
		return
	}

	if s.Fetch != nil {
		err, fet := s.BuildFetchSettings()

		if err != nil {
			lg(fmt.Sprintf("could not fetch settings due to error: %v", err))
			return
		}

		builder := &messages.ConversationBuilder{}

		err = fetch.Collect(model, fet, sett, builder, lg)

		if err != nil {
			lg(fmt.Sprintf("experienced error when writing data collection: %v", err))
			return
		} else {
			lg(fmt.Sprintf("session has ended gracefully"))
		}
	}
}

func main() {
	fPath := "./config.yaml"

	bytes, err := os.ReadFile(fPath)

	if err != nil {
		fmt.Println(err)
		return
	}

	err, config := buildFromYaml(bytes)

	if err != nil {
		fmt.Println(err)
		return
	}

	Start(config)
}
