# fernqclient

[![Go Reference](https://pkg.go.dev/badge/github.com/xfs0205/fernqclient.svg)](https://pkg.go.dev/github.com/xfs0205/fernqclient)
[![Go Report Card](https://goreportcard.com/badge/github.com/xfs0205/fernqclient)](https://goreportcard.com/report/github.com/xfs0205/fernqclient)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

&gt; 一个高性能 Go FernQ 客户端库，支持 P2P、广播、组播等多种消息模式

## 功能特性

- ✅ **P2P 点对点发送** - 向指定客户端发送私密消息
- ✅ **广播模式** - 向房间内所有客户端广播消息（包括自己）
- ✅ **扫描发送（组播）** - 使用正则表达式匹配目标客户端进行组播
- ✅ **自动心跳保活** - 内置心跳机制，自动维持连接
- ✅ **线程安全** - 所有操作均经过互斥锁保护，支持并发使用
- ✅ **上下文控制** - 支持通过 context 优雅关闭连接

## 安装

```bash
go get github.com/xfs0205/fernqclient
```

## 使用示例


```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/xfs0205/fernqclient"
)

func main() {
	// 创建客户端
	client := fernqclient.NewClient("my-client")
	
	// 连接服务器
	err := client.Connect("127.0.0.1:9147", "room-name", "password")
	if err != nil {
		panic(err)
	}
	defer client.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 接收消息
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-client.Read():
				if !ok {
					return
				}
				// msg.From: 发送方名称(string)
				// msg.Message: 消息内容([]byte，需string()转换)
				fmt.Printf("收到来自 %s 的消息: %s\n", msg.From, string(msg.Message))
			}
		}
	}()

	// 发送消息示例
	time.Sleep(time.Second)

	// P2P发送
	client.Send("target-client", []byte("私密消息"))

	// 广播（所有客户端都能收到，包括自己）
	client.Broadcast([]byte("大家好"))

	// 扫描发送（正则匹配目标）
	client.ScanSend("client-[0-9]+", []byte("组播消息"))

	// 阻塞等待
	sigChan := make(chan os.Signal, 1)
	fmt.Println("按 Ctrl+C 退出...")
	<-sigChan
	cancel()
}
```

---

**fernqclient** 让 FernQ 通信更简单，欢迎 Star ⭐ 和贡献代码！