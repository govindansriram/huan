package main

import (
	"agent/llm/messages"
	"agent/scraper"
	"agent/scraper/fetch"
	"fmt"
	"gopkg.in/yaml.v3"
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

func Start(s *scraper.Session) error {

	err, model := scraper.InitLanguageModel(
		s.LlmConfig.Type,
		s.LlmConfig.Settings,
		s.LlmConfig.TryLimit,
		s.LlmConfig.MaxTokens,
		s.LlmConfig.Duration)

	if err != nil {
		return err
	}

	err, sett := s.BuildSettings()

	if err != nil {
		return err
	}

	if s.Fetch != nil {
		err, fet := s.BuildFetchSettings()

		builder := &messages.ConversationBuilder{}

		err = fetch.Collect(model, fet, sett, builder)

		if err != nil {
			return err
		}
	}

	return err
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

	if err := Start(config); err != nil {
		fmt.Println(err)
	}
}
