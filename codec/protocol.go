package codec

import "encoding/json"

const (
	TypeRoomVerify     uint16 = 0x9E // 158 房间验证
	TypeRoomBroadcast  uint16 = 0x9F // 159 房间广播
	TypeUserScan       uint16 = 0xA0 // 160 扫描组播
	TypeP2PRelay       uint16 = 0xA1 // 161 P2P中转
	TypeRoomVerifyRes  uint16 = 0xA2 // 162 房间验证结果
	TypePing           uint16 = 0xA3 // 163 Ping 心跳
	TypePong           uint16 = 0xA4 // 164 Pong 心跳响应
	TypeReceiveMessage uint16 = 0xA5 // 165 接收消息
)

// 创建房间验证
func CreateRoomVerify(from, room, password string) ([]byte, error) {
	mes := &TransitMessage{
		From:    from,
		Target:  room,
		Message: []byte(password),
	}
	mesByte, err := EncodeTransitMessagePB(mes) // 中转消息
	if err != nil {
		return nil, err
	}
	return Encode(TypeRoomVerify, mesByte)
}

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

// 创建心跳消息
func CreatePing() ([]byte, error) {
	return Encode(TypePing, nil)
}

// 创建心跳响应
func CreatePong() ([]byte, error) {
	return Encode(TypePong, nil)
}

// 解析验证信息
func ParseRoomVerify(data []byte) (bool, string, error) {
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
