package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bobg/modver/v2"
	"github.com/bobg/modver/v2/internal"
)

func main() {
	err := filepath.Walk("/", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, "/src/unsafe") {
			return nil
		}
		fmt.Printf("FOUND %s\n", path)
		return nil
	})
	if err != nil && !errors.Is(err, filepath.SkipAll) {
		log.Fatalf("Looking for unsafe: %s", err)
	}

	prURL := os.Getenv("INPUT_PULL_REQUEST_URL")
	host, owner, reponame, prnum, err := internal.ParsePR(prURL)
	if err != nil {
		log.Fatal(err)
	}
	token := os.Getenv("INPUT_GITHUB_TOKEN")
	if token == "" {
		log.Fatal("No GitHub token in the environment variable INPUT_GITHUB_TOKEN")
	}
	ctx := context.Background()
	gh, err := internal.NewClient(ctx, host, token)
	if err != nil {
		log.Fatalf("Creating GitHub client: %s", err)
	}
	result, err := internal.PR(ctx, gh, owner, reponame, prnum)
	if err != nil {
		log.Fatalf("Running comparison: %s", err)
	}
	modver.Pretty(os.Stdout, result)
}
