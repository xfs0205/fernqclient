package codec

import (
	"encoding/binary"
)

const (
	HeaderTotal = 6 // 4 + 2
)

// Encode 将 msgType + payload 打包成帧
func Encode(msgType FernqTypeCode, payload []byte) ([]byte, error) {
	if payload == nil {
		payload = []byte{}
	}
	total := uint32(HeaderTotal + len(payload))
	buf := make([]byte, total)
	binary.BigEndian.PutUint32(buf[0:4], total)
	binary.BigEndian.PutUint16(buf[4:6], uint16(msgType))
	copy(buf[HeaderTotal:], payload)
	return buf, nil
}

// Decode 解出一帧，返回 (msgType, 正文, 剩余数据, 错误)
func Decode(data []byte) (FernqTypeCode, []byte, []byte, error) {
	if len(data) < HeaderTotal {
		return 0, nil, data, ErrLength
	}
	total := binary.BigEndian.Uint32(data[0:4])
	if uint32(len(data)) < total {
		return 0, nil, data, ErrLength
	}
	msgType := binary.BigEndian.Uint16(data[4:6])
	body := data[HeaderTotal:total]
	remain := data[total:]
	return FernqTypeCode(msgType), body, remain, nil
}
