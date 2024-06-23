package fetch

import (
	"agent/llm/messages"
	scraper2 "agent/scraper"
	"context"
	_ "embed"
	"fmt"
	"sync"
)

/*
TODO:
Rename stuff, ensure that if a request is too big it shrinks it recursively process json outputs
a map shoudl be appended too

add threads too yaml config
*/

//go:embed prompts/collect.txt
var collectPrompt string

func loadCollectionPrompt(html, task, template string) string {
	data := fmt.Sprintf(collectPrompt, html, task, template)
	return data
}

func processLoadCollectionPrompt(
	html,
	task,
	template string,
	builder *messages.ConversationBuilder) {

	prompt := loadCollectionPrompt(html, task, template)
	mess := messages.StandardMessage{
		Role:    "user",
		Content: prompt,
	}

	builder.AddStandardMessage(&mess)
}

/*
promptPool

ensures that multiple chat completion requests happen concurrently
*/
func promptPool(
	threadCount uint8,
	task,
	template string,
	llm *scraper2.LanguageModel,
	ctx context.Context,
	builder *messages.ConversationBuilder,
	strs []*string) {

	type chatResult struct {
		err      error
		response string
	}

	channel := make(chan chatResult)               // the channel that will contain teh results of each request
	workerPool := make(chan struct{}, threadCount) // limits how many requests can happen at the same time

	wg := sync.WaitGroup{}

	/*
		schedules all the chat requests that need to happen concurrently
	*/
	processLoadCollectionPrompt(*strs[0], task, template, builder)

	for _, str := range strs {
		/*
			a goroutine that completes a chat request
		*/
		builder.Pop(builder.Size() - 1)
		processLoadCollectionPrompt(*str, task, template, builder)

		err := llm.Validate(builder)
		if err != nil {
			panic(err)
		}

		err, convo := builder.Build()
		if err != nil {
			panic(err)
		}

		wg.Add(1)
		go func() {
			workerPool <- struct{}{}             // signal to the worker pool that, work is being done, blocking it once the buffer is full
			err, result := llm.Chat(ctx, &convo) // start the request

			var response string

			if err == nil {
				response = *result.Content
			}

			channel <- chatResult{
				err:      err,
				response: response,
			}
			wg.Done()
		}()
	}

	/*
		closes the request channel once all go routines are done
	*/
	go func() {
		wg.Wait()
		close(channel)
	}()

	for chatRes := range channel {
		<-workerPool // free a space in the worker pool
		if chatRes.err == nil {
			fmt.Println(chatRes.response)
		}
	}
}
