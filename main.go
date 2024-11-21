package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"path/filepath"
)

func main() {
	downloadWebPage("https://github.com/huangzheng2016/WebBackupper")
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.File("website.html")
	})
	r.GET("/data/*file", func(c *gin.Context) {
		file := c.Param("file")
		path := filepath.Join("data", file)
		fmt.Println(path)
		c.File(path)
	})
	r.Run(":8080")

}
