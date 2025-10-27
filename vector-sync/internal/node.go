package internal

type TreeNode struct {
	Hash     string
	Name     string
	Children map[string]*TreeNode
}

func NewTreeNode(hash string, name string) *TreeNode {
	return &TreeNode{
		Hash:     hash,
		Name:     name,
		Children: make(map[string]*TreeNode),
	}
}

func (n *TreeNode) IsDir() bool {
	return len(n.Name) > 0 && n.Name[len(n.Name)-1] == '/'
}
