package main

import (
	"context"
	"log"
	"os"

	"github.com/bobg/modver/v2"
	"github.com/bobg/modver/v2/internal"
)

func main() {
	os.Setenv("GOROOT", "/usr/local/go") // Work around some Docker weirdness.

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
