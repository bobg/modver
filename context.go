package modver

import "context"

type (
	gitNativeKeyType struct{}
)

// WithGit decorates a context with the value of the gitPath string.
// Retrieve it with GetGit.
func WithGit(ctx context.Context, gitPath string) context.Context {
	return context.WithValue(ctx, gitNativeKeyType{}, gitPath)
}

// GetGit returns the value of the gitPath string added to `ctx` with WithGit
// If the key is not set the default value is an empty string.
func GetGit(ctx context.Context) string {
	val, _ := ctx.Value(gitNativeKeyType{}).(string)
	return val
}
