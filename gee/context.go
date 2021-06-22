package gee

/*
 	封装*http.Request和http.ResponseWriter的方法，简化相关接口的调用，
	只是设计 Context 的原因之一。对于框架来说，还需要支撑额外的功能。
	例如，将来解析动态路由/hello/:name，参数:name的值放在哪呢？
	再比如，框架需要支持中间件，那中间件产生的信息放在哪呢？
	Context 随着每一个请求的出现而产生，请求的结束而销毁，
	和当前请求强相关的信息都应由 Context 承载。因此，设计 Context 结构，
	扩展性和复杂性留在了内部，而对外简化了接口。路由的处理函数，
	以及将要实现的中间件，参数都统一使用 Context 实例，
	Context 就像一次会话的百宝箱，可以找到任何东西。
*/

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type H map[string]interface{}

// Context 提供了对 Method 和 Path 这两个常用属性的直接访问
type Context struct {
	// 原始响应对象
	Writer http.ResponseWriter
	// 原始请求对象
	Req *http.Request
	// 请求信息
	Path   string
	Method string
	// 路由参数解析 即getRouter中的map
	Params map[string]string
	// 响应信息
	StatusCode int
	// 中间件
	handlers []HandlerFunc
	// 当前执行到第几个中间件
	index    int

	engine *Engine
}

// 实例化
func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
		index:  -1,
	}
}

// Next 当在中间件中调用Next方法时，控制权交给了下一个中间件
// 直到调用到最后一个中间件，然后再从后往前，调用每个中间件在Next方法之后定义的部分
func (c *Context) Next() {
	c.index++
	s := len(c.handlers)
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}

func (c *Context) Fail(code int, err string) {
	c.index = len(c.handlers)
	c.JSON(code, H{"message": err})
}

func (c *Context) Param(key string) string {
	value, _ := c.Params[key]
	return value
}

// PostForm 返回表单数据
func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}

// Query 返回query query是给动态网页传递的参数
func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

// Status 设置响应信息
func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

// SetHeader 设置响应头
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

// HTML 支持根据模板文件名选择模板进行渲染
func (c *Context) HTML(code int, name string,data interface{}) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	if err := c.engine.htmlTemplates.ExecuteTemplate(c.Writer,name,data); err != nil {
		c.Fail(500,err.Error())
	}
}
