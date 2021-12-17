#  Aliyun Log Hook for [Logrus](https://github.com/sirupsen/logrus)

[![godoc reference](https://godoc.org/github.com/GotaX/logrus-aliyun-log-hook?status.svg)](https://godoc.org/github.com/GotaX/logrus-aliyun-log-hook)

此 Hook 用于将通过 logrus 记录的日志发送到阿里云日志服务. 

特点:

- 采用非阻塞设计, 由一个后台线程将日志批量刷到远端日志库.
- 采用轻量级设计, 直接使用 [PutLogs](https://help.aliyun.com/document_detail/29026.html) 接口, 不依赖于 `github.com/aliyun/aliyun-log-go-sdk`
- 内存占用较低, 大约是直接使用 sdk 的 70%

## 安装

`go get -u github.com/GotaX/logrus-aliyun-log-hook`

## 使用指南

```go
package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/GotaX/logrus-aliyun-log-hook"
)

func main() {
	hook, err := slsh.New(slsh.Config{
		Endpoint:     os.Getenv("ENDPOINT"),                // 接入点, 例如: "cn-hangzhou-intranet.log.aliyuncs.com",
		AccessKey:    os.Getenv("ACCESS_KEY"),              // 授权密钥对: key
		AccessSecret: os.Getenv("ACCESS_SECRET"),           // 授权密钥对: secret
		Project:      os.Getenv("PROJECT"),                 // 日志项目名称
		Store:        os.Getenv("STORE"),                   // 日志库名称
		Topic:        "demo",                               // 日志 __topic__ 字段
		Extra:        map[string]string{"service": "demo"}, // 日志附加字段, 可选
		// 更多配置说明, 参考字段注释
	})
	if err != nil {
		panic(err)
	}

	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetFormatter(&logrus.TextFormatter{ForceColors: true})
	logrus.AddHook(hook)

	// 加上这行关闭本地日志输出, 仅写入阿里云日志
	// logrus.SetOutput(ioutil.Discard)

	time.AfterFunc(5*time.Second, func() { _ = hook.Close() })

	for i := 0; i < 10; i++ {
		logrus.WithField("n", i).Info("Hi!")
		time.Sleep(time.Duration(rand.Intn(3) * int(time.Second)))
	}
}

```

## Benchmark

I/O 部分对比, 配置: Intel(R) Core(TM) i7-8700 CPU @ 3.20GHz

`go test -run ^BeachmarkWriter$ -bench=BenchmarkWriter -count 5 -benchmem `

| 名称    | CPU/op     | alloc/op    | allocs/op |
| ------- | ---------- | ----------- | --------- |
| hook    | 110µs ± 1% | 9.51kB ± 0% | 135 ± 0%  |
| sls-sdk | 127µs ± 3% | 13.4kB ± 0% | 165 ± 0%  |

## 外部依赖

```
.
  ├ github.com/golang/protobuf/proto
  ├ github.com/pierrec/lz4
  └ github.com/sirupsen/logrus
```

