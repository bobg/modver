package modver

import (
	"fmt"
	"go/types"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestIdenticalArray(t *testing.T) {
	err := withPackage(func(pkg *packages.Package) error {
		var (
			scope          = pkg.Types.Scope()
			resultType     = scope.Lookup("Result").Type()
			resultCodeType = scope.Lookup("ResultCode").Type()
		)

		var (
			a1 = types.NewArray(resultType, 7)
			a2 = types.NewArray(resultType, 7)
			a3 = types.NewArray(resultType, 11)
			a4 = types.NewArray(resultCodeType, 7)
		)

		cases := []struct {
			// t1 is always a1
			t2   *types.Array
			want bool
		}{{
			t2:   a2,
			want: true,
		}, {
			t2:   a3,
			want: false,
		}, {
			t2:   a4,
			want: false,
		}}

		for i, tc := range cases {
			t.Run(fmt.Sprintf("case_%d", i+1), func(t *testing.T) {
				c := newComparer()
				if got := c.identical(a1, tc.t2); got != tc.want {
					t.Errorf("got %v, want %v", got, tc.want)
				}
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIdenticalChan(t *testing.T) {
	err := withPackage(func(pkg *packages.Package) error {
		var (
			scope          = pkg.Types.Scope()
			resultType     = scope.Lookup("Result").Type()
			resultCodeType = scope.Lookup("ResultCode").Type()
		)

		chans := []*types.Chan{
			types.NewChan(types.SendRecv, resultType),
			types.NewChan(types.SendOnly, resultType),
			types.NewChan(types.RecvOnly, resultType),

			types.NewChan(types.SendRecv, resultCodeType),
			types.NewChan(types.SendOnly, resultCodeType),
			types.NewChan(types.RecvOnly, resultCodeType),
		}

		for i := 0; i < len(chans); i++ {
			for j := i; j < len(chans); j++ {
				c := newComparer()
				if got := c.identical(chans[i], chans[j]); got != (i == j) {
					t.Errorf("case %d/%d: got %v, want %v", i, j, got, i == j)
				}
			}
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
