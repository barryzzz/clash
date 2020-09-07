package trie

// Node is the trie's node
type Node struct {
	Data     interface{}
	children map[string]*Node
}

func (n *Node) getChild(s string) *Node {
	return n.children[s]
}

func (n *Node) hasChild(s string) bool {
	return n.getChild(s) != nil
}

func (n *Node) addChild(s string, child *Node) {
	n.children[s] = child
}

func (n *Node) equals(other *Node) bool {
	if n.Data != other.Data {
		return false
	}

	if len(n.children) != len(other.children) {
		return false
	}

	for k, v := range n.children {
		o, ok := other.children[k]
		if !ok || !v.equals(o) {
			return false
		}
	}

	return true
}

func newNode(data interface{}) *Node {
	return &Node{
		Data:     data,
		children: map[string]*Node{},
	}
}
