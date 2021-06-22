package gee

import (
	"html/template"
	"log"
	"net/http"
	"path"
	"strings"
)

// HandlerFunc 是提供给框架用户的，用来定义路由映射的处理方法。
type HandlerFunc func(*Context)

// Engine 实现接口的ServeHTTP方法
type Engine struct {
	// Engine拥有RouterGroup所有的能力，嵌套类型，类似java的继承
	*RouterGroup
	// 路由映射表
	// key 由请求方法和静态路由地址构成，例如GET-/、GET-/hello、POST-/hello
	// value 是用户映射的处理方法Handler
	router *router
	// 存储所有分组
	groups []*RouterGroup

	// html渲染
	// 将所有的模板加载进内存
	htmlTemplates *template.Template
	// 所有的自定义模板渲染函数
	funcMap       template.FuncMap
}

// RouterGroup 路由分组
type RouterGroup struct {
	// 前缀
	prefix string
	// 中间件
	middlewares []HandlerFunc
	// 所有分组共享一个engine实例
	engine *Engine
}

// New 实例化
func New() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}

// Group 新建一个路由分组
// 所有分组共享一个engine实例
func (group *RouterGroup) Group(prefix string) *RouterGroup {
	engine := group.engine
	newGroup := &RouterGroup{
		prefix: group.prefix + prefix,
		engine: engine,
	}

	// 添加分组
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

// SetFuncMap 设置自定义模板渲染函数
func (engine *Engine) SetFuncMap(funcMap template.FuncMap) {
	engine.funcMap = funcMap
}

// LoadHTMLGlob 加载模板的方法
func (engine *Engine) LoadHTMLGlob(pattern string) {
	engine.htmlTemplates = template.Must(template.New("").Funcs(engine.funcMap).ParseGlob(pattern))
}

// 添加路由映射
func (group *RouterGroup) addRoute(method string, comp string, handler HandlerFunc) {
	// 在路径前加上分组前缀
	pattern := group.prefix + comp
	log.Printf("Route %4s - %s", method, pattern)
	group.engine.router.addRoute(method, pattern, handler)
}

// GET 定义了添加 GET 请求的方法
// 会将路由和处理方法注册到映射表 router 中
func (group *RouterGroup) GET(pattern string, handler HandlerFunc) {
	group.addRoute("GET", pattern, handler)
}

// POST 定义了添加 POST 请求的方法
func (group *RouterGroup) POST(pattern string, handler HandlerFunc) {
	group.addRoute("POST", pattern, handler)
}

// 创建静态文件处理
func (group *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	// 找到/assets的原始前缀并拼接得到 absolutePath得到文件真实地址
	absolutePath := path.Join(group.prefix, relativePath)
	// StripPrefix会把路径中的absolutePath去除并将剩余部分交由FileServer处理
	// FileServer 会将/assets转换为/usr/geektutu/blog/static
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))

	return func(c *Context) {

		// 得到filepath的参数
		file := c.Param("filepath")
		if _, err := fs.Open(file); err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		fileServer.ServeHTTP(c.Writer, c.Req)
	}

}

// Static 暴露给用户
// 用户可以将磁盘上的某个文件夹root映射到路由relativePath
// r.Static("/assets", "/usr/geektutu/blog/static")
// 用户访问localhost:9999/assets/js/geektutu.js
// 最终返回/usr/geektutu/blog/static/js/geektutu.js
func (group *RouterGroup) Static(relativePath string, root string) {
	// Dir将字符串转换为文件系统
	handler := group.createStaticHandler(relativePath, http.Dir(root))
	// relativePath == /assets, /*filepath == /js/geektutu.js
	urlPattern := path.Join(relativePath, "/*filepath")

	group.GET(urlPattern, handler)
}

// Run 定义了启动 http 服务器的方法，是 ListenAndServe 的包装
func (engine *Engine) Run(addr string) (err error) {

	return http.ListenAndServe(addr, engine)
}

// Use 将中间件应用到某个 Group
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
}

/*
	第二个参数是 Request ，该对象包含了该HTTP请求的所有的信息，
	比如请求地址、Header和Body等信息；第一个参数是 ResponseWriter ，
	利用 ResponseWriter 可以构造针对该请求的响应。
*/

// Engine实现的 ServeHTTP 方法的作用就是，解析请求的路径，
// 查找路由映射表，如果查到，就执行注册的处理方法。如果查不到，
// 就返回 404 NOT FOUND
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var middlewares []HandlerFunc
	// 判断该请求适用于哪些中间件
	for _, group := range engine.groups {
		// 若请求的路径中存在某分组前缀
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}

	c := newContext(w, req)
	// 将匹配的中间件列表赋值给该请求的 c.handlers
	c.handlers = middlewares
	c.engine = engine
	engine.router.handle(c)
}
