// Modified in the liaogx/xiaohongshu-mcp distribution; see NOTICE.

package xiaohongshu

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccessErrorClassification(t *testing.T) {
	for _, tc := range []struct {
		name  string
		probe searchPageProbe
		code  string
	}{
		{"security gate despite authenticated account", searchPageProbe{SecurityPage: true, Authenticated: true}, "SECURITY_VERIFICATION_REQUIRED"},
		{"security gate before login modal", searchPageProbe{SecurityPage: true, LoginVisible: true}, "SECURITY_VERIFICATION_REQUIRED"},
		{"login prompt", searchPageProbe{LoginVisible: true}, "AUTH_REQUIRED"},
		{"blank page is not proof of expired login", searchPageProbe{}, ""},
		{"logged in", searchPageProbe{Authenticated: true, HasSearchState: true, FeedCount: 5}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.probe.accessError()
			if tc.code == "" {
				require.NoError(t, err)
				return
			}
			var gate *PageAccessError
			require.True(t, errors.As(err, &gate))
			require.Equal(t, tc.code, gate.Code)
			require.Contains(t, err.Error(), "暂停")
		})
	}
}
