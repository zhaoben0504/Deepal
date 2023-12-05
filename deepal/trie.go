package deepal

type node struct {
	pattern  string  //待匹配路由
	part     string  //路由中的一部分，例如:lang
	children []*node //子节点，例如[doc,tutorial,intro]
	isWild   bool    //是否精确匹配，part含有:或*时为true
}
