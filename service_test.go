package slsh

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestService(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		const (
			bufferSize  = 10
			deliverSize = int(1.5 * bufferSize)
		)
		cMessage := 0
		cFlush := 0

		s := NewService(bufferSize, time.Millisecond,
			func(messages ...Message) error { cMessage += len(messages); cFlush += 1; return nil })

		go s.Start()

		ctx := context.Background()
		for i := 0; i < deliverSize; i++ {
			err := s.Push(ctx, Message{})
			assert.NoError(t, err)
		}

		err := s.Stop(ctx)
		assert.NoError(t, err)
		assert.Equal(t, deliverSize, cMessage)
		assert.Equal(t, int(math.Ceil(float64(deliverSize)/float64(bufferSize))), cFlush)
	})

	t.Run("stopped", func(t *testing.T) {
		cMessage := 0
		s := NewService(1, time.Millisecond,
			func(messages ...Message) error { cMessage += len(messages); return nil })

		go s.Start()

		err := s.Stop(context.TODO())
		assert.NoError(t, err)

		err = s.Push(context.TODO(), Message{})
		assert.NoError(t, err)

		assert.Equal(t, cMessage, 0)
	})

	t.Run("push timeout", func(t *testing.T) {
		s := NewService(0, time.Millisecond,
			func(messages ...Message) error { time.Sleep(100 * time.Millisecond); return nil })

		go s.Start()

		ctx, _ := context.WithTimeout(context.TODO(), 2*time.Millisecond)
		err := s.Push(ctx, Message{})
		assert.NoError(t, err)

		err = s.Push(ctx, Message{})
		assert.Error(t, err, context.DeadlineExceeded)

		err = s.Stop(context.TODO())
		assert.NoError(t, err)
	})

	t.Run("close timeout", func(t *testing.T) {
		s := NewService(0, time.Millisecond,
			func(messages ...Message) error { time.Sleep(100 * time.Millisecond); return nil })

		go s.Start()

		err := s.Push(context.TODO(), Message{})
		assert.NoError(t, err)

		ctx, _ := context.WithTimeout(context.TODO(), 2*time.Millisecond)
		err = s.Stop(ctx)
		assert.Error(t, err, context.DeadlineExceeded)
	})

	t.Run("flush error", func(t *testing.T) {
		s := NewService(0, time.Millisecond,
			func(messages ...Message) error { return errors.New("any") })

		go s.Start()

		err := s.Push(context.TODO(), Message{})
		assert.NoError(t, err)

		err = s.Stop(context.TODO())
		assert.NoError(t, err)
	})
}
