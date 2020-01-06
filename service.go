package slsh

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type service struct {
	BufferSize int
	Interval   time.Duration
	Flush      func(...Message) error
	chMessage  chan Message
	chQuit     chan struct{}
	onClose    *sync.Once
	stopped    bool
}

func NewService(bufferSize int, interval time.Duration, flush func(...Message) error) *service {
	return &service{
		BufferSize: bufferSize,
		Interval:   interval,
		Flush:      flush,
		chMessage:  make(chan Message, bufferSize),
		chQuit:     make(chan struct{}),
		onClose:    &sync.Once{},
	}
}

func (s *service) Push(ctx context.Context, message Message) error {
	if s.stopped {
		s.trace("Discard message %v", message)
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.chMessage <- message:
		return nil
	}
}

func (s *service) Start() {
	s.trace("aliyun-log-service start")
	defer s.trace("aliyun-log-service stopped")

	flushTime := time.Now()
	buffer := make([]Message, 0, s.BufferSize)

	tryFlush := func(force bool) {
		if size := len(buffer); size <= 0 ||
			!force && size < s.BufferSize && time.Since(flushTime) < s.Interval {
			return
		}

		defer func() {
			flushTime = time.Now()
			buffer = buffer[:0]
		}()

		st := time.Now()

		if err := s.Flush(buffer...); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Fail to flush logs: %v\n", err)
			return
		}

		s.trace("[%v] Flush %d logs",
			time.Since(st).Truncate(time.Millisecond), len(buffer))
	}

Loop:
	for {
		timer := time.NewTimer(s.Interval / 10)
		select {
		case <-timer.C:
		case message, ok := <-s.chMessage:
			if !ok {
				break Loop
			}
			buffer = append(buffer, message)
			tryFlush(false)
		}
		timer.Stop()
	}

	tryFlush(true)
	close(s.chQuit)
}

func (s *service) Stop(ctx context.Context) (err error) {
	s.onClose.Do(func() {
		s.stopped = true
		close(s.chMessage)
		select {
		case <-ctx.Done():
			err = ctx.Err()
		case <-s.chQuit:
		}
	})
	return
}

func (s *service) trace(message string, args ...interface{}) {
	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		log.Printf(message, args...)
	}
}
