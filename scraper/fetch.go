package scraper

import (
	"agent/llm/messages"
	"context"
	"errors"
	"github.com/chromedp/chromedp"
	"log"
	"time"
)

func initContext(headless bool, timeout uint32) (context.Context, context.CancelFunc) {
	var ctx context.Context
	var cancel context.CancelFunc

	if headless {
		ctx, cancel = chromedp.NewContext(
			context.Background(),
		)
	} else {
		actx, _ := chromedp.NewExecAllocator(
			context.Background(),
			append(
				chromedp.DefaultExecAllocatorOptions[:],
				chromedp.Flag("headless", false))...)

		ctx, cancel = chromedp.NewContext(
			actx,
		)
	}

	ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)

	return ctx, cancel
}

func collectHtml(pString *string) chromedp.ActionFunc {
	return func(c context.Context) error {
		return chromedp.OuterHTML("body", pString).Do(c)
	}
}

func collectScreenshot(buffer *[]byte, quality uint8) (chromedp.ActionFunc, string) {

	if quality > 100 {
		panic("screenshot quality cannot exceed 100")
	}

	if quality == 0 {
		panic("screenshot quality cannot be 0")
	}

	ext := func() string {
		if quality < 100 {
			return "jpeg"
		}

		return "png"
	}()

	return func(c context.Context) error {
		err := chromedp.FullScreenshot(buffer, int(quality)).Do(c)
		if err != nil {
			return err
		}

		return err
	}, ext
}

func collectContext(pString *string, pBuffer *[]byte, ctx context.Context) (error, string) {
	err := collectHtml(pString).Do(ctx)
	if err != nil {
		return err, ""
	}

	function, ext := collectScreenshot(pBuffer, 100)

	err = function.Do(ctx)
	if err != nil {
		return err, ""
	}

	return err, ext
}

func addVisualContext(llm *languageModel, pImageBuffer *[]byte, ext string) {
	mess := messages.StandardMessage{
		Role:    "user",
		Content: `the following image is the webpage that needs to be scraped`,
	}

	err := llm.engine.AppendStandard(&mess)

	if err != nil {
		panic(err)
	}

	err = llm.engine.AppendMultiModal(*pImageBuffer, "user", nil, ext)

	if err != nil {
		panic(err)
	}

	return
}

func scraper(
	url string,
	samples *[]map[string]interface{},
	capacity uint16,
	model *languageModel,
	logger func(message string)) chromedp.ActionFunc {

	return func(c context.Context) error {
		err := chromedp.Navigate(url).Do(c)
		alive := true
		err = chromedp.Sleep(time.Second * 5).Do(c)

		if err != nil {
			return err
		}

		system := messages.StandardMessage{
			Role:    "system",
			Content: "you are an expert webscraper specialized in collecting html data",
		}

		err = model.engine.AppendStandard(&system)

		if err != nil {
			panic(err)
		}

		for alive && (capacity > uint16(len(*samples))) {
			var htmlData string
			var imageBuffer []byte

			err, ext := collectContext(&htmlData, &imageBuffer, c)
			if err != nil {
				return errors.New("failed to collect website data such as background or html")
			}

			addVisualContext(model, &imageBuffer, ext)

		}

		return err
	}

}

func splitStringByLen(pStr *string, strLen uint) []*string {
	var chunks []*string

	if strLen == 0 {
		panic("strLen cannot be 0")
	}

	str := []rune(*pStr)

	if uint(len(str)) < strLen {
		panic("the length of the str is less then strlen")
	}

	hasRemainder := (len(str) % int(strLen)) != 0
	capacity := len(str) / int(strLen)

	if hasRemainder {
		capacity++
	}

	chunks = make([]*string, 0, capacity)

	for index := range capacity {
		startPos := uint(index) * strLen

		var strLength int
		var endPos int

		if index == capacity-1 {
			strLength = len(str) - int(startPos)
			endPos = len(str)
		} else {
			strLength = int(strLen)
			endPos = int(startPos + strLen)
		}

		subsection := make([]rune, strLength)
		copy(subsection, str[startPos:endPos])

		result := string(subsection)
		chunks = append(chunks, &result)
	}

	return chunks
}

func splitStringIntoBuckets(pStr *string, bucketCount uint) []*string {
	if bucketCount == 0 {
		panic("bucketCount must be greater than 0")
	}

	runes := []rune(*pStr)

	if uint(len(runes)) < bucketCount {
		panic("the length of the str is less then teh amount of requested buckets")
	}

	bucketLength := uint(len(runes)) / bucketCount
	bucketRemainder := uint(len(runes)) % bucketCount

	limits := make([]uint, bucketCount)

	for index := range limits {
		limits[index] = bucketLength
	}

	for index := range bucketRemainder {
		limits[index]++
	}

	pStrSlice := make([]*string, bucketCount)

	var start int
	for index, limit := range limits {
		end := start + int(limit)
		subsection := make([]rune, end-start)
		copy(subsection, runes[start:end])
		result := string(subsection)
		pStrSlice[index] = &result
		start = end
	}

	return pStrSlice
}

func collect(llm *languageModel, fetchSettings *fetch, set *setting) error {
	parentContext, cancel := initContext(fetchSettings.headless, fetchSettings.maxRuntime)
	defer cancel()

	sampleSlice := make([]map[string]interface{}, 0, fetchSettings.maxSamples)

	lg := func(message string) {
		if set.verbose {
			log.Println(message)
		}
	}

	scraperAction := scraper(fetchSettings.url, &sampleSlice, fetchSettings.maxSamples, llm, lg)

	lg("fetching data ...")
	err := chromedp.Run(parentContext, scraperAction)
	return err
}
