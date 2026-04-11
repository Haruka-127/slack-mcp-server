package handler

import (
	"testing"

	"github.com/korotovsky/slack-mcp-server/pkg/provider"
	"github.com/stretchr/testify/assert"
)

func TestFilterChannelsByMembership(t *testing.T) {
	channels := []provider.Channel{
		{ID: "C1", Name: "#general", IsMember: true},
		{ID: "C2", Name: "#browse-only", IsMember: false},
		{ID: "C3", Name: "#private", IsPrivate: true, IsMember: true},
		{ID: "D1", Name: "@alice", IsIM: true, IsPrivate: true, IsMember: true},
		{ID: "G1", Name: "@group", IsMpIM: true, IsPrivate: true, IsMember: true},
	}

	t.Run("all returns original channels", func(t *testing.T) {
		res := filterChannelsByMembership(channels, "all")
		assert.Len(t, res, 5)
	})

	t.Run("joined returns joined channels and DMs", func(t *testing.T) {
		res := filterChannelsByMembership(channels, "joined")
		assert.Len(t, res, 4)
		ids := map[string]bool{}
		for _, ch := range res {
			ids[ch.ID] = true
		}
		assert.True(t, ids["C1"])
		assert.True(t, ids["C3"])
		assert.True(t, ids["D1"])
		assert.True(t, ids["G1"])
		assert.False(t, ids["C2"])
	})

	t.Run("not_joined_public returns only public non-memberships", func(t *testing.T) {
		res := filterChannelsByMembership(channels, "not_joined_public")
		assert.Len(t, res, 1)
		assert.Equal(t, "C2", res[0].ID)
	})

	t.Run("invalid filter falls back to original channels", func(t *testing.T) {
		res := filterChannelsByMembership(channels, "unexpected")
		assert.Len(t, res, 5)
	})
}
