package slsh

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func ExampleHook() {
	hook, err := New(Config{
		Endpoint:     os.Getenv("ENDPOINT"),
		AccessKey:    os.Getenv("ACCESS_KEY"),
		AccessSecret: os.Getenv("ACCESS_SECRET"),
		Project:      os.Getenv("PROJECT"),
		Store:        os.Getenv("STORE"),
		Topic:        "demo",
		Extra:        map[string]string{"service": "demo"},
	})
	if err != nil {
		panic(err)
	}

	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetFormatter(&logrus.TextFormatter{ForceColors: true})
	logrus.AddHook(hook)

	time.AfterFunc(5*time.Second, func() { _ = hook.Close() })

	for i := 0; i < 5; i++ {
		logrus.WithField("n", i).Info("Hi!")
		time.Sleep(time.Duration(rand.Intn(3) * int(time.Second)))
	}
	// Output:
}

func TestConfig(t *testing.T) {
	raw := Config{
		Endpoint:     "http://test-project.regionid.example.com/logstores/test-logstore",
		AccessKey:    "123",
		AccessSecret: "321",
		Project:      "test-project",
		Store:        "test-store",
		Topic:        "test-topic",
		Source:       "127.0.0.1",
		Extra:        map[string]string{"k": "v"},
		BufferSize:   10,
		Timeout:      1 * time.Second,
		Interval:     1 * time.Second,
		MessageKey:   "m",
		LevelKey:     "l",
		LevelMapping: func(logrus.Level) int { return 1 },
	}

	t.Run("required", func(t *testing.T) {
		c := raw
		c.Endpoint = " "
		assert.Error(t, c.validate())

		c = raw
		c.AccessKey = " "
		assert.Error(t, c.validate())

		c = raw
		c.AccessSecret = " "
		assert.Error(t, c.validate())

		c = raw
		c.Project = " "
		assert.Error(t, c.validate())

		c = raw
		c.Store = " "
		assert.Error(t, c.validate())

		c = raw
		c.Store = " "
		assert.Error(t, c.validate())

		c = raw
		c.Topic = " "
		assert.Error(t, c.validate())
	})

	t.Run("default", func(t *testing.T) {
		c := raw
		c.Source = " "
		if assert.NoError(t, c.validate()) {
			assert.NotEmpty(t, c.Source)
		}

		c = raw
		c.Extra = nil
		if assert.NoError(t, c.validate()) {
			assert.Nil(t, c.Extra)
		}

		c = raw
		c.BufferSize = 0
		if assert.NoError(t, c.validate()) {
			assert.Equal(t, DefaultBufferSize, c.BufferSize)
		}

		c = raw
		c.Timeout = 0
		if assert.NoError(t, c.validate()) {
			assert.Equal(t, DefaultTimeout, c.Timeout)
		}

		c = raw
		c.Interval = 0
		if assert.NoError(t, c.validate()) {
			assert.Equal(t, DefaultInterval, c.Interval)
		}

		c = raw
		c.MessageKey = ""
		if assert.NoError(t, c.validate()) {
			assert.Equal(t, DefaultMessageKey, c.MessageKey)
		}

		c = raw
		c.LevelKey = ""
		if assert.NoError(t, c.validate()) {
			assert.Equal(t, DefaultLevelKey, c.LevelKey)
		}

		c = raw
		c.LevelMapping = nil
		if assert.NoError(t, c.validate()) {
			assert.Equal(t, fmt.Sprintf("%p", SyslogLevelMapping), fmt.Sprintf("%p", c.LevelMapping))
		}

		c = raw
		c.VisibleLevels = nil
		if assert.NoError(t, c.validate()) {
			assert.Equal(t, DefaultVisibleLevels, c.VisibleLevels)
		}

		c = raw
		c.HttpClient = nil
		if assert.NoError(t, c.validate()) {
			assert.Equal(t, http.DefaultClient, c.HttpClient)
		}
	})
}

func TestHook(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		ops := make([]string, 0)

		rawMsg := Message{
			Time:     time.Now(),
			Contents: map[string]string{"service": "any"},
		}
		writer := &MockWriter{
			onWriteMessage: func(messages ...Message) error {
				ops = append(ops, "writer.onWriteMessage")
				assert.Equal(t, messages[0], rawMsg)
				return nil
			},
		}
		converter := &MockConverter{
			onMessage: func(entry *logrus.Entry) Message {
				ops = append(ops, "converter.onMessage")
				return rawMsg
			},
		}
		service := &MockService{
			onPush: func(ctx context.Context, message Message) error {
				ops = append(ops, "service.onPush")
				assert.Equal(t, rawMsg, message)
				return nil
			},
			onStart: func() {
				ops = append(ops, "service.onStart")
				_ = writer.WriteMessage(rawMsg)
			},
			onStop: func(ctx context.Context) error {
				ops = append(ops, "service.onStop")
				dl, ok := ctx.Deadline()
				if assert.True(t, ok) {
					assert.NotEmpty(t, dl)
				}
				return nil
			},
		}

		hook := NewCustom(DefaultTimeout, DefaultVisibleLevels, converter, writer, service)

		logger := logrus.New()
		logger.AddHook(hook)

		logger.Info("Hi")

		ctx, _ := context.WithTimeout(context.TODO(), time.Second)
		err := hook.CloseContext(ctx)
		assert.NoError(t, err)

		assert.Len(t, ops, 5)
	})

	t.Run("panic", func(t *testing.T) {
		counter := 0
		writer := &MockWriter{
			onWriteMessage: func(messages ...Message) error { return nil },
		}
		converter := &MockConverter{
			onMessage: func(entry *logrus.Entry) Message { return Message{} },
		}
		service := &MockService{
			onPush:  func(ctx context.Context, message Message) error { counter++; panic("no") },
			onStart: func() {},
			onStop:  func(ctx context.Context) error { return nil },
		}

		hook := NewCustom(DefaultTimeout, DefaultVisibleLevels, converter, writer, service)
		logger := logrus.New()
		logger.AddHook(hook)
		logger.Info("Hi")
		err := hook.CloseContext(context.TODO())
		assert.NoError(t, err)
		assert.Equal(t, 1, counter)
	})
}

type MockService struct {
	onPush  func(ctx context.Context, message Message) error
	onStart func()
	onStop  func(ctx context.Context) error
}

func (s MockService) Push(ctx context.Context, message Message) error { return s.onPush(ctx, message) }
func (s MockService) Start()                                          { s.onStart() }
func (s MockService) Stop(ctx context.Context) error                  { return s.onStop(ctx) }

type MockWriter struct {
	onWriteMessage func(messages ...Message) error
}

func (w MockWriter) WriteMessage(messages ...Message) error { return w.onWriteMessage(messages...) }

type MockConverter struct {
	onMessage func(entry *logrus.Entry) Message
}

func (c MockConverter) Message(entry *logrus.Entry) Message { return c.onMessage(entry) }
