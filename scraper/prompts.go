package scraper

import (
	"agent/llm/messages"
	"context"
	_ "embed"
	"fmt"
	"sync"
)

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
	llm *languageModel,
	ctx context.Context) (error, string) {

	prompt := loadCollectionPrompt(html, task, template)
	mess := messages.StandardMessage{
		Role:    "user",
		Content: prompt,
	}

	fmt.Printf("the address is %p \n", &(llm.engine))

	if err := llm.engine.AppendStandard(&mess); err != nil {
		return err, ""
	}

	err, comp := llm.chat(ctx)

	if err != nil {
		return err, ""
	}

	return nil, *comp.Content
}

/*
promptPool

ensures that multiple chat completion requests happen concurrently
*/
func promptPool(
	threadCount uint8,
	promptChat func(str string) (error, string),
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
	for _, str := range strs {
		wg.Add(1)

		/*
			a goroutine that completes a chat request
		*/
		go func() {
			workerPool <- struct{}{}        // signal to the worker pool that, work is being done, blocking it once the buffer is full
			err, result := promptChat(*str) // start the request
			channel <- chatResult{
				err:      err,
				response: result,
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

func runCollectionPool(
	task,
	template string,
	lm *languageModel,
	ctx context.Context,
	threadCount uint8,
	htmlContext []*string) {

	chat := func(str string) (error, string) {
		modelCopy := lm.DeepCopy()
		fmt.Printf("the oriignal address is %p \n", &(lm.engine))
		fmt.Printf("the copy address is %p \n", &(modelCopy.engine))
		return processLoadCollectionPrompt(
			str,
			task,
			template,
			modelCopy,
			ctx)
	}

	promptPool(threadCount, chat, htmlContext)
}
