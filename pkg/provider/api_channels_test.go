package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

// TestChannelTypeFiltering verifies the channel type classification logic used
// in GetChannels to filter the snapshot by requested types.
func TestChannelTypeFiltering(t *testing.T) {
	channels := map[string]Channel{
		"C1": {ID: "C1", Name: "#general", IsPrivate: false, IsIM: false, IsMpIM: false, IsMember: true},
		"C2": {ID: "C2", Name: "#secret", IsPrivate: true, IsIM: false, IsMpIM: false, IsMember: true},
		"D1": {ID: "D1", Name: "@alice", IsPrivate: true, IsIM: true, IsMpIM: false, IsMember: true},
		"G1": {ID: "G1", Name: "mpdm-a-b-c", IsPrivate: true, IsIM: false, IsMpIM: true, IsMember: true},
		"C3": {ID: "C3", Name: "#browse-only", IsPrivate: false, IsIM: false, IsMpIM: false, IsMember: false},
	}

	// This mirrors the filtering logic in GetChannels
	filterByTypes := func(channels map[string]Channel, types []string) []Channel {
		var res []Channel
		for _, t := range types {
			for _, ch := range channels {
				switch {
				case t == "public_channel" && !ch.IsPrivate && !ch.IsIM && !ch.IsMpIM:
					res = append(res, ch)
				case t == "private_channel" && ch.IsPrivate && !ch.IsIM && !ch.IsMpIM:
					res = append(res, ch)
				case t == "im" && ch.IsIM:
					res = append(res, ch)
				case t == "mpim" && ch.IsMpIM:
					res = append(res, ch)
				}
			}
		}
		return res
	}

	t.Run("public_channel only", func(t *testing.T) {
		res := filterByTypes(channels, []string{"public_channel"})
		assert.Len(t, res, 2)
		ids := map[string]bool{}
		for _, ch := range res {
			ids[ch.ID] = true
		}
		assert.True(t, ids["C1"])
		assert.True(t, ids["C3"])
	})

	t.Run("private_channel only", func(t *testing.T) {
		res := filterByTypes(channels, []string{"private_channel"})
		assert.Len(t, res, 1)
		assert.Equal(t, "C2", res[0].ID)
	})

	t.Run("im only", func(t *testing.T) {
		res := filterByTypes(channels, []string{"im"})
		assert.Len(t, res, 1)
		assert.Equal(t, "D1", res[0].ID)
	})

	t.Run("mpim only", func(t *testing.T) {
		res := filterByTypes(channels, []string{"mpim"})
		assert.Len(t, res, 1)
		assert.Equal(t, "G1", res[0].ID)
	})

	t.Run("all types returns all channels", func(t *testing.T) {
		res := filterByTypes(channels, AllChanTypes)
		assert.Len(t, res, 5)
	})

	t.Run("multiple types", func(t *testing.T) {
		res := filterByTypes(channels, []string{"public_channel", "im"})
		assert.Len(t, res, 3)
		ids := map[string]bool{}
		for _, ch := range res {
			ids[ch.ID] = true
		}
		assert.True(t, ids["C1"], "should include public channel")
		assert.True(t, ids["C3"], "should include public channel not joined")
		assert.True(t, ids["D1"], "should include IM")
	})

	t.Run("im is not classified as private_channel", func(t *testing.T) {
		// IMs have IsPrivate=true but should only match "im", not "private_channel"
		res := filterByTypes(channels, []string{"private_channel"})
		for _, ch := range res {
			assert.False(t, ch.IsIM, "IMs should not appear in private_channel results")
		}
	})

	t.Run("mpim is not classified as private_channel", func(t *testing.T) {
		// MPIMs have IsPrivate=true but should only match "mpim", not "private_channel"
		res := filterByTypes(channels, []string{"private_channel"})
		for _, ch := range res {
			assert.False(t, ch.IsMpIM, "MPIMs should not appear in private_channel results")
		}
	})
}

func TestMapChannelPreservesMembership(t *testing.T) {
	ch := mapChannel(
		"C1",
		"general",
		"general",
		"topic",
		"purpose",
		"",
		nil,
		42,
		true,
		false,
		false,
		false,
		false,
		nil,
	)

	assert.True(t, ch.IsMember)
	assert.Equal(t, "#general", ch.Name)
}

func TestMapChannelNormalizesJoinedOnlyMembership(t *testing.T) {
	cases := []struct {
		name      string
		isIM      bool
		isMpIM    bool
		isPrivate bool
		nameNorm  string
	}{
		{name: "private channel", isPrivate: true, nameNorm: "secret"},
		{name: "im", isIM: true, isPrivate: true},
		{name: "mpim", isMpIM: true, isPrivate: true, nameNorm: "group-dm"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ch := mapChannel(
				"C1",
				tc.nameNorm,
				tc.nameNorm,
				"topic",
				"purpose",
				"U1",
				[]string{"U1", "U2"},
				2,
				false,
				tc.isIM,
				tc.isMpIM,
				tc.isPrivate,
				false,
				nil,
			)

			assert.True(t, ch.IsMember)
		})
	}
}

// TestAllChanTypesConstant verifies the expected channel types are defined.
func TestAllChanTypesConstant(t *testing.T) {
	assert.Len(t, AllChanTypes, 4, "should have 4 channel types")
	assert.Contains(t, AllChanTypes, "public_channel")
	assert.Contains(t, AllChanTypes, "private_channel")
	assert.Contains(t, AllChanTypes, "im")
	assert.Contains(t, AllChanTypes, "mpim")
}

func TestChannelsRateLimitRetryDelay(t *testing.T) {
	t.Run("uses RetryAfter when present", func(t *testing.T) {
		retryAfter, ok := channelsRateLimitRetryDelay(&slack.RateLimitedError{RetryAfter: 45 * time.Second})
		assert.True(t, ok)
		assert.Equal(t, 45*time.Second, retryAfter)
	})

	t.Run("uses default when RetryAfter is missing", func(t *testing.T) {
		retryAfter, ok := channelsRateLimitRetryDelay(&slack.RateLimitedError{})
		assert.True(t, ok)
		assert.Equal(t, defaultChannelsRateLimitRetryDelay, retryAfter)
	})

	t.Run("non rate limit error", func(t *testing.T) {
		retryAfter, ok := channelsRateLimitRetryDelay(errors.New("boom"))
		assert.False(t, ok)
		assert.Zero(t, retryAfter)
	})
}

func TestWaitWithContext(t *testing.T) {
	t.Run("returns context error when canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := waitWithContext(ctx, time.Second)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("returns nil after delay", func(t *testing.T) {
		err := waitWithContext(context.Background(), 5*time.Millisecond)
		assert.NoError(t, err)
	})
}
