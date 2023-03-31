package main

func main() {
	prURL := os.Getenv("INPUT_PULL_REQUEST_URL")
	owner, reponame, prnum, err := modver.ParsePR(prURL)
	if err != nil {
		log.Fatal(err)
	}

	ghToken := os.Getenv("GITHUB_TOKEN")
}
