package internal

import (
	"context"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type FileWatcher struct {
	watcher *fsnotify.Watcher
	tree    *Tree
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func NewFileWatcher(ctx context.Context, tree *Tree) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	fw := &FileWatcher{
		watcher: watcher,
		tree:    tree,
		ctx:     ctx,
		cancel:  cancel,
	}
	return fw, nil
}

func (fw *FileWatcher) StartWatching() error {
	log.Printf("Starting to watch directory: %s", fw.tree.Root.Name)
	defer fw.watcher.Close()

	err := filepath.Walk((fw.tree.Root.Name), func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return fw.watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	fw.wg.Add(1)
	go fw.watchLoop()
	fw.wg.Wait()
	return nil
}

func (fw *FileWatcher) watchLoop() {
	defer fw.wg.Done()
	log.Println("File watcher loop started")
	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				log.Println("File watcher events channel closed")
				return
			}
			fw.handleEvent(event)
		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)
			// Handle error (log it, etc.)
			_ = err
		case <-fw.ctx.Done():
			return
		}
	}
}

func (fw *FileWatcher) handleEvent(event fsnotify.Event) {
	select {
	case <-fw.ctx.Done():
		return
	default:
	}

	if !strings.HasSuffix(event.Name, ".md") && !fw.isDirectory(event.Name) {
		return
	}
	fw.mu.Lock()
	defer fw.mu.Unlock()

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		fw.handleCreate(event.Name)
	case event.Op&fsnotify.Write == fsnotify.Write:
		fw.handleWrite(event.Name)
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		fw.handleRemove(event.Name)
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		fw.handleRemove(event.Name)
	}
}

func (fw *FileWatcher) handleCreate(path string) {
	if fw.isDirectory(path) {
		fw.watcher.Add(path)
	} else if strings.HasSuffix(strings.TrimPrefix(path, fw.tree.Root.Name+"/"), ".md") {
		content, err := os.ReadFile(path)
		if err != nil {
			// Handle error (log it, etc.)
			return
		}

		fw.tree.AddNode(strings.TrimPrefix(path, fw.tree.Root.Name), content)
	}
}

func (fw *FileWatcher) handleWrite(path string) {
	if !strings.HasSuffix(path, ".md") {
		return
	}
	content, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Error reading file %s: %v", path, err)
		return
	}
	fw.tree.AddNode(strings.TrimPrefix(path, fw.tree.Root.Name), content)
}

func (fw *FileWatcher) handleRemove(path string) {
	if fw.isDirectory(path) {
		fw.watcher.Remove(path)
		fw.tree.RemoveNode(strings.TrimPrefix(path, fw.tree.Root.Name) + "/")
	} else if strings.HasSuffix(strings.TrimPrefix(path, fw.tree.Root.Name), ".md") {
		fw.tree.RemoveNode(strings.TrimPrefix(path, fw.tree.Root.Name))
	}
}

func (fw *FileWatcher) isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
