package scraper

import (
	_ "embed"
	"fmt"
)

//go:embed prompts/collect.txt
var collectPrompt string

func loadCollectionPrompt(html, task, template string) string {
	data := fmt.Sprintf(collectPrompt, html, task, template)
	return data
}
