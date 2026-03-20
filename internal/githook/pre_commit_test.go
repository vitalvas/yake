package githook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunPreCommit(t *testing.T) {
	err := RunPreCommit()
	assert.NoError(t, err)
}
