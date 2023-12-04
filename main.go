package main

import (
	"deepal"
	"net/http"
)

func main() {
	//http.HandleFunc("/", indexHandler)
	//http.HandleFunc("/hello", helloHandler)
	//log.Fatal(http.ListenAndServe(":9999", nil))

	//engine := new(Engine)
	//log.Fatal(http.ListenAndServe(":9999", engine))

	r := deepal.New()
	r.GET("/", func(c *deepal.Context) {
		c.HTML(http.StatusOK, "<h1>Hello Deepal<h1>")
	})

	r.GET("/hello", func(c *deepal.Context) {
		// expect /hello?name=deepal
		c.String(http.StatusOK, "hello %s, you're at %s\n", c.Query("name"), c.Path)
	})

	r.POST("/login", func(c *deepal.Context) {
		c.JSON(http.StatusOK, deepal.H{
			"username": c.PostForm("username"),
			"password": c.PostForm("password"),
		})
	})

	r.Run(":9999")
}
