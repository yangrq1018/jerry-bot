package gcloud

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadStoragePreference(t *testing.T) {
	require.NoError(t, SaveObject("object1", "hello,world;hello,world"))
}
