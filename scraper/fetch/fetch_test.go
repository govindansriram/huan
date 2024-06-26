package fetch

import (
	"encoding/json"
	"fmt"
	"huan/llm/messages"
	scraper2 "huan/scraper"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func randomString(strLen int) string {
	str := []rune("abcdefghijklmnopqrstuvwxyz")

	builder := strings.Builder{}
	builder.Grow(strLen)

	for range strLen {
		index := rand.Intn(len(str))
		builder.WriteRune(str[index])
	}

	return builder.String()
}

func defRecover(t *testing.T, rec bool) {
	r := recover()
	if (r != nil) != rec {
		state := " not"
		if rec {
			state = ""
		}

		t.Errorf("failed to%s recover", state)
	}
}

func Test_splitStringByLen(t *testing.T) {

	tests := []struct {
		expectedSize int
		expectedCap  int
		split        uint
		name         string
		text         string
	}{
		{
			text:         randomString(100_000),
			split:        uint(100),
			expectedSize: 1000,
			expectedCap:  1000,
			name:         "no remainder in len(string) / split",
		},
		{
			text:         randomString(1_000_000),
			split:        uint(60_000),
			expectedSize: 17,
			expectedCap:  17,
			name:         "has remainder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strSlice := splitStringByLen(&tt.text, tt.split)

			if len(strSlice) != tt.expectedSize {
				t.Errorf(
					"expected a size of %d but got %d",
					tt.expectedSize,
					len(strSlice))
			}

			if cap(strSlice) != tt.expectedCap {
				t.Errorf(
					"expected a capacity of %d but got %d",
					tt.expectedSize,
					cap(strSlice))
			}

			for _, i := range strSlice {
				if len([]rune(*i)) > int(tt.split) {
					t.Errorf("chunk length exceeds split length")
				}
			}

			sum := 0
			for _, i := range strSlice {
				sum += len([]rune(*i))
			}

			if sum != len([]rune(tt.text)) {
				t.Errorf(
					"the correct amount of string charcters were not added, expected %d got %d",
					len([]rune(tt.text)),
					sum)
			}
		})
	}

	t.Run("split string with sequence longer than provided string", func(t *testing.T) {
		defer defRecover(t, true)
		data := randomString(10)
		splitStringByLen(&data, 20)
	})

	t.Run("split string with 0 len sequence", func(t *testing.T) {
		defer defRecover(t, true)
		data := randomString(10)
		splitStringByLen(&data, 0)
	})
}

func Test_splitStringIntoBuckets(t *testing.T) {
	tests := []struct {
		expectedLen int
		expectedCap int
		bucketLen   uint
		name        string
		text        string
	}{
		{
			text:        randomString(100_000),
			bucketLen:   100,
			expectedLen: 1000,
			expectedCap: 1000,
			name:        "no remainder in len(string) / split",
		},
		{
			text:        randomString(1_000_000),
			bucketLen:   uint(60_000),
			expectedLen: 16,
			expectedCap: 16,
			name:        "has remainder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitStringIntoBuckets(&tt.text, tt.bucketLen)

			if len(result) != int(tt.bucketLen) {
				t.Errorf("failed to split string into the requested amount of buckets")
			}

			for _, r := range result {
				if len([]rune(*r)) != tt.expectedLen && len([]rune(*r)) != tt.expectedLen+1 {
					t.Errorf("failed to split string into appropriate lengths")
				}
			}

			var total int

			for _, r := range result {
				total += len([]rune(*r))
			}

			if total != len([]rune(tt.text)) {
				t.Errorf("collected strings do not equal provided string")
			}
		})
	}

	t.Run("split string with 0 buckets", func(t *testing.T) {
		defer defRecover(t, true)
		data := randomString(10)
		splitStringIntoBuckets(&data, 0)
	})

	t.Run("less characters than buckets", func(t *testing.T) {
		defer defRecover(t, true)
		data := randomString(10)
		splitStringIntoBuckets(&data, 20)
	})

}

func Test_Collect(t *testing.T) {

	t.Run("test timeout scenario", func(t *testing.T) {

		tem := scraper2.TestModel{
			WorkTime: time.Duration(2),
			Template: map[string]interface{}{
				"content": "123",
			},
			Key: "content",
		}

		llm := scraper2.GetTestLanguageModel(tem)

		fetch := scraper2.Fetch{
			MaxRuntime: 1,
			Headless:   true,
			MaxSamples: 50,
			Task:       "asdasdasd",
			Urls: []string{
				"https://www.youtube.com/watch?v=ZX6-UWNsRnM",
			},
			ExampleTemplate: map[string]interface{}{
				"content": "123",
			},
			SavePath: os.TempDir(),
		}

		set := scraper2.Settings{
			Verbose:     false,
			SessionName: "temp",
		}

		cb := messages.ConversationBuilder{}

		logger := func(message string) {

		}

		n := time.Now()
		err := Collect(&llm, &fetch, &set, &cb, logger)

		if err == nil {
			err := os.Remove(filepath.Join(os.TempDir(), "temp-fetched.json"))
			if err != nil {
				t.Error(err)
			}
		} else {
			t.Error("errored for no reason")
		}

		elapsed := time.Now().Sub(n).Seconds()

		if elapsed > 10 {
			t.Errorf("failed")
		} else {
			fmt.Println(elapsed)
		}
	})

	t.Run("test completed scenario", func(t *testing.T) {

		tem := scraper2.TestModel{
			WorkTime: time.Duration(0),
			Template: map[string]interface{}{
				"content": "123",
			},
			Key:      "content",
			Quantity: 100,
		}

		llm := scraper2.GetTestLanguageModel(tem)

		fetch := scraper2.Fetch{
			MaxRuntime: 10,
			Headless:   true,
			MaxSamples: 100,
			Task:       "asdasdasd",
			Urls: []string{
				"https://www.youtube.com/watch?v=ZX6-UWNsRnM",
				"https://www.youtube.com/watch?v=ZX6-UWNsRnM",
			},
			ExampleTemplate: map[string]interface{}{
				"content": "123",
			},
			SavePath: os.TempDir(),
			Workers:  2,
		}

		set := scraper2.Settings{
			Verbose:     false,
			SessionName: "temp",
		}

		cb := messages.ConversationBuilder{}

		logger := func(message string) {

		}

		err := Collect(&llm, &fetch, &set, &cb, logger)

		if err == nil {
			pth := filepath.Join(os.TempDir(), "temp-fetched.json")
			file, err := os.ReadFile(pth)
			if err != nil {
				t.Error(err)
			}

			var data []interface{}

			err = json.Unmarshal(file, &data)

			if err != nil {
				t.Error(err)
			}

			err = os.Remove(filepath.Join(os.TempDir(), "temp-fetched.json"))
			if err != nil {
				t.Error(err)
			}

			if len(data) < 100 {
				t.Error("did not write data properly")
			}
		} else {
			t.Error("errored for no reason")
		}

	})

}
