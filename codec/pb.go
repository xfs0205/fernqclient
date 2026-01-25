package codec

import "google.golang.org/protobuf/proto"

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
