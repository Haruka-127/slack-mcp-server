package main

import (
	"errors"
	"testing"
	"time"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

func TestCacheSyncRetryDelay(t *testing.T) {
	t.Run("uses retry-after when rate limited", func(t *testing.T) {
		err := &slack.RateLimitedError{RetryAfter: 75 * time.Second}
		assert.Equal(t, 75*time.Second, cacheSyncRetryDelay(err))
	})

	t.Run("falls back to default when retry-after is zero", func(t *testing.T) {
		err := &slack.RateLimitedError{RetryAfter: 0}
		assert.Equal(t, defaultCacheSyncRetryDelay, cacheSyncRetryDelay(err))
	})

	t.Run("falls back to default for non-rate-limit errors", func(t *testing.T) {
		err := errors.New("boom")
		assert.Equal(t, defaultCacheSyncRetryDelay, cacheSyncRetryDelay(err))
	})
}
