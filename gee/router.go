package gee

import (
	"net/http"
	"strings"
)

/*
	将和路由相关的方法和结构提取了出来
	方便下一次对 router 的功能进行增强，例如提供动态路由的支持。
*/

// 路由映射表
type router struct {
	// key 为请求方法'GET' 'POST' value存储每种请求方式的trie树根节点
	roots map[string]*node
	// key 由请求方法和静态路由地址构成，例如GET-/、GET-/hello、POST-/hello
	// value 是用户映射的处理方法Handler
	handlers map[string]HandlerFunc
}

func newRouter() *router {
	return &router{
		roots:    make(map[string]*node),
		handlers: make(map[string]HandlerFunc),
	}
}

// 路径按/分隔并返回分隔数组
func parsePattern(pattern string) []string {
	vs := strings.Split(pattern, "/")
	parts := make([]string, 0)
	for _, item := range vs {
		if item != "" {
			parts = append(parts, item)
			// 路径只允许存在一个*
			if item[0] == '*' {
				break
			}
		}
	}
	return parts
}

// 添加路由映射
func (r *router) addRoute(method string, pattern string, handler HandlerFunc) {
	parts := parsePattern(pattern)
	key := method + "-" + pattern

	_, ok := r.roots[method]
	if !ok {
		r.roots[method] = &node{}
	}

	// 将路由添加在前缀树
	r.roots[method].insert(pattern, parts, 0)
	r.handlers[key] = handler
}

func (r *router) getRoute(method string, path string) (*node, map[string]string) {
	// searchParts为查找的路由 即匹配路径
	searchParts := parsePattern(path)
	params := make(map[string]string)

	root, ok := r.roots[method]
	if !ok {
		return nil, nil
	}

	n := root.search(searchParts, 0)

	// 解析了:和*两种匹配符的参数，返回一个 map
	if n != nil {
		// parts为原路径
		parts := parsePattern(n.pattern)
		for index, part := range parts {
			// /p/go/doc匹配到/p/:lang/doc，解析结果为：{lang: "go"}
			if part[0] == ':' {
				params[part[1:]] = searchParts[index]
			}
			// /static/css/geektutu.css匹配到/static/*filepath，
			// 解析结果为{filepath: "css/geektutu.css"}
			if part[0] == '*' && len(part) > 1 {
				params[part[1:]] = strings.Join(searchParts[index:], "/")
				break
			}
		}
		return n, params
	}

	return nil, nil

}

// 查找路由映射表，如果查到，就执行注册的处理方法。如果查不到，
// 就返回 404 NOT FOUND
func (r *router) handle(c *Context) {
	n,params := r.getRoute(c.Method,c.Path)
	if n != nil {
		c.Params = params
		key := c.Method + "-" + n.pattern

		// 将从路由匹配得到的 Handler 添加到 c.handlers列表中
		// 看logger文件 此handler不是中间件
		c.handlers = append(c.handlers,r.handlers[key])
	} else {
		c.handlers = append(c.handlers, func(c *Context) {
			c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
		})
	}
	c.Next()

}


func (r *router) getRoutes(method string) []*node {
	root, ok := r.roots[method]
	if !ok {
		return nil
	}
	nodes := make([]*node, 0)
	root.travel(&nodes)
	return nodes
}