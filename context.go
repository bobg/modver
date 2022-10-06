package modver

import "context"

type (
	gitKeyType struct{}
)

// WithGit decorates a context with the value of the gitPath string.
// This is the path of an executable to use for Git operations in calls to CompareGit.
// Without it, the go-git library is used.
// (But a git program is preferable.)
// Retrieve it with GetGit.
func WithGit(ctx context.Context, gitPath string) context.Context {
	return context.WithValue(ctx, gitKeyType{}, gitPath)
}

// GetGit returns the value of the gitPath string added to `ctx` with WithGit.
// If the key is not set the default value is an empty string.
func GetGit(ctx context.Context) string {
	val, _ := ctx.Value(gitKeyType{}).(string)
	return val
}
