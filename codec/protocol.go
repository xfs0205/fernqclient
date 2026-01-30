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

// ValidateAndExtractAddress 验证URL格式，返回可直接连接的地址
// 输入参数:
//   - username: 用户名（如 "alice"）
//   - roomURL: 目标URL（如 "fernq://connect/node-a.local:8080/uuid#room?room_pass=secret"）
//
// 输出: ("node-a.local:8080", []byte(encoded), nil)  域名+端口
//
//	("node-a.local", []byte(encoded), nil)        域名无端口
//	("192.168.1.100:9147", []byte(encoded), nil)  IP无端口时用默认9147
//	("[::1]:9147", []byte(encoded), nil)          IPv6无端口时用默认9147
func ValidateAndExtractAddress(username string, roomURL string) (address string, raw []byte, err error) {
	// 1. 基础检查
	if !strings.HasPrefix(roomURL, "fernq://connect/") {
		return "", nil, fmt.Errorf("invalid scheme: must start with fernq://connect/")
	}

	// 2. 标准URL解析
	u, err := url.Parse(roomURL)
	if err != nil {
		return "", nil, fmt.Errorf("invalid url format: %w", err)
	}

	// 3. 验证 host 必须是 "connect"
	if u.Host != "connect" {
		return "", nil, fmt.Errorf("invalid host: expected 'connect', got '%s'", u.Host)
	}

	// 4. 提取节点地址（path 的第一段，可能包含端口）
	path := strings.TrimPrefix(u.Path, "/")
	pathParts := strings.SplitN(path, "/", 2)
	if len(pathParts) < 2 || pathParts[0] == "" {
		return "", nil, fmt.Errorf("missing node address")
	}
	nodePart := pathParts[0]

	// 5. 解析节点地址和端口
	var node string
	var port string
	var isIP bool

	// 检查是否包含端口（IPv6用[]包裹，需要特殊处理）
	if strings.HasPrefix(nodePart, "[") {
		// IPv6 格式: [::1]:8080 或 [::1]
		if idx := strings.LastIndex(nodePart, "]:"); idx != -1 {
			node = nodePart[:idx+1] // 包含 []
			port = nodePart[idx+2:]
		} else if strings.HasSuffix(nodePart, "]") {
			node = nodePart
			port = ""
		} else {
			return "", nil, fmt.Errorf("invalid IPv6 format: %s", nodePart)
		}
		isIP = true
	} else {
		// IPv4 或域名格式: host:port 或 host
		if idx := strings.LastIndex(nodePart, ":"); idx != -1 &&
			!strings.Contains(nodePart[idx+1:], ":") { // 确保不是IPv6冒号
			node = nodePart[:idx]
			port = nodePart[idx+1:]
		} else {
			node = nodePart
			port = ""
		}
		// 判断是IP还是域名
		isIP = net.ParseIP(node) != nil
	}

	// 6. 验证节点地址格式（IP或域名）
	if !isValidHost(node) {
		return "", nil, fmt.Errorf("invalid node format: %s", node)
	}

	// 7. 组装地址
	// IP 没有端口时默认 9147，域名保持原样
	if isIP && port == "" {
		port = "9147"
	}

	if port != "" {
		if strings.Contains(node, ":") && !strings.HasPrefix(node, "[") {
			// IPv6 裸地址需要括号（理论上不会走到这里，因为IPv6都用[]包裹）
			address = fmt.Sprintf("[%s]:%s", node, port)
		} else {
			address = fmt.Sprintf("%s:%s", node, port)
		}
	} else {
		// 域名无端口
		address = node
	}

	// 8. 构造 VerifyMessage
	vm := &VerifyMessage{
		ClientId: username,
		Token:    roomURL,
	}

	// 9. protobuf 编码
	vmData, err := EncodeVerifyMessagePB(vm)
	if err != nil {
		return "", nil, fmt.Errorf("failed to encode verify message: %w", err)
	}

	// 10. 外层封装编码
	raw, err = Encode(TypeRoomVerify, vmData)
	if err != nil {
		return "", nil, err
	}

	return address, raw, nil
}

// isValidHost 验证host是有效IP或域名
func isValidHost(host string) bool {
	// 去掉 IPv6 的括号
	cleanHost := strings.Trim(host, "[]")

	// 尝试作为IP解析
	if net.ParseIP(cleanHost) != nil {
		return true
	}

	// 域名验证
	if len(cleanHost) > 253 {
		return false
	}

	// 不能全是数字和点
	if strings.Trim(cleanHost, "0123456789.") == "" {
		return false
	}

	// 基本字符检查
	for _, ch := range cleanHost {
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

// ValidateAndExtractInfo 验证并提取房间信息
// 输入: data 是经过 protobuf 编码的 VerifyMessage
// 输出: (RoomInfo{}, nil)
func ValidateAndExtractInfo(data []byte) (RoomInfo, error) {
	var info RoomInfo

	// 0. 解析 VerifyMessage
	vm, err := DecodeVerifyMessagePB(data)
	if err != nil {
		return info, fmt.Errorf("failed to decode verify message: %w", err)
	}

	// 1. 提取 Username（来自 ClientId）
	info.Username = vm.ClientId
	if info.Username == "" {
		return info, fmt.Errorf("missing client_id in verify message")
	}

	// 2. 从 VerifyMessage.Token 获取目标 URL
	roomURL := vm.Token

	// 3. 基础检查
	if !strings.HasPrefix(roomURL, "fernq://connect/") {
		return info, fmt.Errorf("invalid scheme: must start with fernq://connect/")
	}

	// 4. 去掉前缀，手动解析
	rest := roomURL[16:] // 去掉 "fernq://connect/"

	// 5. 跳过节点地址（node-a.local/）
	if slashIdx := strings.Index(rest, "/"); slashIdx == -1 {
		return info, fmt.Errorf("missing path separator after node")
	} else {
		rest = rest[slashIdx+1:] // 剩下: uuid#room_name?params
	}

	// 6. 提取查询参数（?之后）
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

	// 7. 提取UUID和房间名（#分割）
	if hashIdx := strings.Index(rest, "#"); hashIdx == -1 {
		return info, fmt.Errorf("missing room name separator #")
	} else {
		info.UUID = rest[:hashIdx]
		info.RoomName = rest[hashIdx+1:]
	}

	// 8. 验证必填字段
	if info.UUID == "" {
		return info, fmt.Errorf("missing uuid")
	}
	if info.RoomName == "" {
		return info, fmt.Errorf("missing room name")
	}

	// 9. URL解码房间名
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
func CreateRoomBroadcast(room string, message []byte) ([]byte, error) {
	mes := &TransitMessage{
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
func CreateUserScan(scan string, message []byte) ([]byte, error) {
	mes := &TransitMessage{
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
func CreateUserScanSingle(scan string, message []byte) ([]byte, error) {
	mes := &TransitMessage{
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
func CreateP2PRelay(target string, message []byte) ([]byte, error) {
	mes := &TransitMessage{
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
func CreateRequestMessage(target, url string, body []byte) (string, []byte, error) {
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
func CreateRequestMessageScan(scan string, url string, body []byte) (string, []byte, error) {
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
func CreateResponseMessage(target string, xxuuid, body []byte, status StatusCode) ([]byte, error) {
	// 创建响应体
	message, err := EncodeResponseBodyPB(&ResponseBody{
		Status: int32(status),
		Body:   body,
	})
	// 封装为中转消息
	mes := &TransitMessage{
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
