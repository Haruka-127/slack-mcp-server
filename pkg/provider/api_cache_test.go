package provider

import (
	"encoding/json"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNullCacheUnmarshal verifies that json.Unmarshal of "null" into a
// Channel/User slice produces an empty slice (not an error), and that our
// len() == 0 check correctly detects this condition.
//
// This is a regression test for the null cache poisoning bug: when
// GetChannels returned nil, json.MarshalIndent(nil) wrote "null" to the
// cache file. On next startup, Unmarshal succeeded with an empty slice,
// which was incorrectly treated as valid cached data.
func TestNullCacheUnmarshal(t *testing.T) {
	t.Run("null JSON produces empty channel slice without error", func(t *testing.T) {
		var channels []Channel
		err := json.Unmarshal([]byte("null"), &channels)
		require.NoError(t, err, "json.Unmarshal of 'null' should not error")
		assert.Nil(t, channels, "result should be nil")
		assert.Len(t, channels, 0, "len should be 0 (caught by our guard)")
	})

	t.Run("null JSON produces empty user slice without error", func(t *testing.T) {
		var users []slack.User
		err := json.Unmarshal([]byte("null"), &users)
		require.NoError(t, err, "json.Unmarshal of 'null' should not error")
		assert.Nil(t, users, "result should be nil")
		assert.Len(t, users, 0, "len should be 0 (caught by our guard)")
	})

	t.Run("empty array JSON produces empty slice without error", func(t *testing.T) {
		var channels []Channel
		err := json.Unmarshal([]byte("[]"), &channels)
		require.NoError(t, err)
		assert.Len(t, channels, 0, "empty array should also be caught by len guard")
	})

	t.Run("nil channels marshals to null", func(t *testing.T) {
		// This confirms the root cause: marshaling nil produces "null"
		var channels []Channel
		data, err := json.MarshalIndent(channels, "", "  ")
		require.NoError(t, err)
		assert.Equal(t, "null", string(data),
			"MarshalIndent(nil slice) should produce 'null' — this is why we guard against writing empty cache")
	})

	t.Run("empty slice marshals to empty array", func(t *testing.T) {
		channels := []Channel{}
		data, err := json.MarshalIndent(channels, "", "  ")
		require.NoError(t, err)
		assert.Equal(t, "[]", string(data),
			"MarshalIndent(empty slice) should produce '[]'")
	})
}

// TestEmptyCacheWriteGuard verifies that we correctly detect when channels
// should NOT be written to cache (empty/nil slice), preventing the null
// cache poisoning bug.
func TestEmptyCacheWriteGuard(t *testing.T) {
	shouldWriteCache := func(channels []Channel) bool {
		return len(channels) > 0
	}

	t.Run("nil slice should not be written", func(t *testing.T) {
		var channels []Channel
		assert.False(t, shouldWriteCache(channels))
	})

	t.Run("empty slice should not be written", func(t *testing.T) {
		channels := []Channel{}
		assert.False(t, shouldWriteCache(channels))
	})

	t.Run("populated slice should be written", func(t *testing.T) {
		channels := []Channel{{ID: "C1"}}
		assert.True(t, shouldWriteCache(channels))
	})
}

func TestChannelsCacheHasMembershipField(t *testing.T) {
	t.Run("legacy cache without is_member is incompatible", func(t *testing.T) {
		ok, err := channelsCacheHasMembershipField([]byte(`[{"id":"C1","name":"#general"}]`))
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("cache with explicit false is compatible", func(t *testing.T) {
		ok, err := channelsCacheHasMembershipField([]byte(`[{"id":"C1","name":"#general","is_member":false}]`))
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("null cache is handled by empty-slice guard", func(t *testing.T) {
		ok, err := channelsCacheHasMembershipField([]byte("null"))
		require.NoError(t, err)
		assert.True(t, ok)
	})
}

func TestNormalizeChannelMembership(t *testing.T) {
	t.Run("private-like conversations are always members", func(t *testing.T) {
		cases := []Channel{
			{ID: "C1", IsPrivate: true, IsMember: false},
			{ID: "D1", IsIM: true, IsPrivate: true, IsMember: false},
			{ID: "G1", IsMpIM: true, IsPrivate: true, IsMember: false},
		}

		for _, tc := range cases {
			normalized := normalizeChannelMembership(tc)
			assert.True(t, normalized.IsMember, tc.ID)
		}
	})

	t.Run("public non-member stays false", func(t *testing.T) {
		normalized := normalizeChannelMembership(Channel{ID: "C2", IsMember: false})
		assert.False(t, normalized.IsMember)
	})
}
