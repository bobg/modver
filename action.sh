#!/bin/sh

go run ./cmd/modver -pr $INPUT_PULL_REQUEST_URL -token $GITHUB_TOKEN
