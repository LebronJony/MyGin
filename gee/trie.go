package gee

import (
	"fmt"
	"strings"
)

// 前缀树trie树
type node struct {
	// 待匹配路由，例如 /p/:lang
	pattern string
	// 路由中的一部分，例如 :lang
	part string
	// 子节点，例如 [doc, tutorial, intro]
	children []*node
	// 是否模糊匹配，part 含有 : 或 * 时为true
	isWild bool
}

func (n *node) String() string {
	return fmt.Sprintf("node{pattern = %s, part = %s, isWild = %t}", n.pattern, n.part, n.isWild)
}

// 查找前缀树中第一个匹配成功的子节点，用于插入
func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part || child.isWild {
			return child
		}
	}
	return nil
}

// 遍历前缀树中所有匹配成功的子节点，用于查找
func (n *node) matchChildern(part string) []*node {
	nodes := make([]*node, 0)
	for _, child := range n.children {
		if child.part == part || child.isWild {
			nodes = append(nodes, child)
		}
	}
	return nodes
}

// 前缀树插入操作
func (n *node) insert(pattern string, parts []string, height int) {

	// 如果遍历至插入路径结尾则返回
	if len(parts) == height {
		n.pattern = pattern
		return
	}

	part := parts[height]
	child := n.matchChild(part)

	// 若找不到匹配，则在前缀树该节点处新生成一个子节点
	if child == nil {
		child = &node{part: part, isWild: part[0] == ':' || part[0] == '*'}
		n.children = append(n.children, child)
	}
	// 子节点递归插入
	child.insert(pattern, parts, height+1)

}

func (n *node) search(parts []string, height int) *node {
	// 若遍历至路径结尾 或 匹配到*，则返回
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		if n.pattern == "" {
			return nil
		}
		return n
	}

	part := parts[height]
	children := n.matchChildern(part)

	// 遍历该节点所有匹配成功的子节点
	for _, child := range children {
		// 匹配成功的子节点递归遍历查找
		result := child.search(parts, height+1)
		if result != nil {
			// 返回查找的结果
			return result
		}
	}

	return nil
}

func (n *node) travel(list *([]*node)) {
	if n.pattern != "" {
		*list = append(*list, n)
	}
	for _, child := range n.children {
		child.travel(list)
	}
}
