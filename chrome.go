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
	"regexp"
	"strings"
	"time"
)

var dataDir = "data"

func downloadWebPage(downloadUrl string) {
	options := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", false),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("enable-use-zoom-for-dsf", "false"),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-field-trial-config", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("enable-features", "NetworkService,NetworkServiceInProcess"),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-back-forward-cache", true),
		chromedp.Flag("disable-breakpad", true),
		chromedp.Flag("disable-client-side-phishing-detection", true),
		chromedp.Flag("disable-component-extensions-with-background-pages", true),
		chromedp.Flag("disable-component-update", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-features", "ImprovedCookieControls,LazyFrameLoading,GlobalMediaControls,DestroyProfileOnBrowserClose,MediaRouter,DialMediaRouteProvider,AcceptCHFrame,AutoExpandDetailsElement,CertificateTransparencyComponentUpdater,AvoidUnnecessaryBeforeUnloadCheckSync,Translate,HttpsUpgrades,PaintHolding"),
		chromedp.Flag("disable-hang-monitor", true),
		chromedp.Flag("disable-ipc-flooding-protection", true),
		chromedp.Flag("disable-popup-blocking", true),
		chromedp.Flag("disable-prompt-on-repost", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("force-color-profile", "srgb"),
		chromedp.Flag("enable-automation", true),
		chromedp.Flag("password-store", "basic"),
		chromedp.Flag("use-mock-keychain", true),
		chromedp.Flag("no-service-autorun", true),
		chromedp.Flag("disable-search-engine-choice-screen", true),
		chromedp.Flag("bwsi", true),
	}
	ctx, cancel := chromedp.NewExecAllocator(context.Background(), options...)
	ctx, cancel = chromedp.NewContext(ctx)
	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var requestMap = make(map[string]network.RequestID)
	chromedp.ListenTarget(ctx, func(v interface{}) {
		switch ev := v.(type) {
		case *network.EventRequestWillBeSent:
			requestMap[ev.Request.URL] = ev.RequestID
			//fmt.Println(ev.Request.URL)
		}
	})
	var htmlContent string
	err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.EmulateViewport(1920, 1080),
		chromedp.Navigate(downloadUrl),
		chromedp.OuterHTML("html", &htmlContent, chromedp.ByQuery),
		chromedp.Location(&downloadUrl),
	)
	baseUrl, err := url.Parse(downloadUrl)
	if err != nil {
		log.Println(err)
		return
	}
	htmlContent = replaceHtml(ctx, baseUrl, requestMap, htmlContent) + cacheContent(downloadUrl)
	_ = os.WriteFile("website.html", []byte(htmlContent), 0644)
}
func cacheContent(url string) string {
	return fmt.Sprintf("\n<!-- Cached %s at %s -->", url, time.Now().Format("2006-01-02 15:04:05"))
}

func getFileExt(url string) string {
	return filepath.Ext(strings.Split(url, "?")[0])
}
func saveFile(ctx context.Context, fileName string, fileBuf []byte, requestId network.RequestID, requestMap map[string]network.RequestID, baseUrl *url.URL, types string) string {
	if fileBuf == nil {
		if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			fileBuf, err = network.GetResponseBody(requestId).Do(ctx)
			return err
		})); err != nil {
			log.Println(err)
			return ""
		}
	}
	ext := getFileExt(fileName)
	if types == "link" || ext == ".css" {
		re := regexp.MustCompile(`url\(\s*["']?([^"')]+)`)
		replacedCSS := re.ReplaceAllStringFunc(string(fileBuf), func(s string) string {
			sub := re.FindStringSubmatch(s)
			for _, oldUrl := range sub {
				absUrl := oldUrl
				if !strings.HasPrefix(absUrl, "http") {
					absUrlParsed, err := url.Parse(absUrl)
					if err != nil {
						continue
					}
					absoluteUrl := baseUrl.ResolveReference(absUrlParsed)
					absUrl = absoluteUrl.String()
				}
				if newRequestId, exist := requestMap[absUrl]; exist {
					newUrl := saveFile(ctx, absUrl, nil, newRequestId, requestMap, baseUrl, "")
					//fmt.Println(oldUrl, newUrl)
					s = strings.Replace(s, oldUrl, "../../../"+newUrl, -1)
				}
			}
			return s
		})
		fileBuf = []byte(replacedCSS)
	}
	hash := sha256.New()
	hash.Write(fileBuf)
	fileName = fmt.Sprintf("%x", hash.Sum(nil))[:32] + ext
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

var replaceArray = []string{"img", "script", "link", "style"}
var srcArray = []string{"script"}
var hrefArray = []string{"link"}
var importArray = []string{"style"}

func searchNode(ctx context.Context, baseUrl *url.URL, requestMap map[string]network.RequestID, n *html.Node) {
	if n.Type == html.ElementNode && inArray(replaceArray, n.Data) {
		for i := 0; i < len(n.Attr); i++ {
			attr := &n.Attr[i]
			if attr.Key == "src" || attr.Key == "href" {
				absUrl := attr.Val
				if !strings.HasPrefix(absUrl, "http") {
					absUrlParsed, err := url.Parse(absUrl)
					if err != nil {
						return
					}
					absoluteUrl := baseUrl.ResolveReference(absUrlParsed)
					absUrl = absoluteUrl.String()
				}
				if requestId, exist := requestMap[absUrl]; exist {
					attr.Val = saveFile(ctx, absUrl, nil, requestId, requestMap, baseUrl, n.Data)
				}
				return
			}
		}
		if n.FirstChild != nil {
			c := n.FirstChild
			if c.Type == html.TextNode {
				filePath := saveFile(ctx, "", []byte(c.Data), "", requestMap, baseUrl, "")
				if inArray(srcArray, n.Data) {
					n.Attr = append(n.Attr, html.Attribute{Key: "src", Val: filePath})
					n.FirstChild = nil
				}
				if inArray(hrefArray, n.Data) {
					n.Attr = append(n.Attr, html.Attribute{Key: "href", Val: filePath})
					n.FirstChild = nil
				}
				if inArray(importArray, n.Data) {
					c.Data = fmt.Sprintf("@import url('%s');", filePath)
				}
			}
		}
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		searchNode(ctx, baseUrl, requestMap, c)
	}
}

func replaceHtml(ctx context.Context, baseUrl *url.URL, requestMap map[string]network.RequestID, htmlContent string) string {
	root, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		log.Fatal(err)
	}
	searchNode(ctx, baseUrl, requestMap, root)
	var buf bytes.Buffer
	if err := html.Render(&buf, root); err != nil {
		log.Fatal(err)
	}
	return buf.String()
}
