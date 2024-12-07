package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"github.com/PuerkitoBio/goquery"
	"github.com/vincent-petithory/dataurl"
	"os"
)

func parseHTML(htmlPage bytes.Buffer) string {
	dom, err := goquery.NewDocumentFromReader(&htmlPage)
	if err != nil {
		panic("Error loading HTML")
	}
	//解析所有能进行跳转的组件
	//a标签获取href
	//input获取submit类型触发的form
	//button获取submit类型触发的form

	//解析所有图片，保存
	dom.Find("img").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Attr("src")
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

		if err != nil {
			println(err)
		}
		s.SetAttr("src", filename)

	})

	//读静态文件内容
	dom.Find("$('script')")
	modedHtml, _ := dom.Html()
	return modedHtml
}
