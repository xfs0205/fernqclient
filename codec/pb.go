package codec

import "google.golang.org/protobuf/proto"

// ========== VerifyMessage ==========
func EncodeVerifyMessagePB(vm *VerifyMessage) ([]byte, error) {
	return proto.Marshal(vm)
}
func DecodeVerifyMessagePB(b []byte) (*VerifyMessage, error) {
	var vm VerifyMessage
	if err := proto.Unmarshal(b, &vm); err != nil {
		return nil, err
	}
	return &vm, nil
}

// ========== TransitMessage ==========
func EncodeTransitMessagePB(tm *TransitMessage) ([]byte, error) {
	return proto.Marshal(tm)
}
func DecodeTransitMessagePB(b []byte) (*TransitMessage, error) {
	var tm TransitMessage
	if err := proto.Unmarshal(b, &tm); err != nil {
		return nil, err
	}
	return &tm, nil
}

// ========== ReceiveMessage ==========
func EncodeReceiveMessagePB(rm *ReceiveMessage) ([]byte, error) {
	return proto.Marshal(rm)
}
func DecodeReceiveMessagePB(b []byte) (*ReceiveMessage, error) {
	var rm ReceiveMessage
	if err := proto.Unmarshal(b, &rm); err != nil {
		return nil, err
	}
	return &rm, nil
}

// ========== RequestBody ==========
func EncodeRequestBodyPB(rb *RequestBody) ([]byte, error) {
	return proto.Marshal(rb)
}
func DecodeRequestBodyPB(b []byte) (*RequestBody, error) {
	var rb RequestBody
	if err := proto.Unmarshal(b, &rb); err != nil {
		return nil, err
	}
	return &rb, nil
}

// ========== ResponseBody ==========
func EncodeResponseBodyPB(rb *ResponseBody) ([]byte, error) {
	return proto.Marshal(rb)
}
func DecodeResponseBodyPB(b []byte) (*ResponseBody, error) {
	var rb ResponseBody
	if err := proto.Unmarshal(b, &rb); err != nil {
		return nil, err
	}
	return &rb, nil
}
