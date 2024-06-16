package scraper

import (
	"agent/helper"
	"context"
	"github.com/chromedp/chromedp"
)

/*
startLiveSession
starts a live session in a chromedp action loop
*/

func SessionFunc(
	template map[string]interface{},
) chromedp.ActionFunc {

	return func(c context.Context) error {
		commandSet := helper.Set[string]{}
		alive := true

		var exitErr error

		go func() {
			// this function runs as a background process, it waits for the context to exceed and
			//signals to the while loop that the session has ended
			<-c.Done()
			alive = false
		}()

		for alive {
			if commandSlice, err := collectCommandFiles(sessionPath, commandSet); err == nil {
				for _, commandFileName := range commandSlice {
					exitErr = processOperations(commandFileName, c, sessionPath, waitTime)

					if exitErr != nil {
						alive = false
					}

					commandSet.Insert(commandFileName)
				}
			} else {
				alive = false
				exitErr = err
			}
		}

		if exitErr == nil {
			// there is no scenario where the session just exits on its own. It's either an internal error,
			// an error caused by faulty commands,
			// or the default in this case a timeout
			exitErr = context.DeadlineExceeded
		}

		err := endSession(sessionPath, exitErr)
		return err
	}
}
