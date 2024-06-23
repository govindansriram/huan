/**
Key Terms:

snapshot: a snapshot is a name of a folder often passed in as a browser command argument.
It signifies what folder name you want to assign the data being collected too. It's useful because you can
group data collected at certain points of time together.


*/

package chrome

import (
	"context"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/css"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"huan/helper"
	"log"
	"time"
)

// parseThroughNodes
/*
iterates through nodes and returns structures to recollect them
*/
func parseThroughNodes(nodeSlice []*nodeWithStyles) []nodeMetaData {

	var nodeMetaDataSlice []nodeMetaData

	for _, nodeMd := range nodeSlice {
		attrMap := map[string]string{}

		for i := 0; i < len(nodeMd.node.Attributes); i += 2 {
			attrMap[nodeMd.node.Attributes[i]] += nodeMd.node.Attributes[i+1]
		}

		cssMap := make([]map[string]string, 0)

		for _, cascade := range nodeMd.cssStyles {
			tempMap := map[string]string{
				"name":  cascade.Name,
				"value": cascade.Value,
			}

			cssMap = append(cssMap, tempMap)
		}

		metaData := nodeMetaData{
			Id:         nodeMd.node.NodeID.Int64(),
			Type:       nodeMd.node.NodeType.String(),
			Xpath:      nodeMd.node.FullXPath(),
			Attributes: attrMap,
			CssStyles:  cssMap,
		}

		nodeMetaDataSlice = append(nodeMetaDataSlice, metaData)
	}

	return nodeMetaDataSlice
}

func flattenNode(nodeSlice []*cdp.Node) []*cdp.Node {
	deleteByIndex := helper.DeleteByIndex[*cdp.Node]

	var newNodeSlice []*cdp.Node

	for len(nodeSlice) > 0 {
		nodeSlice = append(nodeSlice, nodeSlice[0].Children...)
		newNodeSlice = append(newNodeSlice, nodeSlice[0])
		nodeSlice = deleteByIndex(nodeSlice, 0)
	}

	return newNodeSlice
}

// nodeMetaData
/*
carries all essential node data for web interaction
*/
type nodeMetaData struct {
	Id         int64               `json:"id"`
	Type       string              `json:"type"`
	Xpath      string              `json:"xpath"`
	Attributes map[string]string   `json:"attributes"`
	CssStyles  []map[string]string `json:"css_styles"`
}

// nodeWithStyles
/*
node with its correlating styles
*/
type nodeWithStyles struct {
	cssStyles []*css.ComputedStyleProperty
	node      *cdp.Node
}

// NavigateToUrl
/*
opens a webpage with the url provided
*/
func NavigateToUrl(
	url string) chromedp.ActionFunc {
	return func(c context.Context) error {
		err := chromedp.Navigate(url).Do(c)
		if err != nil {
			return err
		}
		return err
	}
}

// TakeFullPageScreenshot
/*
takes screenshots of the entire webpage on the browser

quality: how high resolution the image should be, higher is better

buffer: a pointer to a slice that will store the data
*/
func TakeFullPageScreenshot(
	quality uint8,
	buffer *[]byte) chromedp.ActionFunc {
	return func(c context.Context) error {
		err := chromedp.FullScreenshot(buffer, int(quality)).Do(c)
		if err != nil {
			return err
		}

		return err
	}
}

// TakeElementScreenshot
/*
takes a screenshot of a dom element

scale: the size of the screenshot bigger scale larger picture, the picture tends to get more pixelated as the
size increases

selector: the xpath of the html element

buffer: a byte array pointer that will contain the image bytes
*/
func TakeElementScreenshot(
	scale float64,
	selector string,
	buffer *[]byte) chromedp.ActionFunc {
	return func(c context.Context) error {
		err := chromedp.WaitVisible(selector).Do(c)
		if err != nil {
			return err
		}
		err = chromedp.ScreenshotScale(selector, scale, buffer, chromedp.NodeVisible).Do(c)
		if err != nil {
			return err
		}

		return err
	}
}

// ClickOnElement
/*
Instructs the chrome agent to click on a section of the webpage

selector: data representing the elements location in the dom eg: xpath, js path, tag name

queryFunc: chromedp selector representing the query type eg: byId, xpath etc.
*/
func ClickOnElement(
	selector string,
	queryFunc func(s *chromedp.Selector)) chromedp.ActionFunc {
	return func(c context.Context) error {
		err := chromedp.Click(selector, queryFunc).Do(c)
		return err
	}
}

// SleepForMs
/*
forces the browser to sleep for several ms amount of time
*/
func SleepForMs(ms uint64) chromedp.ActionFunc {
	return func(c context.Context) error {
		err := chromedp.Sleep(time.Duration(ms) * time.Millisecond).Do(c)
		if err != nil {
			log.Fatal(err)
		}

		return err
	}
}

// collectHtml
/*
collects the html present on the website
*/
func collectHtml(
	selector string,
	pString *string) chromedp.ActionFunc {
	return func(c context.Context) error {
		err := chromedp.OuterHTML(selector, pString).Do(c)
		if err != nil {
			log.Fatal(err)
		}

		return err
	}
}

// populateNode
/*
collects all node data

selector: collects only nodes belonging to this selector e.g: body

prepopulate: if true waits 1 second after requesting nodes (allows more to be collected)

recurse: if true uses bfs to collect all the child nodes of a parent node

collectStyles: if true collects all the css styles associated with a node

styledNodeList: the pointer that will hold all the data being collected
*/
func populatedNode(
	selector string,
	prepopulate bool,
	recurse bool,
	collectStyles bool,
	styledNodeList *[]*nodeWithStyles) chromedp.ActionFunc {
	return func(c context.Context) error {
		popSlice := make([]chromedp.PopulateOption, 0, 1)

		if prepopulate {
			popSlice = append(popSlice, chromedp.PopulateWait(1*time.Second))
		}

		var nodeSlice []*cdp.Node

		err := chromedp.Nodes(
			selector,
			&nodeSlice,
			chromedp.Populate(-1, true, popSlice...),
		).Do(c)

		if err != nil {
			return err
		}

		if recurse {
			nodeSlice = flattenNode(nodeSlice)
		}

		for _, node := range nodeSlice {

			var cs []*css.ComputedStyleProperty
			csErr := errors.New("failed")

			if collectStyles {
				cs, csErr = css.GetComputedStyleForNode(node.NodeID).Do(c)
			}

			styledNode := nodeWithStyles{
				node: node,
			}

			if csErr == nil {
				styledNode.cssStyles = cs
			}

			*styledNodeList = append(*styledNodeList, &styledNode)
		}

		return nil
	}
}

// scrollToPixel
/*
xPos the x coordinate location to scroll too
yPos the y coordinate location to scroll too

scrolls to certain location on the screen
*/
func scrollToPixel(xPos, yPos uint32) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		jsCode := fmt.Sprintf(`window.scrollTo(%d, %d);`, xPos, yPos)
		_, exp, err := runtime.Evaluate(jsCode).Do(ctx)

		if exp != nil {
			return exp
		}

		if err != nil {
			return err
		}
		return nil
	}
}

// scrollByPercentage
/*
percent the percent to scroll by

scrolls to certain location on the screen
*/
func scrollByPercentage(percent float32) (error, chromedp.ActionFunc) {

	if percent > 1.0 || percent <= 0 {
		return errors.New("percent must be greater than 0 and less than 1"), func(ctx context.Context) error {
			return errors.New("percent must be greater than 0 and less than 1")
		}
	}

	return nil, func(ctx context.Context) error {
		jsCode := fmt.Sprintf(`window.scrollTo(0, Math.floor(document.body.scrollHeight * %f));`, percent)
		_, exp, err := runtime.Evaluate(jsCode).Do(ctx)

		if exp != nil {
			return exp
		}

		if err != nil {
			return err
		}
		return nil
	}
}

//func (b *Executor) AcquireLocation(snapshot string) {
//	var loc string
//
//	b.appendTask(chromedp.ActionFunc(func(c context.Context) error {
//		err := chromedp.Location(&loc).Do(c)
//		if err != nil {
//			return err
//		}
//		return err
//	}))
//
//	b.locationMap[snapshot] = append(b.locationMap[snapshot], &loc)
//}
