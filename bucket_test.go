package bucket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenBucket(t *testing.T) {
	assert := assert.New(t)

	t.Run("Should init the bucket with full available tokens", func(t *testing.T) {
		var cap int64 = 100

		b := New(time.Minute, cap)
		defer b.Destory()

		assert.Equal(cap, b.Capability())
		assert.Equal(cap, b.avail)
	})

	t.Run("Should panic when interval and cap is negative", func(t *testing.T) {
		assert.Panics(func() { New(time.Minute, -1) })
		assert.Panics(func() { New(-time.Minute, 1) })
		assert.Panics(func() { New(-time.Minute, -1) })
	})

	t.Run("Should panic the count pass to checkCount is negative", func(t *testing.T) {
		b := New(time.Minute, 1)
		defer b.Destory()

		assert.Panics(func() { b.checkCount(-1) })
	})

	t.Run("Should panic the count pass to checkCount is greater than cap", func(t *testing.T) {
		b := New(time.Minute, 1)
		defer b.Destory()

		assert.Panics(func() { b.checkCount(2) })
	})

	t.Run("Should return true when tryTake is succsessful", func(t *testing.T) {
		b := New(time.Minute, 5)
		defer b.Destory()

		assert.True(b.TryTake(1))
		assert.Equal(int64(4), b.avail)
	})

	t.Run("Should return false when tryTake is failed", func(t *testing.T) {
		b := New(time.Minute, 1)
		defer b.Destory()

		assert.True(b.TryTake(1))
		assert.False(b.TryTake(1))
		assert.Equal(int64(0), b.avail)
	})
}
