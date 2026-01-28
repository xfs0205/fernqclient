package codec

// fernq协议类型
type FernqTypeCode uint16

// StatusCode 直接采用 protobuf 的 int32，零转换
type StatusCode int32

const (
	TypeRoomVerify         FernqTypeCode = 0x9E // 158 房间验证
	TypeRoomBroadcast      FernqTypeCode = 0x9F // 159 房间广播
	TypeUserScan           FernqTypeCode = 0xA0 // 160 扫描组播
	TypeP2PRelay           FernqTypeCode = 0xA1 // 161 P2P中转
	TypeRoomVerifyRes      FernqTypeCode = 0xA2 // 162 房间验证结果
	TypePing               FernqTypeCode = 0xA3 // 163 Ping 心跳
	TypePong               FernqTypeCode = 0xA4 // 164 Pong 心跳响应
	TypeReceiveMessage     FernqTypeCode = 0xA5 // 165 接收消息
	TypeRequestMessage     FernqTypeCode = 0xA6 // 166 请求消息
	TypeResponseMessage    FernqTypeCode = 0xA7 // 167 响应消息
	TypeUserScanSingle     FernqTypeCode = 0xA8 // 168 扫描单播，随机选择一个
	TypeRequestMessageScan FernqTypeCode = 0xA9 // 169 请求消息扫描,随机选择一个发送
)

const (
	// 2xx 成功
	StatusOK        StatusCode = 200
	StatusCreated   StatusCode = 201
	StatusAccepted  StatusCode = 202
	StatusNoContent StatusCode = 204

	// 4xx 客户端错误
	StatusBadRequest      StatusCode = 400
	StatusUnauthorized    StatusCode = 401
	StatusForbidden       StatusCode = 403
	StatusNotFound        StatusCode = 404
	StatusConflict        StatusCode = 409
	StatusPayloadTooLarge StatusCode = 413
	StatusTooManyRequests StatusCode = 429

	// 5xx 服务端错误
	StatusInternalServerError StatusCode = 500
	StatusBadGateway          StatusCode = 502
	StatusServiceUnavailable  StatusCode = 503
)
