package codec

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

// ====================== 注册连接 ======================

// 客户端使用
// 创建房间验证
// ValidateAndExtractAddress 验证URL格式，返回可直接连接的地址（支持IP和域名）
// 输入: "fernq://alice@node-a.local/uuid#room?room_pass=secret"
//
//	"fernq://alice@192.168.1.100:8080/uuid#room?pass=123"
//	"fernq://alice@room.example.com/uuid#room?pass=123"
//
// 输出: ("node-a.local:7777", []byte(original), nil)  域名原样返回
//
//	("192.168.1.100:8080", []byte(original), nil) IP+端口
func ValidateAndExtractAddress(roomURL string) (address string, raw []byte, err error) {
	// 1. 基础检查
	if !strings.HasPrefix(roomURL, "fernq://") {
		return "", nil, fmt.Errorf("invalid scheme: must start with fernq://")
	}

	// 2. 标准URL解析
	u, err := url.Parse(roomURL)
	if err != nil {
		return "", nil, fmt.Errorf("invalid url format: %w", err)
	}

	// 3. 必须有host（IP或域名都可以）
	host := u.Hostname()
	if host == "" {
		return "", nil, fmt.Errorf("missing host")
	}

	// 4. 必须有path（uuid位置）
	path := strings.TrimPrefix(u.Path, "/")
	if path == "" {
		return "", nil, fmt.Errorf("missing uuid in path")
	}

	// 5. 提取端口（默认9147）
	port := u.Port()
	if port == "" {
		port = "9147"
	}

	// 6. 验证host格式（IP或域名）
	if !isValidHost(host) {
		return "", nil, fmt.Errorf("invalid host format: %s", host)
	}

	// 7. 组装地址（host:port）
	// 注意：如果是IPv6，需要加括号
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		// IPv6地址，需要括号
		address = fmt.Sprintf("[%s]:%s", host, port)
	} else {
		address = fmt.Sprintf("%s:%s", host, port)
	}

	raw, err = Encode(TypeRoomVerify, []byte(roomURL)) // 添加房间验证类型，并编码
	if err != nil {
		return "", nil, err
	}

	return address, raw, nil
}

// isValidHost 验证host是有效IP或域名
func isValidHost(host string) bool {
	// 尝试作为IP解析
	if net.ParseIP(host) != nil {
		return true
	}

	// 尝试作为域名验证
	// 简单规则：非空，不包含非法字符，有合理长度
	if len(host) > 253 {
		return false
	}

	// 域名不能全是数字和点（那是IP）
	if strings.Trim(host, "0123456789.") == "" {
		return false
	}

	// 基本字符检查
	for _, ch := range host {
		if !isValidDomainChar(ch) && ch != '.' && ch != '-' {
			return false
		}
	}

	return true
}

// isValidDomainChar 检查字符是否有效
func isValidDomainChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9')
}

// 服务端使用
// 解析房间验证信息
// RoomInfo 房间信息
type RoomInfo struct {
	Username string // alice
	UUID     string // 550e8400-e29b-41d4-a716-446655440000
	RoomName string // 战斗房
	Password string // secret123
}

// ValidateAndExtractInfo 验证URL格式，提取所有业务字段
// 输入: "fernq://alice@node-a.local/uuid#room?room_pass=secret"
// 输出: (RoomInfo{}, nil)
func ValidateAndExtractInfo(link []byte) (RoomInfo, error) {
	var info RoomInfo
	// 0. 将链接解码
	roomURL := string(link)

	// 1. 基础检查
	if !strings.HasPrefix(roomURL, "fernq://") {
		return info, fmt.Errorf("invalid scheme: must start with fernq://")
	}

	// 2. 去掉协议前缀，手动解析
	rest := roomURL[8:] // 去掉 "fernq://"

	// 3. 提取用户名（@之前）
	if atIdx := strings.Index(rest, "@"); atIdx != -1 {
		info.Username = rest[:atIdx]
		rest = rest[atIdx+1:]
	} else {
		return info, fmt.Errorf("missing username, expected user@host")
	}

	// 4. 提取主机和路径（/分割）
	if slashIdx := strings.Index(rest, "/"); slashIdx == -1 {
		return info, fmt.Errorf("missing path separator")
	} else {
		rest = rest[slashIdx+1:] // 剩下: uuid#room_name?params
	}

	// 5. 提取查询参数（?之后）
	queryPart := ""
	if qIdx := strings.Index(rest, "?"); qIdx != -1 {
		queryPart = rest[qIdx+1:]
		rest = rest[:qIdx]
	}

	// 解析 room_pass
	if queryPart != "" {
		values, _ := url.ParseQuery(queryPart)
		if pass, ok := values["room_pass"]; ok && len(pass) > 0 {
			info.Password = pass[0]
		}
	}

	// 6. 提取UUID和房间名（#分割）
	if hashIdx := strings.Index(rest, "#"); hashIdx == -1 {
		return info, fmt.Errorf("missing room name separator #")
	} else {
		info.UUID = rest[:hashIdx]
		info.RoomName = rest[hashIdx+1:]
	}

	// 7. 验证必填字段
	if info.UUID == "" {
		return info, fmt.Errorf("missing uuid")
	}
	if info.RoomName == "" {
		return info, fmt.Errorf("missing room name")
	}

	// 8. URL解码房间名
	if decoded, err := url.QueryUnescape(info.RoomName); err == nil {
		info.RoomName = decoded
	}

	return info, nil
}

// 服务端使用
// 创建房间验证结果
func CreateRoomVerifyRes(room string, res bool, msg string) ([]byte, error) {
	// 创建包含验证结果和消息的 JSON 对象
	result := map[string]any{
		"res": res,
		"msg": msg,
	}

	// 将结果序列化为 JSON
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	mes := &ReceiveMessage{
		From:    room,
		Message: resultJSON,
	}
	mesByte, err := EncodeReceiveMessagePB(mes) // 接收消息
	if err != nil {
		return nil, err
	}
	return Encode(TypeRoomVerifyRes, mesByte)
}

// 客户端使用
// 客户端解析验证信息
func ParseRoomVerifyRes(data []byte) (bool, string, error) {
	res, err := DecodeReceiveMessagePB(data)
	if err != nil {
		return false, "", err
	}

	// 解析结果
	var result struct {
		Res bool   `json:"res"`
		Msg string `json:"msg"`
	}
	err = json.Unmarshal(res.Message, &result)
	if err != nil {
		return false, "", err
	}

	return result.Res, result.Msg, nil
}

// ====================== 基本操作 ======================

// 创建房间广播
func CreateRoomBroadcast(from, room string, message []byte) ([]byte, error) {
	mes := &TransitMessage{
		From:    from,
		Target:  room,
		Message: message,
	}
	mesByte, err := EncodeTransitMessagePB(mes) // 中转消息
	if err != nil {
		return nil, err
	}
	return Encode(TypeRoomBroadcast, mesByte)
}

// 创建扫描组播
func CreateUserScan(from, scan string, message []byte) ([]byte, error) {
	mes := &TransitMessage{
		From:    from,
		Target:  scan,
		Message: message,
	}
	mesByte, err := EncodeTransitMessagePB(mes) // 中转消息
	if err != nil {
		return nil, err
	}
	return Encode(TypeUserScan, mesByte)
}

// 创建扫描单播
func CreateUserScanSingle(from, scan string, message []byte) ([]byte, error) {
	mes := &TransitMessage{
		From:    from,
		Target:  scan,
		Message: message,
	}
	mesByte, err := EncodeTransitMessagePB(mes) // 中转消息
	if err != nil {
		return nil, err
	}
	return Encode(TypeUserScanSingle, mesByte)
}

// 创建P2P中转
func CreateP2PRelay(from, target string, message []byte) ([]byte, error) {
	mes := &TransitMessage{
		From:    from,
		Target:  target,
		Message: message,
	}
	mesByte, err := EncodeTransitMessagePB(mes) // 中转消息
	if err != nil {
		return nil, err
	}
	return Encode(TypeP2PRelay, mesByte)
}

// 创建接收消息
func CreateReceiveMessage(from string, message []byte) ([]byte, error) {
	mes := &ReceiveMessage{
		From:    from,
		Message: message,
	}
	mesByte, err := EncodeReceiveMessagePB(mes) // 接收消息
	if err != nil {
		return nil, err
	}
	return Encode(TypeReceiveMessage, mesByte)
}

// ======================= 活性测试 =======================

// 创建心跳消息
func CreatePing() ([]byte, error) {
	return Encode(TypePing, nil)
}

// 创建心跳响应
func CreatePong() ([]byte, error) {
	return Encode(TypePong, nil)
}

// ====================== 请求响应 ======================

// 客户端使用
// 创建请求的中转消息,返回其请求id和中转消息
func CreateRequestMessage(from, target, url string, body []byte) (string, []byte, error) {
	// 生成请求体的uuid的[]byte数组
	xxuuid := uuid.New()

	// 生成请求消息
	message, err := EncodeRequestBodyPB(&RequestBody{
		Url:  url,
		Body: body,
	})
	if err != nil {
		return "", nil, err
	}

	// 封装为中转消息
	mes := &TransitMessage{
		From:    from,
		Target:  target,
		Message: append(xxuuid[:], message...), // 添加uuid
	}
	mesByte, err := EncodeTransitMessagePB(mes) // 中转消息
	if err != nil {
		return "", nil, err
	}

	result, err := Encode(TypeRequestMessage, mesByte)
	if err != nil {
		return "", nil, err
	}

	return xxuuid.String(), result, nil
}

// 客户端使用
// 创建模糊扫描的请求体的中转消息
func CreateRequestMessageScan(from, scan string, url string, body []byte) (string, []byte, error) {
	// 生成请求体的uuid的[]byte数组
	xxuuid := uuid.New()

	// 生成请求消息
	message, err := EncodeRequestBodyPB(&RequestBody{
		Url:  url,
		Body: body,
	})
	if err != nil {
		return "", nil, err
	}
	// 封装为中转消息
	mes := &TransitMessage{
		From:    from,
		Target:  scan,
		Message: append(xxuuid[:], message...), // 添加uuid
	}
	mesByte, err := EncodeTransitMessagePB(mes) // 中转消息

	if err != nil {
		return "", nil, err
	}
	result, err := Encode(TypeRequestMessage, mesByte)
	if err != nil {
		return "", nil, err
	}

	return xxuuid.String(), result, nil
}

// 客户端/服务器 使用
// 创建响应的中转消息
func CreateResponseMessage(from, target string, xxuuid, body []byte, status StatusCode) ([]byte, error) {
	// 创建响应体
	message, err := EncodeResponseBodyPB(&ResponseBody{
		Status: int32(status),
		Body:   body,
	})
	// 封装为中转消息
	mes := &TransitMessage{
		From:    from,
		Target:  target,
		Message: append(xxuuid, message...), // 添加uuid
	}
	mesByte, err := EncodeTransitMessagePB(mes) // 中转消息
	if err != nil {
		return nil, err
	}
	return Encode(TypeResponseMessage, mesByte)
}

// 服务器使用
// 创建请求中转的接收消息
func CreateRequestReceiveMessage(from string, message []byte) ([]byte, error) {
	mes := &ReceiveMessage{
		From:    from,
		Message: message,
	}
	mesByte, err := EncodeReceiveMessagePB(mes) // 接收消息
	if err != nil {
		return nil, err
	}
	return Encode(TypeRequestMessage, mesByte)
}

// 服务器使用
// 创建响应中转的接收消息
func CreateResponseReceiveMessage(from string, message []byte) ([]byte, error) {
	mes := &ReceiveMessage{
		From:    from,
		Message: message,
	}
	mesByte, err := EncodeReceiveMessagePB(mes) // 接收消息
	if err != nil {
		return nil, err
	}
	return Encode(TypeResponseMessage, mesByte)
}

// 服务器使用
// 解析请求或响应,获取id
func ParseRequestOrResponseId(data []byte) ([]byte, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("data too short")
	}

	// 拷贝 16 字节，与原 data 脱钩
	xxuuid := make([]byte, 16)
	copy(xxuuid, data[:16])

	return xxuuid, nil
}

// 客户端使用
// 解析请求接收消息
func ParseRequestReceiveMessage(data []byte) ([]byte, *RequestBody, error) {
	if len(data) < 16 {
		return nil, nil, fmt.Errorf("data too short")
	}

	// 拷贝 16 字节，与原 data 脱钩
	xxuuid := make([]byte, 16)
	copy(xxuuid, data[:16])

	// 解析剩余部分,为*RequestBody类型
	mes, err := DecodeRequestBodyPB(data[16:])
	if err != nil {
		return nil, nil, err
	}
	return xxuuid, mes, nil
}

// 客户端使用
// 解析响应接收消息
func ParseResponseReceiveMessage(data []byte) (string, *ResponseBody, error) {
	if len(data) < 16 {
		return "", nil, fmt.Errorf("data too short")
	}

	// 1. 取出并拷贝 16 字节
	var raw [16]byte
	copy(raw[:], data[:16])

	// 2. 转成 uuid.UUID
	uid, err := uuid.FromBytes(raw[:])
	if err != nil { // 几乎不会失败
		return "", nil, err
	}

	// 3. 解析剩余部分,为*ResponseBody类型
	mes, err := DecodeResponseBodyPB(data[16:])
	if err != nil {
		return "", nil, err
	}

	// 4. 返回标准字符串形式，与 CreateRequestMessage 里的一致
	return uid.String(), mes, nil
}
