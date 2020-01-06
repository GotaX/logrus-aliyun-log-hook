package slsh

import (
	"context"
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"
)

type Secret []byte

func (s Secret) String() string { return "******" }

type AliyunError struct {
	HTTPCode  int32  `json:"-"`
	Code      string `json:"errorCode"`
	Message   string `json:"errorMessage"`
	RequestID string `json:"-"`
}

func (a AliyunError) Error() string {
	if data, err := json.Marshal(a); err != nil {
		return err.Error()
	} else {
		return string(data)
	}
}

type Message struct {
	Time     time.Time
	Contents map[string]string
}

type Writer interface {
	WriteMessage(messages ...Message) error
}

type Service interface {
	Push(ctx context.Context, message Message) error
	Start()
	Stop(ctx context.Context) error
}

type Converter interface {
	Message(entry *logrus.Entry) Message
}
