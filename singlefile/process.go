package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/vincent-petithory/dataurl"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

var db *gorm.DB

func saveStaticFile(link string) string {
	//解析data类型url
	dataURL, _ := dataurl.DecodeString(link)
	// md5文件内容作为文件名
	hash := md5.Sum(dataURL.Data)
	filename := STATIC_FILE_PATH + hex.EncodeToString(hash[:]) + "." + dataURL.Subtype
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		//不存在就写文件
		file, err := os.Create(filename)
		if err != nil {
			println(err)
		}
		defer file.Close()
		_, err = file.Write(dataURL.Data)
		if err != nil {
			println(err)
		}
	}
	return filename
}
func parseHTML(htmlPage *bytes.Buffer, page Page) string {
	db, _ = gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})

	dom, err := goquery.NewDocumentFromReader(htmlPage)
	if err != nil {
		panic("Error loading HTML")
	}
	//解析所有能进行跳转的组件
	//a标签获取href
	//input获取submit类型触发的form
	//button获取submit类型触发的form

	//解析所有图片，保存
	dom.Find("img").Each(func(i int, s *goquery.Selection) {
		link, exists := s.Attr("src")
		if exists {
			filename := saveStaticFile(link)
			s.SetAttr("src", filename)
		}
	})
	//读静态文件内容
	dom.Find("script").Each(func(i int, s *goquery.Selection) {
		link, exists := s.Attr("src")
		if exists {
			filename := saveStaticFile(link)
			s.SetAttr("src", filename)
		}
	})
	//读取所有能跳转的a标签
	dom.Find("a").Each(func(i int, s *goquery.Selection) {
		link, exists := s.Attr("href")
		if exists {
			//判断是否是站内链接，第一种情况，页面中的链接是相对路径
			//其实在single file中不存在（）
			//第二种情况是直接写了完整URL
			hrefUrl, _ := url.Parse(link)
			if hrefUrl.Host == page.Host {
				//站内链接，判断是否存在于数据库中
				var page Page
				db.Where(&Page{Host: hrefUrl.Host, Path: hrefUrl.Path}).First(&page)
				if page.ID == 0 {
					//创建page，发起请求，保存
					page.RawURL = link
					page.Host = hrefUrl.Host
					page.Path = hrefUrl.Path
					savePage(page)
				}
			}
			if strings.HasSuffix(page.Path, "/") {
				s.SetAttr("href", page.Path[0:len(page.Path)-1]+".html")
			} else {
				s.SetAttr("href", page.Path+".html")
			}
		}
	})
	modedHtml, _ := dom.Html()
	return modedHtml
}
func getPageContent(page Page) (bytes.Buffer, error) {
	args := []string{
		//"--browser-executable-path=" + BROWSER_PATH,
		"--browser-args='[\"--no-sandbox\"]'",
		"--block-scripts=false",
		"--browser-width=1600",
		"--browser-height=900",
		"--compress-CSS=true",
		"--browser-ignore-insecure-certs=true",
		"--save-original-urls=true",
		"--max-resource-size=50",
		"--browser-wait-delay=1000",
		"--browser-load-max-time=60000",
		"--load-deferred-images-max-idle-time=10000",
		"--dump-content=true"}

	args = append(args, page.RawURL)
	cmd := exec.Command(SINGLEFILE_EXECUTABLE,
		args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return bytes.Buffer{}, err
	}
	//其他情况直接返回stdout
	return stdout, nil
}
func savePage(page Page) {
	db, _ = gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	println("save page")
	println(page.RawURL)
	//直接getpage然后parse
	content, err := getPageContent(page)
	if err != nil {
		fmt.Println(err)
	}
	modedHtml := parseHTML(&content, page)
	//保存文件
	file, err := os.Create(STATIC_FILE_PATH + page.Host + page.Path + ".html")
	if err != nil {
		println(err)
	}
	defer file.Close()
	_, err = file.Write([]byte(modedHtml))
	//保存到数据库
	db.Create(&page)
}
