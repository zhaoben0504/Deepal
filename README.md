

手写Web服务框架-Deepal_笔记

# Deepal框架

## 设计初衷

`net/http`库提供了基础的web功能，即监听端口，映射静态路由，解析Http报文等，一些web开发中简单的需求并不支持，需要手工实现

1. 动态路由：例如`hello/:name`,`hello/*`这类的规则
2. 鉴权：没有分组/统一鉴权的能力，需要在每个路由映射的handler中实现
3. 工具集：cookies等
4. ...

 本框架参考了Gin框架，在实现Router时，为了提高性能，用`Trie树`实现的，很多实现的功能较为简单，重在实现过程中的逻辑问题思考与框架设计

## 一、HTTP基础

```go
func main() {
	http.HandleFunc("/hello", helloHandler)
	log.Fatal(http.ListenAndServe(":9999", nil))
}

// handler echoes r.URL.Header
func helloHandler(w http.ResponseWriter, req *http.Request) {
	for k, v := range req.Header {
		fmt.Fprintf(w, "Header[%q] = %q\n", k, v)
	}
}

```

打印header信息

```
curl http://localhost:9999/hello
```

`ListenAndServer`函数的第一个参数是地址，`:9999`表示在9999端口监听，而第二个参数则代表处理所有的http请求实例，nil代表用标准库中的实例处理，**第二个参数，则表示我们基于 `net/http`标准库中实现Web框架的入口，他是一个接口，需要实现方法`ServeHttp`，也就是说，只要传入任何实现了ServerHTTP接口的实例，所有的HTTP请求，就都交给该实例处理了**

```go
type Engine struct {
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		fmt.Fprintf(w, "url.path = [%s]", r.URL.Path)
	case "/hello":
		for k, v := range r.Header {
			fmt.Fprintf(w, "header[%q] = [%q]\n", k, v)
		}
	default:
		fmt.Fprintf(w, "404 for url[%s]", r.URL.Path)
	}
}

func main() {
	engine := new(Engine)
	log.Fatal(http.ListenAndServe(":9999", engine))
}
```

1. 定义了一个空的结构体Engine，实现了`ServerHTTP`。这个方法有两个参数，第二个参数是Request，该对象包含了该HTTP请求的所有信息，比如请求地址，Header和Body等信息；第一个参数是ResponseWriter，利用ResponseWriter可以构造针对该请求的响应
2. 在main函数中，我们给ListenAndServe方法的第二个参数传入了刚刚传入的engine实例。至此，我们将所有的HTTP请求转向了我们自己的处理逻辑，拥有了统一的控制入口，在这里我们可以自由定义路由映射的规则，也可以统一添加一些处理逻辑，比如日志、日常处理等

为了方便我们使用路由，所以可以为框架设置一个路由映射表

```go
package deepal

import (
	"fmt"
	"net/http"
)

// HandlerFunc defines the request handler used by gee
type HandlerFunc func(http.ResponseWriter, *http.Request)

// Engine implement the interface of ServeHTTP
type Engine struct {
	router map[string]HandlerFunc
}

// New is the constructor of gee.Engine
func New() *Engine {
	return &Engine{router: make(map[string]HandlerFunc)}
}

func (engine *Engine) addRoute(method string, pattern string, handler HandlerFunc) {
	key := method + "-" + pattern
	engine.router[key] = handler
}

// GET defines the method to add GET request
func (engine *Engine) GET(pattern string, handler HandlerFunc) {
	engine.addRoute("GET", pattern, handler)
}

// POST defines the method to add POST request
func (engine *Engine) POST(pattern string, handler HandlerFunc) {
	engine.addRoute("POST", pattern, handler)
}

// Run defines the method to start a http server
func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	key := req.Method + "-" + req.URL.Path
	if handler, ok := engine.router[key]; ok {
		handler(w, req)
	} else {
		fmt.Fprintf(w, "404 NOT FOUND: %s\n", req.URL)
	}
}
```

1. 首先我们定义了类型`HandlerFunc`,这是提供给框架用户的，用来定义路由映射的处理方法，我们在`Engine`中，添加了一张路由表`router`，`key`由请求方法和静态路由地址组成，例如`GET-/`、`GET-/hello`、`POST-/hello`，这样针对相同的路由，如果请求方法不同，可以映射不同的处理方法`(Handler)`，`value`是用户映射的处理方法。
2. 用户调用`(Engine).GET()`方法时，会将路由和处理方法映射到映射表`router`中，`(*Engine).Run()`方法，是`ListenAndServer`的包装
3. `Engine`实现的`ServeHTTP`方法的作用是，解析请求的路径，查找路由映射表，如果查到，就执行注册的处理方法，如果查不到，就返回`404`

## 二、上下文

接下来我们将`router`独立出来，并用上下文`(Context)`，封装`Request`和`Response`，提供对`JSON`、`HTML`等放回类型的支持

上下文`Context`

1. 对于Web服务来说，无非是根据请求`*http.Request`，构造响应`http.ResponseWriter`。但是这两个对象提供的接口粒度太细，比如我们要构造一个完整的响应，需要考虑头(Header)和消息体`(Body)`，而``Header`包含了状态码`(StatusCode)`，消息类型`(contentType)`等几乎每次请求都需要设置的信息，因此，如果不进行封装就需要写很多重复的代码。

   例如，对jSON的封装前

   ```go
   obj = map[string]interface{}{
       "name": "geektutu",
       "password": "1234",
   }
   w.Header().Set("Content-Type", "application/json")
   w.WriteHeader(http.StatusOK)
   encoder := json.NewEncoder(w)
   if err := encoder.Encode(obj); err != nil {
       http.Error(w, err.Error(), 500)
   }
   ```

   封装后：

   ```go
   c.JSON(http.StatusOK, gee.H{
       "username": c.PostForm("username"),
       "password": c.PostForm("password"),
   })
   ```

2. 针对使用场景，封装`*http.Request`和`http.ResponseWriter`的方法，简化相关接口的调用，只是`context`的原因之一，设计`context`时拓展性和复杂性留在了内部，对外简化了接口，路由的处理函数，以及将要实现的中间件，参数都统一使用`Context`实例。

   ```go
   type H map[string]interface{}
   
   type Context struct {
   	// origin objects
   	Writer http.ResponseWriter
   	Req    *http.Request
   	// request info
   	Path   string
   	Method string
   	// response info
   	StatusCode int
   }
   
   func newContext(w http.ResponseWriter, req *http.Request) *Context {
   	return &Context{
   		Writer: w,
   		Req:    req,
   		Path:   req.URL.Path,
   		Method: req.Method,
   	}
   }
   
   func (c *Context) PostForm(key string) string {
   	return c.Req.FormValue(key)
   }
   
   func (c *Context) Query(key string) string {
   	return c.Req.URL.Query().Get(key)
   }
   
   func (c *Context) Status(code int) {
   	c.StatusCode = code
   	c.Writer.WriteHeader(code)
   }
   
   func (c *Context) SetHeader(key string, value string) {
   	c.Writer.Header().Set(key, value)
   }
   
   func (c *Context) String(code int, format string, values ...interface{}) {
   	c.SetHeader("Content-Type", "text/plain")
   	c.Status(code)
   	c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
   }
   
   func (c *Context) JSON(code int, obj interface{}) {
   	c.SetHeader("Content-Type", "application/json")
   	c.Status(code)
   	encoder := json.NewEncoder(c.Writer)
   	if err := encoder.Encode(obj); err != nil {
   		http.Error(c.Writer, err.Error(), 500)
   	}
   }
   
   func (c *Context) Data(code int, data []byte) {
   	c.Status(code)
   	c.Writer.Write(data)
   }
   
   func (c *Context) HTML(code int, html string) {
   	c.SetHeader("Content-Type", "text/html")
   	c.Status(code)
   	c.Writer.Write([]byte(html))
   }
   ```

   - 代码最开头，给`map[string]interface{}`起了一个别名`gee.H`，构建`JSON`数据时，显得更简洁
   - `Context`目前只包含了`http.ResponseWriter`和`*http.Request`，另外提供了对`method`和`Path`这两个常用属性的直接访问
   - 对`Query`和`PostForm`进行了简单明了的封装，提供了对这两种方法的直接访问
   - 提供了快速构造`String/Data/Json/Html`响应的方法

3. 最后因为我们引入了`context`对`router`进行封装，所以我们对框架入口进行一些修改

   ```go
   // HandlerFunc defines the request handler used by gee
   type HandlerFunc func(*Context)
   
   // Engine implement the interface of ServeHTTP
   type Engine struct {
   	router *router
   }
   
   // New is the constructor of gee.Engine
   func New() *Engine {
   	return &Engine{router: newRouter()}
   }
   
   func (engine *Engine) addRoute(method string, pattern string, handler HandlerFunc) {
   	engine.router.addRoute(method, pattern, handler)
   }
   
   // GET defines the method to add GET request
   func (engine *Engine) GET(pattern string, handler HandlerFunc) {
   	engine.addRoute("GET", pattern, handler)
   }
   
   // POST defines the method to add POST request
   func (engine *Engine) POST(pattern string, handler HandlerFunc) {
   	engine.addRoute("POST", pattern, handler)
   }
   
   // Run defines the method to start a http server
   func (engine *Engine) Run(addr string) (err error) {
   	return http.ListenAndServe(addr, engine)
   }
   
   func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
   	c := newContext(w, req)
   	engine.router.handle(c)
   }
   ```

## 三、前缀树

- 使用前缀树实现动态路由解析
- 支持`:name`和`*filepath`两种模式

简介：

之前我们用map存储路由表，使用map存储键值对，索引非常高效，但是键值对的存储方式，不支持静态路由

动态路由：

由一条路由规则可以匹配某一类型而非某一条固定的路由，例如`hello/:name`可以匹配`/hello/mike`、`hello/jack`等

实现动态路由最常用的数据结构，被称为前缀树(Trie树)，每一个节点的所有子节点都拥有相同的前缀，这种结构非常适用于路由匹配，例如我们定义了如下路由规则

- /:lang/doc
- /:lang/tutorial
- /:lang/intro
- /about
- /p/blog
- /p/related

那我们的前缀树就是这样

![image-20231205221258089](.\image\Trie树.jpg)

http请求的路径恰好是由/分割成段的，因此可以把每一段作为前缀树的一个节点，我们通过树结构查询，如果中间某一层节点不满足条件，则代表没匹配到，查询结束



所以我们的路由应该具备两个功能

1. 参数匹配`：`,例如`/p/:lang/doc`，可以匹配`/p/c/doc`和`/p/go/doc`
2. 通配`*`，例如`/static/*filepath`，可以匹配`/static/fav.ico`，也可以匹配`/static/js/jQuery.js`，这种模式常用于静态服务器，能够递归匹配子路径

```go
type node struct {
	pattern  string // 待匹配路由，例如 /p/:lang
	part     string // 路由中的一部分，例如 :lang
	children []*node // 子节点，例如 [doc, tutorial, intro]
	isWild   bool // 是否精确匹配，part 含有 : 或 * 时为true
}
```

为实现动态路由匹配，加上了`isWild`这个参数，当我们匹配`/p/go/doc`这个路由时，第一层节点，p精准匹配到了p，第二层节点，go模糊匹配到了`:lang`，那么将会把`lang`这个参数赋值为go，继续下一层匹配。我们将匹配的逻辑封装

```go
// 第一个匹配成功的节点，用于插入
func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part || child.isWild {
			return child
		}
	}
	return nil
}
// 所有匹配成功的节点，用于查找
func (n *node) matchChildren(part string) []*node {
	nodes := make([]*node, 0)
	for _, child := range n.children {
		if child.part == part || child.isWild {
			nodes = append(nodes, child)
		}
	}
	return nodes
}
```

