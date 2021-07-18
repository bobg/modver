package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"golang.org/x/tools/go/packages"

	"github.com/bobg/modver"
)

func main() {
	var (
		repoURL      = flag.String("repo", "", "repo URL")
		olderHashStr = flag.String("older", "", "older hash")
		newerHashStr = flag.String("newer", "", "newer hash")
		existing     = flag.String("existing", "", "xxx testing: existing tmpdir")
	)
	flag.Parse()

	var (
		parent string
		err    error
	)

	if *existing != "" {
		parent = *existing
	} else {
		parent, err = os.MkdirTemp("", "gitmodver")
		if err != nil {
			log.Fatal(err)
		}
		// xxx defer os.RemoveAll(parent)
	}

	var (
		olderDir             = filepath.Join(parent, "older")
		newerDir             = filepath.Join(parent, "newer")
		olderRepo, newerRepo *git.Repository
	)

	if *existing != "" {
		olderRepo, err = git.PlainOpen(olderDir)
		if err != nil {
			log.Fatal(err)
		}
		newerRepo, err = git.PlainOpen(newerDir)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		olderDir = filepath.Join(parent, "older")
		err = os.Mkdir(olderDir, 0755)
		if err != nil {
			log.Fatal(err)
		}
		newerDir = filepath.Join(parent, "newer")
		err = os.Mkdir(newerDir, 0755)
		if err != nil {
			log.Fatal(err)
		}

		cloneOpts := &git.CloneOptions{URL: *repoURL, NoCheckout: true}

		olderRepo, err = git.PlainClone(olderDir, false, cloneOpts)
		if err != nil {
			log.Fatal(err)
		}
		newerRepo, err = git.PlainClone(newerDir, false, cloneOpts)
		if err != nil {
			log.Fatal(err)
		}
	}

	olderWorktree, err := olderRepo.Worktree()
	if err != nil {
		log.Fatal(err)
	}
	err = olderWorktree.Checkout(&git.CheckoutOptions{Hash: plumbing.NewHash(*olderHashStr)})
	if err != nil {
		log.Fatal(err)
	}

	newerWorktree, err := newerRepo.Worktree()
	if err != nil {
		log.Fatal(err)
	}
	err = newerWorktree.Checkout(&git.CheckoutOptions{Hash: plumbing.NewHash(*newerHashStr)})
	if err != nil {
		log.Fatal(err)
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo,
	}

	err = os.Chdir("/")
	if err != nil {
		log.Fatal(err)
	}

	err = os.Setenv("GO111MODULE", "off")
	if err != nil {
		log.Fatal(err)
	}

	loadName := filepath.Join(parent, "top")

	err = os.Remove(loadName)
	if err != nil {
		log.Fatal(err)
	}

	err = os.Symlink(olderDir, loadName)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("xxx loading older packages\n")
	olderPkgs, err := packages.Load(cfg, "./"+loadName+"/...")
	if err != nil {
		log.Fatal(err)
	}

	err = os.Remove(loadName)
	if err != nil {
		log.Fatal(err)
	}

	err = os.Symlink(newerDir, loadName)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("xxx loading newer packages\n")
	newerPkgs, err := packages.Load(cfg, "./"+loadName+"/...")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("xxx comparing\n")
	res := modver.Compare(olderPkgs, newerPkgs, nil)
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
