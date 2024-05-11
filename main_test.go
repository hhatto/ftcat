package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsMatchFileName(t *testing.T) {
	cases := []struct {
		name     string
		src, dst string
		expected bool
	}{
		{name: "same", src: "test.go", dst: "test.go", expected: true},
		{name: "with prefix dot slash", src: "test.go", dst: "./test.go", expected: true},
		{name: "same with directory", src: "dir/test.go", dst: "dir/test.go", expected: true},
		{name: "same with directory and prefix dot slash", src: "dir/test.go", dst: "./dir/test.go", expected: true},
		{name: "absolute path", src: "/path/to/file.go", dst: "/path/to/file.go", expected: true},
		{name: "not match", src: "test.go", dst: "./dtest.go", expected: false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isMatchFileName(tt.src, tt.dst))
		})
	}
}
