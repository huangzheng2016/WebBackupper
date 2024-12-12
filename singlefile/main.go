package main

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/url"
	"os/exec"
	"strings"
)

const (
	SINGLEFILE_EXECUTABLE = "single-file"
	BROWSER_PATH          = "/usr/bin/chromium-browser"
	STATIC_FILE_PATH      = "static/"
)

type Page struct {
	gorm.Model
	Host   string
	Path   string
	RawURL string
}

//Path重命名规则
//没有最后/的，判断结尾是否是html，不是就加上
//有的话/去掉用上述规则

func main() {
	db, _ := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	db.AutoMigrate(&Page{})
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"test": "OK"})
	})
	r.POST("/", func(c *gin.Context) {
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

		argsInput := c.PostFormArray("args")
		for i := range argsInput {
			bk := false
			for j := range args {
				if strings.HasPrefix(argsInput[i], strings.Split(args[j], "=")[0]) {
					args[j] = argsInput[i]
					bk = true
					break
				}
			}
			if !bk {
				args = append(args, argsInput[i])
			}
		}

		urlInput := c.PostForm("url")
		urlObj, _ := url.Parse(urlInput)
		page := Page{
			Host:   urlObj.Host,
			Path:   urlObj.Path,
			RawURL: urlInput,
		}

		if urlInput != "" {
			db.Create(&page)
			savePage(page)

			args = append(args, urlInput)
			cmd := exec.Command(SINGLEFILE_EXECUTABLE,
				args...)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()
			if err != nil {
				c.JSON(500, gin.H{
					"code": 500,
					"msg":  fmt.Sprint("Error: %v", err),
					"out":  stdout.String(),
					"err":  stderr.String(),
				})
				return
			}
			c.JSON(200, gin.H{
				"code": 200,
				"msg":  "success",
				"out":  stdout.String(),
				"err":  stderr.String(),
			})
		} else {
			c.JSON(500, gin.H{
				"code": 500,
				"msg":  "Error: url parameter not found.",
			})
		}
	})
	//进行站点创建
	r.POST("/create", func(c *gin.Context) {
		//创建新的Site对象
		//获取请求中的url参数
		urlParam := c.PostForm("url")
		//解析url
		_, err := url.Parse(urlParam)
		if err != nil {
			c.JSON(500, gin.H{
				"code": 500,
				"msg":  "Error: url parameter error.",
			})
		}
		//获取host
		//获取path

	})
	r.Static("/static", STATIC_FILE_PATH)
	r.Run(":8010")
}
