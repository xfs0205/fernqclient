package fernqclient

import (
	"context"
	"fmt"
	"log"
	"net"
	"regexp"
	"sync"
	"time"

	"github.com/xfs0205/fernqclient/codec"
)

// 向外界暴露接收的消息类型结构体
type FernqMessage struct {
	From    string
	Message []byte
}

// 客户端
type Client struct {
	ClientName string // 客户端名称

	wg       sync.WaitGroup     // 等待组
	ctx      context.Context    // 上下文
	cancel   context.CancelFunc // 取消函数
	readChan chan FernqMessage  // 读取通道

	conn    net.Conn   // TCP连接
	writeMu sync.Mutex // 写操作互斥锁

	isConnected bool       // 是否已连接
	statusMu    sync.Mutex // 状态访问互斥锁

	connMu sync.Mutex // 连接访问互斥锁
}

// 安全发送信息
func (c *Client) safeWrite(data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("未连接")
	}
	_, err := c.conn.Write(data)
	return err
}

// 读取信息协程
func (c *Client) readLoop(xxbuff []byte) {
	c.wg.Add(1)
	go func() {
		defer func() {
			c.writeMu.Lock()
			if c.conn != nil {
				c.conn.Close()
				c.conn = nil
			}
			c.writeMu.Unlock()

			c.statusMu.Lock()
			c.isConnected = false
			c.statusMu.Unlock()
			c.wg.Done()

			// 关闭输出通道
			close(c.readChan)
			c.readChan = nil
		}()
		for {
			select {
			case <-c.ctx.Done():
				return
			default:
			}
			buff := make([]byte, 1024)
			// 设置读取超时时间
			if err := c.conn.SetReadDeadline(time.Now().Add(time.Second * 5)); err != nil {
				continue
			}
			n, err := c.conn.Read(buff)
			if err != nil {
				// 检查是否为超时错误
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// log.Println("读取超时，重新设置超时并继续等待...")
					continue // 超时后重新循环，不执行下面的数据处理
				}
				return
			}

			// 拼接数据
			xxbuff = append(xxbuff, buff[:n]...)

			// 循环处理数据
			for {
				msgType, body, remain, err := codec.Decode(xxbuff)
				if err != nil {
					if err == codec.ErrLength {
						break
					}
					return
				}

				// 将剩余数据保存起来
				xxbuff = remain

				// 如果数据类型为心跳
				if msgType == codec.TypePong || msgType == codec.TypePing {
					// log.Println("收到心跳包")
					// 创建并发送pong
					pong, err := codec.CreatePong()
					if err != nil {
						log.Println("创建pong失败")
						continue
					}
					if err := c.safeWrite(pong); err != nil {
						log.Println("发送pong失败")
						continue
					}
					continue
				}

				// 解析数据
				message, err := codec.DecodeReceiveMessagePB(body)
				if err != nil {
					log.Println("解析数据失败")
					continue
				}
				// 添加到输出通道
				c.readChan <- FernqMessage{
					From:    message.From,
					Message: message.Message,
				}
			}
		}
	}()
}

// Send P2P模式，发送消息到指定目标
// 参数:
//   - to: 目标标识
//   - message: 消息内容
//
// 返回值:
//   - error: 发送过程中的错误
func (c *Client) Send(to string, message []byte) error {
	// 点对点发送：to为目标客户端名称
	data, err := codec.CreateP2PRelay(c.ClientName, to, message)
	if err != nil {
		return fmt.Errorf("创建P2P消息失败: %w", err)
	}
	return c.safeWrite(data)
}

// Broadcast 广播模式，将消息发送给房间内所有客户端，包括自己
// 参数:
//   - message: 消息内容
//
// 返回值:
//   - error: 发送过程中的错误
func (c *Client) Broadcast(message []byte) error {
	data, err := codec.CreateRoomBroadcast(c.ClientName, "room", message)
	if err != nil {
		return fmt.Errorf("创建广播消息失败: %w", err)
	}
	return c.safeWrite(data)
}

// ScanSend 扫描发送模式(属于组播模式)，发送消息给指定正则表达式匹配的用户
// 参数:
//   - to: 用于匹配目标用户的正则表达式
//   - message: 消息内容
//
// 返回值:
//   - error: 发送过程中的错误（包括正则表达式无效或消息创建失败）
func (c *Client) ScanSend(to string, message []byte) error {
	// 验证正则表达式有效性
	if _, err := regexp.Compile(to); err != nil {
		return fmt.Errorf("无效的正则表达式 '%s': %w", to, err)
	}

	data, err := codec.CreateUserScan(c.ClientName, to, message)
	if err != nil {
		return fmt.Errorf("创建扫描发送消息失败: %w", err)
	}
	return c.safeWrite(data)
}

// ScanOnlySend 扫描发送模式(属于单播模式)，发送消息给指定正则表达式匹配的用户中的随机一个
func (c *Client) ScanOnlySend(to string, message []byte) error {
	// 验证正则表达式有效性
	if _, err := regexp.Compile(to); err != nil {
		return fmt.Errorf("无效的正则表达式 '%s': %w", to, err)
	}

	data, err := codec.CreateUserScanSingle(c.ClientName, to, message)
	if err != nil {
		return fmt.Errorf("创建扫描发送消息失败: %w", err)
	}
	return c.safeWrite(data)
}

// Read 返回一个只读通道，用于接收来自服务器转发的消息
//
// 返回:
//   - <-chan FernqMessage: 只读消息通道，通过 range 或 select 读取消息
//
// 使用方式:
//
//	通过 range 遍历读取（阻塞等待）:
//	  for msg := range client.Read() {
//	      fmt.Printf("收到来自 %s 的消息: %s\n", msg.From, string(msg.Message))
//	  }
//
//	通过 select 非阻塞读取:
//	  select {
//	  case msg, ok := <-client.Read():
//	      if !ok {
//	          // 通道已关闭，连接断开
//	          return
//	      }
//	      fmt.Printf("收到来自 %s 的消息: %s\n", msg.From, string(msg.Message))
//	  default:
//	      // 无消息时执行其他逻辑
//	  }
//
// FernqMessage 字段说明:
//   - From:    string 类型，表示发送方的客户端名称
//   - Message: []byte 类型，原始消息内容字节数组，可根据业务需求转换为 string 或其他格式
//
// 注意事项:
//   - 通道在连接断开或调用 Stop() 后会被关闭，读取时需注意判断通道是否关闭（ok 值）
//   - 该方法是线程安全的，可在多个 goroutine 中同时读取（但通常建议单 goroutine 消费）
//   - Message 为原始字节数组，如需字符串形式需手动转换: string(msg.Message)
func (c *Client) Read() <-chan FernqMessage {
	return c.readChan
}

// Connect 连接服务器
//
// 参数:
//   - FQC: 服务器连接地址，fernq URL 格式
//
// URL 格式: fernq://[用户名@]主机[:端口]/UUID#房间名[?room_pass=密码]
//
// 示例:
//
//	// 本地测试（IP + 默认端口 9147）
//	"fernq://alice@127.0.0.1/uuid#test?room_pass=123456"
//
//	// 指定端口
//	"fernq://alice@192.168.1.100:9147/uuid#room?room_pass=123"
//
//	// 域名连接（生产环境）
//	"fernq://alice@room.example.com/uuid#room?room_pass=secret"
//
// 返回值:
//   - error: 连接过程中的错误，nil 表示成功
//     可能的错误：
//   - 格式错误：URL 不符合 fernq 协议规范
//   - 网络错误：无法连接到指定主机
//   - 认证错误：房间密码错误
//   - 房间错误：UUID 不存在或房间已关闭
func (c *Client) Connect(FQC string) error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	// 检查状态
	c.statusMu.Lock()
	if c.isConnected {
		c.statusMu.Unlock()
		return fmt.Errorf("已连接")
	}
	c.statusMu.Unlock()
	// 生成验证信息
	serverAddr, verify, err := codec.ValidateAndExtractAddress(FQC)
	if err != nil {
		return fmt.Errorf("无效的FQC地址: %w", err)
	}

	// 创建连接
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return fmt.Errorf("连接服务器失败: %w", err)
	}
	// 连接成功，尝试验证
	// 创建验证消息
	_, err = conn.Write(verify)
	if err != nil {
		conn.Close()
		return fmt.Errorf("发送验证消息失败: %w", err)
	}
	// 设置最长总时间 3分钟
	timeout := time.NewTimer(3 * time.Minute)
	defer timeout.Stop()
	// 读取数据
	var xxbuff []byte
	for {
		select {
		case <-timeout.C:
			conn.Close()
			return fmt.Errorf("验证超时")
		default:
		}

		err := conn.SetReadDeadline(time.Now().Add(time.Second * 5))
		if err != nil {
			conn.Close()
			return fmt.Errorf("设置读取超时失败: %w", err)
		}

		// 读取数据
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		// 首先检查是否有错误
		if err != nil {
			// 检查是否为超时错误
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// log.Println("读取超时，重新设置超时并继续等待...")
				continue // 超时后重新循环，不执行下面的数据处理
			}
			conn.Close()
			return fmt.Errorf("读取数据失败: %w", err) // 退出循环，连接会被清理
		}

		// 拼接数据
		xxbuff = append(xxbuff, buf[:n]...)
		for {
			msgType, body, remain, err := codec.Decode(xxbuff)
			if err != nil {
				if err == codec.ErrLength {
					break
				}
				conn.Close()
				return fmt.Errorf("解析数据失败: %w", err)
			}
			// 保存剩余数据
			xxbuff = remain

			// 判断是否是心跳
			if msgType == codec.TypePong || msgType == codec.TypePing {
				continue
			}

			// 判断是否是验证结果
			if msgType == codec.TypeRoomVerifyRes {
				result, resm, err := codec.ParseRoomVerifyRes(body)
				if err != nil {
					conn.Close()
					return fmt.Errorf("解析验证结果失败: %w", err)
				}
				if result {
					// 验证成功

					// 赋值到c.conn
					c.writeMu.Lock()
					c.conn = conn
					c.writeMu.Unlock()

					// 设置状态为已连接
					c.statusMu.Lock()
					c.isConnected = true
					c.statusMu.Unlock()

					// 添加上下文和取消函数
					c.ctx, c.cancel = context.WithCancel(context.Background())

					// 添加读输入通道
					c.readChan = make(chan FernqMessage, 1024)

					// 添加读协程
					c.readLoop(xxbuff)

					return nil
				}
				conn.Close()
				return fmt.Errorf("房间验证失败: %s", resm)
			}
			conn.Close()
			return fmt.Errorf("验证失败")
		}
	}
}

// 断开连接
func (c *Client) Stop() error {
	c.statusMu.Lock()
	if !c.isConnected {
		c.statusMu.Unlock()
		return fmt.Errorf("未连接")
	}
	c.statusMu.Unlock()
	c.cancel()
	c.writeMu.Lock()
	c.conn.Close()
	c.writeMu.Unlock()
	c.wg.Wait()
	return nil
}

// 创建客户端
func NewClient(clientName string) *Client {
	return &Client{
		ClientName:  clientName,
		wg:          sync.WaitGroup{},
		isConnected: false,
	}
}
