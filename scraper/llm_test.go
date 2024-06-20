package scraper

import (
	"agent/llm/messages"
	"context"
	"errors"
	"testing"
	"time"
)

type fakeLLM struct {
	fail bool
	wait bool
	slp  int
}

func (f *fakeLLM) Chat(ctx context.Context) (error, *bool, *messages.ChatCompletion) {
	c := make(chan struct{})

	go func() {
		time.Sleep(time.Duration(f.slp) * time.Second)
		close(c)
	}()

	select {
	case <-c:
		var wait bool
		if f.wait {
			wait = true
			return errors.New("rate limit hit"), &wait, nil
		}

		if f.fail {
			return errors.New("failed due to ___"), &wait, nil
		}

		return nil, nil, &messages.ChatCompletion{
			Choices: make([]messages.Choice, 5),
		}
	case <-ctx.Done():
		return errors.New("cancelling operation due to deadline"), nil, nil
	}
}

func (f *fakeLLM) AppendAssistant(message *messages.AssistantMessage) error {
	return nil
}
func (f *fakeLLM) AppendStandard(message *messages.StandardMessage) error {
	return nil
}
func (f *fakeLLM) AppendMultiModal(imageBytes []byte, role string, detail *string, imageType string) error {
	return nil
}
func (f *fakeLLM) Pop(index uint) {

}

func Test_exponentialBackoff(t *testing.T) {

	timeCode := func(
		sleepTime,
		wait,
		tryLimit,
		globalWait uint8,
		didFail,
		didWait,
		isErr bool,
		t *testing.T,
		f func(sleepTime, wait, tryLimit, globalWait uint8, didFail, didWait, isErr bool, t *testing.T)) float64 {
		start := time.Now()
		f(sleepTime, wait, tryLimit, globalWait, didFail, didWait, isErr, t)
		elapsed := time.Now().Sub(start).Seconds()
		return elapsed
	}

	test1 := func(sleepTime, wait, tryLimit, globalWait uint8, didFail, didWait, isErr bool, t *testing.T) {
		parentContext, can := context.WithTimeout(context.Background(), time.Second*time.Duration(globalWait))
		defer can()
		err, _ := exponentialBackoff(
			parentContext,
			&fakeLLM{slp: int(sleepTime), fail: didFail, wait: didWait},
			uint16(wait),
			tryLimit)

		if (err == nil) == isErr {
			expected := func(state bool) string {
				if state {
					return "nil"
				}
				return "error"
			}
			t.Errorf(
				"expected TestImageContent_Validate to return an %s but got %s",
				expected(isErr),
				expected(!isErr))
		}
	}

	tests := []struct {
		sleepTime   uint8
		wait        uint8
		tryLimit    uint8
		globalWait  uint8
		didFail     bool
		didWait     bool
		isErr       bool
		name        string
		elapsedTime float64
	}{
		{
			sleepTime:   1,
			wait:        2,
			tryLimit:    3,
			globalWait:  10,
			didFail:     false,
			didWait:     false,
			isErr:       false,
			name:        "passed",
			elapsedTime: 1,
		},
		{
			sleepTime:   1,
			wait:        2,
			tryLimit:    3,
			globalWait:  3,
			didFail:     true,
			didWait:     true,
			isErr:       true,
			name:        "hits global wait",
			elapsedTime: 3,
		},
		{
			sleepTime:   1,
			wait:        2,
			tryLimit:    3,
			globalWait:  20,
			didFail:     true,
			didWait:     true,
			isErr:       true,
			name:        "hits try limit",
			elapsedTime: 5,
		},
		{
			sleepTime:   1,
			wait:        0,
			tryLimit:    3,
			globalWait:  20,
			didFail:     true,
			didWait:     true,
			isErr:       true,
			name:        "hits local limit",
			elapsedTime: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tm := timeCode(tt.sleepTime, tt.wait, tt.tryLimit, tt.globalWait, tt.didFail, tt.didWait, tt.isErr, t, test1)
			if tm < tt.elapsedTime {

				t.Errorf(
					"expected func to run for minimum of %e but got %e",
					tt.elapsedTime, tm)
			}
		})
	}
}
