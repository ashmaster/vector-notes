package internal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
	"vector-sync/pkg"
)

const (
	syncInterval = 5 * time.Second
)

type Synchronizer struct {
	clientTree *Tree
	serverTree *Tree
	vectorDb   *pkg.Vector
}

func NewSynchronizer(ctx context.Context, clientTree *Tree, serverTree *Tree, vectorDb *pkg.Vector) *Synchronizer {
	return &Synchronizer{
		clientTree: clientTree,
		serverTree: serverTree,
		vectorDb:   vectorDb,
	}
}

func (s *Synchronizer) Start(ctx context.Context) {
	ticker := time.NewTicker(syncInterval)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("Synchronizer stopping...")
				return
			case <-ticker.C:
				if err := s.performSync(ctx); err != nil {
					log.Printf("Sync error: %v", err)
				}
			}
		}
	}()
	<-ctx.Done()
	log.Println("Synchronizer exited.")
}

func (s *Synchronizer) performSync(ctx context.Context) error {
	log.Println("Checking for changes...")

	diffChan := make(chan TreeDiff)
	var handlerWg sync.WaitGroup

	go s.clientTree.CompareAndBuildDiff(ctx, s.serverTree, diffChan)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case diff, ok := <-diffChan:
			if !ok {
				handlerWg.Wait()
				return s.serverTree.SaveToJSON("server.json")
			}

			handlerWg.Add(1)
			go func(d TreeDiff) {
				defer handlerWg.Done()
				s.handleDiff(ctx, d)
			}(diff)
		}
	}
}

func (s *Synchronizer) handleDiff(ctx context.Context, diff TreeDiff) {
	switch diff.Type {
	case Added:
		s.handleFileAdd(ctx, diff.Path)
	case Removed:
		s.handleFileRemove(diff.Path)
	case Modified:
		s.handleFileAdd(ctx, diff.Path)
	default:
		log.Printf("Unknown diff type: %v for path: %s", diff.Type, diff.Path)
	}
}

func (s *Synchronizer) handleFileAdd(ctx context.Context, path string) {
	content, err := s.readFile(path)
	if err != nil {
		log.Printf("Error reading file for %s: %v", path, err)
		return
	}
	fileInfo, statErr := os.Stat(path)
	var fileTime time.Time
	if statErr != nil {
		log.Printf("Error getting file info for %s: %v", path, statErr)
		fileTime = time.Now()
	} else {
		fileTime = fileInfo.ModTime()
	}

	log.Printf("File added: %s", path)
	id := s.createVectorId(path)
	err = s.vectorDb.Upsert(ctx, id, content, path, fmt.Sprintf("%d", fileTime.Unix()))
	if err != nil {
		log.Printf("Error upserting vector for %s: %v", path, err)
		return
	}
	relativePath := s.getRelativePath(path)
	s.serverTree.AddNode(relativePath, content)
}

func (s *Synchronizer) handleFileRemove(path string) {
	log.Printf("File removed: %s", path)
	relativePath := s.getRelativePath(path)
	s.serverTree.RemoveNode(relativePath)
}

func (s *Synchronizer) handleFileModify(path string) {
	content, err := s.readFile(path)
	if err != nil {
		log.Printf("Error handling file modify for %s: %v", path, err)
		return
	}

	log.Printf("File modified: %s", path)
	relativePath := s.getRelativePath(path)
	s.serverTree.AddNode(relativePath, content)
}

func (s *Synchronizer) readFile(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return content, nil
}

func (s *Synchronizer) getRelativePath(path string) string {
	return strings.Replace(path, s.serverTree.Root.Name, "", 1)
}

func (s *Synchronizer) createVectorId(path string) string {
	h := sha256.New()
	h.Write([]byte(path))
	return hex.EncodeToString(h.Sum(nil))
}
