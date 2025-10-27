package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"vector-sync/internal"
	"vector-sync/pkg"
)

func main() {
	config, err := internal.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}
	vectorDb, err := pkg.NewVector(config.PineconeAPIKey, config.PineconeHost, "joyful-elm", config.EmbeddingUrl)
	if err != nil {
		fmt.Printf("Error creating vector database client: %v\n", err)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clientTree := internal.NewTree("", config.NotesDir)
	serverTree := internal.NewTree("", config.NotesDir)
	err = clientTree.BuildTree()
	serverTree.LoadTreeFromJSON("server.json")
	if err != nil {
		fmt.Printf("Error building tree: %v\n", err)
		return
	}

	watcher, err := internal.NewFileWatcher(ctx, clientTree)
	if err != nil {
		fmt.Printf("Error creating file watcher: %v\n", err)
		return
	}
	go func() {
		watcher.StartWatching()
	}()

	synchronizer := internal.NewSynchronizer(ctx, clientTree, serverTree, vectorDb)
	go synchronizer.Start(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

}
