package slsh

import (
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"
)

type converter struct {
	MessageKey   string
	LevelKey     string
	LevelMapping LevelMapping
	Extra        map[string]string
}

func NewConverter(messageKey, levelKey string, levelMapping LevelMapping, extra map[string]string) *converter {
	return &converter{
		MessageKey:   messageKey,
		LevelKey:     levelKey,
		LevelMapping: levelMapping,
		Extra:        extra,
	}
}

func (c converter) Message(entry *logrus.Entry) Message {
	contents := make(map[string]string)
	for k, v := range c.Extra {
		contents[k] = v
	}
	contents[c.MessageKey] = entry.Message
	contents[c.LevelKey] = strconv.Itoa(c.LevelMapping(entry.Level))
	for k, v := range entry.Data {
		switch v := v.(type) {
		case string:
			contents[k] = v
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64:
			contents[k] = fmt.Sprintf("%d", v)
		case float32, float64:
			contents[k] = fmt.Sprintf("%f", v)
		case bool:
			contents[k] = strconv.FormatBool(v)
		default:
			contents[k] = fmt.Sprintf("%v", v)
		}
	}

	return Message{
		Time:     entry.Time,
		Contents: contents,
	}
}
