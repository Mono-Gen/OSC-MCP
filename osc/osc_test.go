package osc

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func TestEncodeDecodeMessage(t *testing.T) {
	tests := []struct {
		name    string
		message Message
	}{
		{
			name: "Simple message with int and float",
			message: Message{
				Address: "/test/address",
				Args: []Argument{
					{Type: "int", Value: int32(42)},
					{Type: "float", Value: float32(3.14)},
				},
			},
		},
		{
			name: "Message with string and bool",
			message: Message{
				Address: "/another/test",
				Args: []Argument{
					{Type: "string", Value: "hello"},
					{Type: "bool", Value: true},
					{Type: "bool", Value: false},
				},
			},
		},
		{
			name: "Address and type tag padding alignment tests",
			message: Message{
				Address: "/abc", // 4 chars -> aligned with 4 null bytes
				Args: []Argument{
					{Type: "string", Value: "abcd"}, // 4 chars -> aligned with 4 null bytes
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := EncodeMessage(tt.message)
			if err != nil {
				t.Fatalf("failed to encode: %v", err)
			}

			// Ensure packet is 4-byte aligned
			if len(encoded)%4 != 0 {
				t.Errorf("encoded packet length %d is not a multiple of 4", len(encoded))
			}

			decodedList, err := DecodePacket(encoded)
			if err != nil {
				t.Fatalf("failed to decode: %v", err)
			}

			if len(decodedList) != 1 {
				t.Fatalf("expected 1 message, got %d", len(decodedList))
			}

			decoded := decodedList[0]
			if decoded.Address != tt.message.Address {
				t.Errorf("expected address %s, got %s", tt.message.Address, decoded.Address)
			}

			if len(decoded.Args) != len(tt.message.Args) {
				t.Fatalf("expected %d args, got %d", len(tt.message.Args), len(decoded.Args))
			}

			for i, arg := range decoded.Args {
				expectedArg := tt.message.Args[i]
				if arg.Type != expectedArg.Type {
					t.Errorf("arg %d: expected type %s, got %s", i, expectedArg.Type, arg.Type)
				}

				switch arg.Type {
				case "int":
					var exp int32
					switch v := expectedArg.Value.(type) {
					case int32:
						exp = v
					case int:
						exp = int32(v)
					}
					got := arg.Value.(int32)
					if got != exp {
						t.Errorf("arg %d: expected value %v, got %v", i, exp, got)
					}
				case "float":
					var exp float32
					switch v := expectedArg.Value.(type) {
					case float32:
						exp = v
					case float64:
						exp = float32(v)
					}
					got := arg.Value.(float32)
					if math.Abs(float64(got-exp)) > 1e-6 {
						t.Errorf("arg %d: expected value %v, got %v", i, exp, got)
					}
				case "string":
					got := arg.Value.(string)
					exp := expectedArg.Value.(string)
					if got != exp {
						t.Errorf("arg %d: expected value %s, got %s", i, exp, got)
					}
				case "bool":
					got := arg.Value.(bool)
					exp := expectedArg.Value.(bool)
					if got != exp {
						t.Errorf("arg %d: expected value %v, got %v", i, exp, got)
					}
				}
			}
		})
	}
}

func TestDecodeBundle(t *testing.T) {
	var buf bytes.Buffer
	buf.Write([]byte("#bundle\x00"))
	buf.Write([]byte{0, 0, 0, 0, 0, 0, 0, 1})

	// Msg 1
	msg1, _ := EncodeMessage(Message{Address: "/m1"})
	size1 := make([]byte, 4)
	binary.BigEndian.PutUint32(size1, uint32(len(msg1)))
	buf.Write(size1)
	buf.Write(msg1)

	// Msg 2
	msg2, _ := EncodeMessage(Message{
		Address: "/m2",
		Args:    []Argument{{Type: "int", Value: 42}},
	})
	size2 := make([]byte, 4)
	binary.BigEndian.PutUint32(size2, uint32(len(msg2)))
	buf.Write(size2)
	buf.Write(msg2)

	decoded, err := DecodePacket(buf.Bytes())
	if err != nil {
		t.Fatalf("failed to decode bundle: %v", err)
	}

	if len(decoded) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(decoded))
	}

	if decoded[0].Address != "/m1" {
		t.Errorf("expected /m1, got %s", decoded[0].Address)
	}
	if decoded[1].Address != "/m2" {
		t.Errorf("expected /m2, got %s", decoded[1].Address)
	}
	if decoded[1].Args[0].Value.(int32) != 42 {
		t.Errorf("expected 42, got %v", decoded[1].Args[0].Value)
	}
}

func TestInvalidPackets(t *testing.T) {
	// Test empty
	_, err := DecodePacket([]byte{})
	if err == nil {
		t.Error("expected error for empty packet")
	}

	// Test invalid address
	_, err = DecodePacket([]byte("invalid_no_slash\x00\x00\x00\x00,\x00\x00\x00"))
	if err == nil {
		t.Error("expected error for invalid address")
	}

	// Test invalid int size
	msgBytes, _ := EncodeMessage(Message{
		Address: "/test",
		Args:    []Argument{{Type: "int", Value: 10}},
	})
	// Truncate the last 2 bytes of the int32 argument
	truncated := msgBytes[:len(msgBytes)-2]
	_, err = DecodePacket(truncated)
	if err == nil {
		t.Error("expected error for truncated packet")
	}
}
