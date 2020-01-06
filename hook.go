package slsh

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/GotaX/logrus-aliyun-log-hook/internal/validator"
)

const (
	DefaultBufferSize = 100
	DefaultMessageKey = "message"
	DefaultLevelKey   = "level"
	DefaultTimeout    = 500 * time.Millisecond
	DefaultInterval   = 3 * time.Second
)

var (
	// levels >= info will be hooked
	DefaultVisibleLevels = []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	}
	// Mapping to syslog level
	SyslogLevelMapping = func() LevelMapping {
		m := [7]int{0, 2, 3, 4, 6, 7, 8}
		return func(level logrus.Level) int { return m[level] }
	}()
)

// 日志级别映射
type LevelMapping func(level logrus.Level) int

// 日志配置
type Config struct {
	// 阿里云日志接入地址, 格式: "<region>.log.aliyuncs.com",
	// 例如: "cn-hangzhou-intranet.log.aliyuncs.com",
	// 更多接入点参考: https://help.aliyun.com/document_detail/29008.html?spm=a2c4g.11174283.6.1118.292a1caaVMpfPu
	Endpoint      string
	AccessKey     string            // 密钥对: key
	AccessSecret  string            // 密钥对: secret
	Project       string            // 日志项目名称
	Store         string            // 日志库名称
	Topic         string            // 日志 __topic__ 字段
	Source        string            // 日志 __source__ 字段, 可选, 默认为 hostname
	Extra         map[string]string // 日志附加字段, 可选
	BufferSize    int               // 本地缓存日志条数, 可选, 默认为 100
	Timeout       time.Duration     // 写缓存最大等待时间, 可选, 默认为 500ms
	Interval      time.Duration     // 缓存刷新间隔, 可选, 默认为 3s
	MessageKey    string            // 日志 Message 字段映射, 可选, 默认为 "message"
	LevelKey      string            // 日志 Level 字段映射, 可选, 默认为 "level"
	LevelMapping  LevelMapping      // 日志 Level 内容映射, 可选, 默认按照 syslog 规则映射
	VisibleLevels []logrus.Level    // 日志推送 Level, 可选, 默认推送 level >= info 的日志
	uri           *url.URL
}

func (c *Config) validate() (err error) {
	if err := validator.All(
		validator.Required("Endpoint", c.Endpoint),
		validator.Required("AccessKey", c.AccessKey),
		validator.Required("AccessSecret", c.AccessSecret),
		validator.Required("Project", c.Project),
		validator.Required("Store", c.Store),
		validator.Required("Topic", c.Topic),
	); err != nil {
		return err
	}

	source, _ := os.Hostname()
	c.Source = validator.CoalesceStr(c.Source, source)
	c.BufferSize = validator.CoalesceInt(c.BufferSize, DefaultBufferSize)
	c.MessageKey = validator.CoalesceStr(c.MessageKey, DefaultMessageKey)
	c.LevelKey = validator.CoalesceStr(c.LevelKey, DefaultLevelKey)
	c.Timeout = validator.CoalesceDur(c.Timeout, DefaultTimeout)
	c.Interval = validator.CoalesceDur(c.Interval, DefaultInterval)

	if c.LevelMapping == nil {
		c.LevelMapping = SyslogLevelMapping
	}

	if c.VisibleLevels == nil {
		c.VisibleLevels = DefaultVisibleLevels
	}

	c.uri, err = url.Parse(fmt.Sprintf(
		"http://%s.%s/logstores/%s/shards/lb", c.Project, c.Endpoint, c.Store))
	if err != nil {
		return validator.IllegalArgument("Endpoint", err.Error())
	}
	return
}

type Hook struct {
	timeout       time.Duration
	visibleLevels []logrus.Level
	writer        Writer
	converter     Converter
	service       Service
}

func New(c Config) (*Hook, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}

	writer := NewWriter(c.uri, c.Topic, c.Source, c.AccessKey, Secret(c.AccessSecret))
	service := NewService(c.BufferSize, c.Interval, writer.WriteMessage)
	converter := NewConverter(c.MessageKey, c.LevelKey, c.LevelMapping, c.Extra)
	hook := NewCustom(c.Timeout, c.VisibleLevels, converter, writer, service)
	return hook, nil
}

func NewCustom(timeout time.Duration, visibleLevels []logrus.Level,
	converter Converter, writer Writer, service Service) *Hook {
	go service.Start()

	return &Hook{
		timeout:       timeout,
		visibleLevels: visibleLevels,
		writer:        writer,
		converter:     converter,
		service:       service,
	}
}

func (h *Hook) Fire(entry *logrus.Entry) error {
	defer func() {
		if err := recover(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Hook recover from panic: %v\n", err)
		}
	}()

	ctx, _ := context.WithTimeout(context.Background(), h.timeout)
	return h.service.Push(ctx, h.converter.Message(entry))
}

func (h *Hook) Levels() []logrus.Level                 { return h.visibleLevels }
func (h *Hook) Close() error                           { return h.CloseContext(context.Background()) }
func (h *Hook) CloseContext(ctx context.Context) error { return h.service.Stop(ctx) }
