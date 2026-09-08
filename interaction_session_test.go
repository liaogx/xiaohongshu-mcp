// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInteractionPageCache(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cookies.json")
	t.Setenv("COOKIES_PATH", path)
	require.NoError(t, os.WriteFile(path, []byte("fixture-session-A"), 0600))
	digest, err := credentialDigest()
	require.NoError(t, err)
	var cache interactionPageCache
	closed := 0
	closePage := func() { closed++ }
	cache.mu.Lock()
	defer cache.mu.Unlock()
	defer cache.clear()
	cache.keep("fixture-note", nil, closePage, digest)
	entry := cache.take("fixture-note")
	require.NotNil(t, entry)
	require.Equal(t, 0, closed)
	entry.close()
	require.Equal(t, 1, closed)
	require.Nil(t, cache.take("fixture-note"))
	cache.keep("fixture-note", nil, closePage, digest)
	require.Nil(t, cache.take("another-note"))
	require.Equal(t, 2, closed)
	cache.keep("fixture-note", nil, closePage, digest)
	require.NoError(t, os.WriteFile(path, []byte("fixture-session-B"), 0600))
	require.Nil(t, cache.take("fixture-note"))
	require.Equal(t, 3, closed)
	cache.keep("fixture-note", nil, closePage, digest)
	cache.keep("another-note", nil, closePage, digest)
	require.Equal(t, 4, closed)
}
