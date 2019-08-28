// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package counter

import (
	"os"
	"testing"
)

func TestAnnotate(t *testing.T) {
	type args struct {
		name   string
		output string
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		//{"1", args{"./testdata/d1.go", ""}, 2},
		{"2", args{"./testdata/d2.go", "./testdata/aaa.txt"}, 15},
		// {"3", args{"./testdata/d3.go", ""}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Remove(tt.args.output)
			if got := Annotate(tt.args.name, tt.args.output); got != tt.want {
				t.Errorf("Annotate() = %v, want %v", got, tt.want)
			}
		})
	}
}
