package modver

import (
	"context"
	"testing"

	"github.com/bobg/modver/v2/shared"
)

func Test_useGitNative(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{
			name: "returns false when key is not set",
			ctx:  context.Background(),
			want: false,
		},
		{
			name: "returns false when key is to set to boolean false",
			ctx:  context.WithValue(context.Background(), shared.NativeGitKey, false),
			want: false,
		},
		{
			name: "returns true when key is to set to boolean true",
			ctx:  context.WithValue(context.Background(), shared.NativeGitKey, true),
			want: true,
		},
		{
			name: "returns false when key is to set to non-boolean type",
			ctx:  context.WithValue(context.Background(), shared.NativeGitKey, "wrong_type"),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := useGitNative(tt.ctx); got != tt.want {
				t.Errorf("useGitNative() = %v, want %v", got, tt.want)
			}
		})
	}
}
