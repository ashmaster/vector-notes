package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Tree struct {
	Root *TreeNode
}

type DiffType int

const (
	Added DiffType = iota
	Removed
	Modified
)

type TreeDiff struct {
	Type DiffType
	Path string
}

func NewTree(rootHash string, relativePath string) *Tree {
	return &Tree{
		Root: NewTreeNode(rootHash, strings.TrimSuffix(relativePath, "/")+"/"),
	}
}
func (t *Tree) BuildTree() error {
	var mu sync.Mutex
	var wg sync.WaitGroup

	numWorkers := 1
	fileChan := make(chan string, numWorkers)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileChan {
				content, err := os.ReadFile(filepath.Join(t.Root.Name, path))
				if err != nil {
					continue
				}
				mu.Lock()
				t.AddNode(path, content)
				mu.Unlock()
			}
		}()
	}
	err := fs.WalkDir(os.DirFS(t.Root.Name), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			if strings.HasSuffix(path, ".md") {
				fileChan <- path
			}
		} else {
			// Add directories immediately (they're fast)
			mu.Lock()
			t.AddNode(path+"/", nil)
			mu.Unlock()
		}
		return nil
	})

	close(fileChan)
	wg.Wait()
	return err
}

func (t *Tree) AddNode(path string, content []byte) {
	current := t.Root
	segments := SplitPath(path)

	for i, segment := range segments {
		if segment == "" {
			continue
		}
		isFile := (i == len(segments)-1) && content != nil
		if _, exists := current.Children[segment]; !exists {
			if isFile {
				// It's a file
				current.Children[segment] = NewTreeNode(CalculateHash(content), segment)
			} else {
				// It's a directory
				current.Children[segment] = NewTreeNode("", segment)
			}
		} else {
			if isFile {
				// Update file content hash if it already exists
				current.Children[segment].Hash = CalculateHash(content)
			}
		}
		current = current.Children[segment]
	}
	t.CalculateDirectoryHashes()
}

func (t *Tree) PrintTree() {
	log.Println("************** Tree Structure **************")
	printNode(t.Root, 0)
}

func printNode(node *TreeNode, level int) {
	prefix := strings.Repeat("  ", level)
	if node.IsDir() {
		fmt.Printf("%s- %s (hash: %s)\n", prefix, node.Name, node.Hash)
		for _, child := range node.Children {
			printNode(child, level+1)
		}
	} else {
		fmt.Printf("%s- %s (hash: %s)\n", prefix, node.Name, node.Hash)
	}
}

func (t *Tree) RemoveNode(path string) {

	current := t.Root
	segments := SplitPath(path)

	for i, segment := range segments {
		if segment == "" {
			continue
		}
		if child, exists := current.Children[segment]; exists {
			if i == len(segments)-1 {
				delete(current.Children, segment)
				break
			}
			current = child
		} else {
			break
		}
	}
	t.CalculateDirectoryHashes()
}

func (t *Tree) CalculateDirectoryHashes() string {
	hash := calculateNodeHash(t.Root)
	return hash
}

func calculateNodeHash(node *TreeNode) string {
	if !node.IsDir() {
		return node.Hash
	}

	var content strings.Builder
	childrenToRemove := make([]string, 0)

	for name, child := range node.Children {
		childHash := calculateNodeHash(child)

		if child.IsDir() && len(child.Children) == 0 {
			childrenToRemove = append(childrenToRemove, name)
		} else {
			content.WriteString(childHash)
		}
	}

	// Remove empty directories
	for _, name := range childrenToRemove {
		delete(node.Children, name)
	}

	node.Hash = CalculateHash([]byte(content.String()))
	return node.Hash
}

// SerializableNode represents the JSON structure for a tree node
type SerializableNode struct {
	Name     string                       `json:"name"`
	Hash     string                       `json:"hash"`
	Children map[string]*SerializableNode `json:"Children"`
}

// SaveToJSON saves the tree structure to a JSON file in .client directory
func (t *Tree) SaveToJSON(filename string) error {
	clientDir := ".server"
	if err := os.MkdirAll(clientDir, 0755); err != nil {
		return err
	}

	rootData := t.convertToSerializable(t.Root)

	jsonData := map[string]*SerializableNode{
		t.Root.Name: rootData,
	}

	data, err := json.MarshalIndent(jsonData, "", "    ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(clientDir, filename)
	return os.WriteFile(filePath, data, 0644)
}

// convertToSerializable converts TreeNode to SerializableNode recursively
func (t *Tree) convertToSerializable(node *TreeNode) *SerializableNode {
	serializable := &SerializableNode{
		Name:     node.Name,
		Hash:     node.Hash,
		Children: make(map[string]*SerializableNode),
	}

	for _, child := range node.Children {
		serializable.Children[child.Name] = t.convertToSerializable(child)
	}

	return serializable
}

func (t *Tree) LoadTreeFromJSON(filename string) error {
	filePath := filepath.Join(".server", filename)
	data, err := os.ReadFile(filePath)
	emptyTree := false
	if err != nil {
		emptyTree = true
	}

	var jsonData map[string]*SerializableNode

	if emptyTree {
		t.Root = NewTreeNode("", t.Root.Name)
		return nil
	}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return err
	}

	if _, exists := jsonData[t.Root.Name]; !exists {
		t.Root = NewTreeNode("", t.Root.Name)
		return nil
	}

	// Rebuild the tree from the JSON data
	t.Root = t.buildFromSerializable(jsonData[t.Root.Name])
	t.CalculateDirectoryHashes()
	return nil
}

func (t *Tree) buildFromSerializable(sNode *SerializableNode) *TreeNode {
	node := NewTreeNode(sNode.Hash, sNode.Name)
	for _, child := range sNode.Children {
		node.Children[child.Name] = t.buildFromSerializable(child)
	}
	return node
}

func (t *Tree) CompareAndBuildDiff(ctx context.Context, other *Tree, diffChan chan<- TreeDiff) {
	defer close(diffChan)
	t.compareNodes(ctx, t.Root, other.Root, t.Root.Name, diffChan)
}

func (t *Tree) compareNodes(ctx context.Context, currentNode, otherNode *TreeNode, currentPath string, diffChan chan<- TreeDiff) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	for name, currentChild := range currentNode.Children {
		childPath := currentPath + name
		if otherChild, exists := otherNode.Children[name]; exists {
			if currentChild.IsDir() && otherChild.IsDir() {
				t.compareNodes(ctx, currentChild, otherChild, childPath, diffChan)
			} else if !currentChild.IsDir() && !otherChild.IsDir() {
				if currentChild.Hash != otherChild.Hash {
					diffChan <- TreeDiff{Type: Modified, Path: childPath}
				}
			} else {
				diffChan <- TreeDiff{Type: Modified, Path: childPath}
			}
		} else {
			if currentChild.IsDir() {
				t.compareNodes(ctx, currentChild, otherNode, childPath, diffChan)
			} else {
				diffChan <- TreeDiff{Type: Added, Path: childPath}
			}

		}
	}

	for name, _ := range otherNode.Children {
		if _, exists := currentNode.Children[name]; !exists {
			childPath := currentPath + name
			diffChan <- TreeDiff{Type: Removed, Path: childPath}
		}
	}
}
