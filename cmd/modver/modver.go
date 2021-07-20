// Command modver compares two versions of the same Go packages
// and tells whether a Major, Minor, or Patchlevel version bump
// (or None)
// is needed to go from one to the other.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/bobg/modver"
)

func main() {
	gitRepo := flag.String("git", "", "Git repo URL")
	flag.Parse()

	if *gitRepo != "" {
		if flag.NArg() != 2 {
			fmt.Printf("Usage: %s -git OLDERSHA NEWERSHA\n", os.Args[0])
			os.Exit(1)
		}

		res, err := modver.CompareGit(context.Background(), *gitRepo, flag.Arg(0), flag.Arg(1))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(res)
		return
	}

	if flag.NArg() != 2 {
		fmt.Printf("Usage: %s -git OLDERSHA NEWERSHA\n", os.Args[0])
		os.Exit(1)
	}
	res, err := modver.CompareDirs(flag.Arg(0), flag.Arg(1))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)
}
