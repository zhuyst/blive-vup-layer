package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsRepeatedChar(t *testing.T) {
	assert.Equal(t, true, IsRepeatedChar("哈哈哈哈"))
}
