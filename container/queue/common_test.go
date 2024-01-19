package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCircularLessThanU64(t *testing.T) {
	assert.True(t, circularLessThanU64(8, 9))
}
