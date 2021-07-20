package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/bobg/modver"
)

func main() {
	var (
		repoURL  = flag.String("repo", "", "repo URL")
		olderSHA = flag.String("older", "", "older SHA")
		newerSHA = flag.String("newer", "", "newer SHA")
		existing = flag.String("existing", "", "xxx testing: existing tmpdir")
	)
	flag.Parse()

	if *existing != "" {
		res, err := modver.CompareDirs(filepath.Join(*existing, "older"), filepath.Join(*existing, "newer"))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(res)
		return
	}

	res, err := modver.CompareGit(context.Background(), *repoURL, *olderSHA, *newerSHA)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)
}

func stripPrefixFn(parent string) func(string) string {
	return func(in string) string {
		fmt.Printf("xxx stripPrefix(%s): %s\n", parent, in)

		if result := strings.TrimPrefix(in, "_"+parent+"/older"); result != in {
			return result
		}
		return strings.TrimPrefix(in, "_"+parent+"/newer")
	}
}
