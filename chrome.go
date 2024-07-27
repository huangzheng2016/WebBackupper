package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"golang.org/x/net/html"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var dataDir = "data"

func downloadWebPage(url string) {
	ctx, cancel := chromedp.NewContext(context.Background())
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var requestMap = make(map[string]network.RequestID)
	chromedp.ListenTarget(ctx, func(v interface{}) {
		switch ev := v.(type) {
		case *network.EventRequestWillBeSent:
			requestMap[ev.Request.URL] = ev.RequestID
			fmt.Println(ev.Request.URL)
		}
	})
	var htmlContent string
	err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.EmulateViewport(2440, 1920),
		chromedp.Navigate(url),
		chromedp.OuterHTML("html", &htmlContent, chromedp.ByQuery),
	)
	htmlContent = replaceHtml(ctx, htmlContent)
	_ = os.WriteFile("website.html", []byte(htmlContent), 0644)
	if err != nil {
		log.Println(err)
		return
	}
}

func getFileExt(url string) string {
	return filepath.Ext(strings.Split(url, "?")[0])
}
func saveFile(ctx context.Context, fileName string, fileBuf []byte, requestID network.RequestID) string {
	if fileBuf == nil {
		if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			fileBuf, err = network.GetResponseBody(requestID).Do(ctx)
			return err
		})); err != nil {
			log.Println(err)
			return ""
		}
	}
	hash := sha256.New()
	hash.Write(fileBuf)
	fileName = fmt.Sprintf("%x", hash.Sum(nil))[:32] + getFileExt(fileName)
	dir := filepath.Join(dataDir, fileName[0:2], fileName[2:4])
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.MkdirAll(dir, 0755)
	}
	filePath := filepath.Join(dir, fileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		_ = os.WriteFile(filePath, fileBuf, 0644)
	}
	return filePath
}

func searchNode(ctx context.Context, n *html.Node) {
	if n.Type == html.ElementNode && n.Data == "img" {
		for _, attr := range n.Attr {
			if attr.Key == "src" {
				fmt.Println(attr.Val)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		searchNode(ctx, c)
	}
}

func replaceHtml(ctx context.Context, htmlContent string) string {
	root, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		log.Fatal(err)
	}
	searchNode(ctx, root)
	var buf bytes.Buffer
	if err := html.Render(&buf, root); err != nil {
		log.Fatal(err)
	}
	return buf.String()
}
func MustParse(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

/*
func searchNode(ctx context.Context, baseUrl string, nodes []*cdp.Node, requestMap map[string]network.RequestID) {
	var evalQueue []string
	base, err := url.Parse(baseUrl)
	if err != nil {
		return
	}
	for _, node := range nodes {
		if node.NodeName == "IMG" || node.NodeName == "SCRIPT" || node.NodeName == "LINK" || node.NodeName == "STYLE" {
			nodeUrl := node.AttributeValue("src")
			if nodeUrl == "" {
				nodeUrl = node.AttributeValue("href")
			}
			var filePath string
			if nodeUrl == "" {
				for _, child := range node.Children {
					if child.NodeName == "#text" {
						filePath = saveFile(ctx, "", []byte(child.NodeValue), "")
						var js string
						switch node.NodeName {
						case "SCRIPT":
							js = "element.innerText='';element.src='%s';"
						case "STYLE":
							js = "element.innerText='';element.innerText='@import url(\\'%s\\');';"
						}
						js = fmt.Sprintf(js, filePath)
						js = fmt.Sprintf(`document.querySelectorAll('*').forEach(function(element){if(element.innerText==%s){%s}});`, strconv.Quote(child.NodeValue), js)
						evalQueue = append(evalQueue, js)
						break
					}
				}
			} else {
				absUrl := nodeUrl
				if !strings.HasPrefix(absUrl, "http") {
					absoluteUrl := base.ResolveReference(MustParse(absUrl))
					absUrl = absoluteUrl.String()
				}
				if requestId, exist := requestMap[absUrl]; exist {
					filePath = saveFile(ctx, absUrl, nil, requestId)
					//fmt.Println(nodeUrl, absUrl, requestId, filePath)
					if node.AttributeValue("src") != "" {
						evalQueue = append(evalQueue, fmt.Sprintf(`document.querySelectorAll('[src="%s"]').forEach(function(element){element.src="%s";});`, nodeUrl, filePath))
					} else {
						evalQueue = append(evalQueue, fmt.Sprintf(`document.querySelectorAll('[href="%s"]').forEach(function(element){element.href="%s";});`, nodeUrl, filePath))
					}
				}
			}
		}
	}
	network.Disable()
	for _, js := range evalQueue {
		_ = chromedp.Evaluate(js, nil).Do(ctx)
	}
}
*/
