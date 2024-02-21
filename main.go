package main

import (
	"deepal"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
)

func onlyForV2() deepal.HandlerFunc {
	return func(c *deepal.Context) {
		// Start timer
		t := time.Now()
		// if a server error occurred
		// Calculate resolution time
		log.Printf("[%d] %s in %v for group v2", c.StatusCode, c.Req.RequestURI, time.Since(t))
	}
}

type student struct {
	Name string
	Age  int8
}

func FormatAsDate(t time.Time) string {
	year, month, day := t.Date()
	return fmt.Sprintf("%d-%02d-%02d", year, month, day)
}

func main() {
	r := deepal.Default()
	r.Use(deepal.Logger())

	r.SetFuncMap(template.FuncMap{
		"FormatAsDate": FormatAsDate,
	})
	r.LoadHTMLGlob("templates/*")
	r.Static("/assets", "./static")

	stu1 := &student{Name: "Deepal", Age: 20}
	stu2 := &student{Name: "Jack", Age: 22}

	r.GET("/index", func(c *deepal.Context) {
		c.HTML(http.StatusOK, "Index Page", "<h1>Index Page</h1>")
	})

	v1 := r.Group("/v1")
	{
		v1.GET("/", func(c *deepal.Context) {
			c.HTML(http.StatusOK, "Hello Deepal", "<h1>Hello Deepal</h1>")
		})

		v1.GET("/hello", func(c *deepal.Context) {
			// expect /hello?name=deepal
			c.String(http.StatusOK, "hello %s, you're at %s\n", c.Query("name"), c.Path)
		})
	}

	// 同组路由使用统一中间件
	v2 := r.Group("/v2")
	v2.Use(onlyForV2())
	{
		v2.GET("/hello/:name", func(c *deepal.Context) {
			// expect /hello/deepal
			c.String(http.StatusOK, "hello %s, you're at %s\n", c.Param("name"), c.Path)
		})
		v2.POST("/login", func(c *deepal.Context) {
			c.JSON(http.StatusOK, deepal.H{
				"username": c.PostForm("username"),
				"password": c.PostForm("password"),
			})
		})

	}

	// 渲染个人信息
	r.GET("/students", func(c *deepal.Context) {
		c.HTML(http.StatusOK, "arr.tmpl", deepal.H{
			"title":  "deepal",
			"stuArr": [2]*student{stu1, stu2},
		})
	})

	// 渲染日期
	r.GET("/date", func(c *deepal.Context) {
		c.HTML(http.StatusOK, "custom_func.tmpl", deepal.H{
			"title": "deepal",
			"now":   time.Date(2019, 8, 17, 0, 0, 0, 0, time.UTC),
		})
	})

	// 错误处理
	r.GET("/panic", func(c *deepal.Context) {
		names := []string{"deepal"}
		c.String(http.StatusOK, names[100])
	})

	r.Run(":9999")
}
