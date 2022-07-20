package emby

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDiscoverEmbyServer(t *testing.T) {
	_, err := DiscoverEmbyServer()
	require.NoError(t, err)
}
