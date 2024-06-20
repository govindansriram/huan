package main

import (
	"agent/scraper"
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

	err = config.Start()
}
