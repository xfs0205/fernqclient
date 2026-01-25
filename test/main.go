package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/xfs0205/fernqclient"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	client1 := fernqclient.NewClient("test-client1")
	err := client1.Connect("127.0.0.1:9147", "test", "123456")
	if err != nil {
		panic(err)
	}
	defer client1.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-client1.Read():
				if !ok {
					return
				}
				fmt.Println("client1收到来自", msg.From, "信息：", string(msg.Message))
			}
		}
	}()

	client2 := fernqclient.NewClient("test-client2")
	err = client2.Connect("127.0.0.1:9147", "test", "123456")
	if err != nil {
		panic(err)
	}
	defer client2.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-client2.Read():
				if !ok {
					return
				}
				fmt.Println("client2收到来自", msg.From, "信息：", string(msg.Message))
			}
		}
	}()

	client3 := fernqclient.NewClient("test-client3")
	err = client3.Connect("127.0.0.1:9147", "test", "123456")
	if err != nil {
		panic(err)
	}
	defer client3.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-client3.Read():
				if !ok {
					return
				}
				fmt.Println("client3收到来自", msg.From, "信息：", string(msg.Message))
			}
		}
	}()

	// 发送信息，测试功能的协程
	go func() {
		testCount := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second * 5):
				testCount++
				switch testCount % 3 {
				case 1:
					// 测试 Send P2P模式：client1 发送给 client2
					fmt.Println("\n[测试 Send P2P模式] client1 -> client2")
					msg := fmt.Sprintf("这是P2P消息 #%d", testCount)
					if err := client1.Send("test-client2", []byte(msg)); err != nil {
						fmt.Println("Send 失败:", err)
					} else {
						fmt.Println("Send 成功:", msg)
					}

				case 2:
					// 测试 Broadcast 广播模式：client2 广播给房间内所有客户端
					fmt.Println("\n[测试 Broadcast 广播模式] client2 -> 所有客户端(包括自己)")
					msg := fmt.Sprintf("这是广播消息 #%d", testCount)
					if err := client2.Broadcast([]byte(msg)); err != nil {
						fmt.Println("Broadcast 失败:", err)
					} else {
						fmt.Println("Broadcast 成功:", msg)
					}

				case 0:
					// 测试 ScanSend 扫描发送模式：client3 发送给匹配正则的客户端
					fmt.Println("\n[测试 ScanSend 扫描发送模式] client3 -> 匹配 'test-client[12]' 的客户端")
					msg := fmt.Sprintf("这是扫描发送消息 #%d", testCount)
					// 正则表达式匹配 client1 和 client2（不包含 client3 自己）
					if err := client3.ScanSend("test-client[12]", []byte(msg)); err != nil {
						fmt.Println("ScanSend 失败:", err)
					} else {
						fmt.Println("ScanSend 成功:", msg)
					}
				}
			}
		}
	}()

	// 堵塞，监听退出信号
	sigChan := make(chan os.Signal, 1)
	fmt.Println("服务已启动，按 Ctrl+C 退出...")
	<-sigChan
	fmt.Println("\n收到退出信号，正在关闭...")
	cancel()
}
