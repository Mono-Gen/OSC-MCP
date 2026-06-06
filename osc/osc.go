package osc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
)

// Argument represents an OSC argument.
type Argument struct {
	Type  string `json:"type"`  // "int", "float", "string", "bool"
	Value any    `json:"value"`
}

// Message represents an OSC Message.
type Message struct {
	Address string     `json:"address"`
	Args    []Argument `json:"args,omitempty"`
}

// EncodeMessage converts an OSC Message struct to raw bytes.
func EncodeMessage(msg Message) ([]byte, error) {
	var buf bytes.Buffer

	// 1. Address Pattern
	if msg.Address == "" || msg.Address[0] != '/' {
		return nil, errors.New("OSC address must start with '/'")
	}
	addressBytes := encodeString(msg.Address)
	buf.Write(addressBytes)

	// 2. Type Tag String
	typeTag := ","
	for _, arg := range msg.Args {
		switch arg.Type {
		case "int":
			typeTag += "i"
		case "float":
			typeTag += "f"
		case "string":
			typeTag += "s"
		case "bool":
			if val, ok := arg.Value.(bool); ok && val {
				typeTag += "T"
			} else {
				typeTag += "F"
			}
		default:
			return nil, fmt.Errorf("unsupported argument type: %s", arg.Type)
		}
	}
	buf.Write(encodeString(typeTag))

	// 3. Arguments
	for _, arg := range msg.Args {
		switch arg.Type {
		case "int":
			var val int32
			switch v := arg.Value.(type) {
			case int32:
				val = v
			case int:
				val = int32(v)
			case float64:
				if v < float64(math.MinInt32) || v > float64(math.MaxInt32) {
					return nil, fmt.Errorf("integer value out of int32 range: %f", v)
				}
				val = int32(v)
			default:
				return nil, fmt.Errorf("invalid value for int type: %v", arg.Value)
			}
			var b [4]byte
			binary.BigEndian.PutUint32(b[:], uint32(val))
			buf.Write(b[:])
		case "float":
			var val float32
			switch v := arg.Value.(type) {
			case float32:
				val = v
			case float64:
				val = float32(v)
			case int:
				val = float32(v)
			default:
				return nil, fmt.Errorf("invalid value for float type: %v", arg.Value)
			}
			var b [4]byte
			binary.BigEndian.PutUint32(b[:], math.Float32bits(val))
			buf.Write(b[:])
		case "string":
			str, ok := arg.Value.(string)
			if !ok {
				return nil, fmt.Errorf("invalid value for string type: %v", arg.Value)
			}
			buf.Write(encodeString(str))
		case "bool":
			// Handled by Type Tag T/F, no data bytes.
		}
	}

	return buf.Bytes(), nil
}

func encodeString(s string) []byte {
	b := []byte(s)
	b = append(b, 0x00) // MUST have at least one null terminator
	pad := (4 - (len(b) % 4)) % 4
	for i := 0; i < pad; i++ {
		b = append(b, 0x00)
	}
	return b
}

// DecodePacket parses raw OSC packet data (Message or Bundle) into Messages.
func DecodePacket(data []byte) ([]Message, error) {
	if len(data) == 0 {
		return nil, errors.New("empty packet")
	}

	// Check for OSC Bundle signature
	if len(data) >= 8 && string(data[:8]) == "#bundle\x00" {
		return decodeBundle(data)
	}

	// Otherwise parse as a single Message
	msg, err := DecodeMessage(data)
	if err != nil {
		return nil, err
	}
	return []Message{msg}, nil
}

// DecodeMessage parses raw bytes of an OSC Message.
func DecodeMessage(data []byte) (Message, error) {
	var msg Message
	offset := 0

	// 1. Address Pattern
	address, nextOffset, err := decodeString(data, offset)
	if err != nil {
		return msg, fmt.Errorf("failed to decode address: %w", err)
	}
	if address == "" || address[0] != '/' {
		return msg, fmt.Errorf("invalid OSC address: %s", address)
	}
	msg.Address = address
	offset = nextOffset

	// 2. Type Tag String
	typeTag, nextOffset, err := decodeString(data, offset)
	if err != nil {
		return msg, fmt.Errorf("failed to decode type tag: %w", err)
	}
	if typeTag == "" || typeTag[0] != ',' {
		return msg, fmt.Errorf("invalid OSC type tag: %s", typeTag)
	}
	offset = nextOffset

	// 3. Arguments
	for i := 1; i < len(typeTag); i++ {
		tag := typeTag[i]
		switch tag {
		case 'i':
			if offset+4 > len(data) {
				return msg, errors.New("insufficient data for int32 argument")
			}
			val := int32(binary.BigEndian.Uint32(data[offset : offset+4]))
			msg.Args = append(msg.Args, Argument{Type: "int", Value: val})
			offset += 4
		case 'f':
			if offset+4 > len(data) {
				return msg, errors.New("insufficient data for float32 argument")
			}
			bits := binary.BigEndian.Uint32(data[offset : offset+4])
			val := math.Float32frombits(bits)
			msg.Args = append(msg.Args, Argument{Type: "float", Value: val})
			offset += 4
		case 's':
			str, next, err := decodeString(data, offset)
			if err != nil {
				return msg, fmt.Errorf("failed to decode string argument: %w", err)
			}
			msg.Args = append(msg.Args, Argument{Type: "string", Value: str})
			offset = next
		case 'T':
			msg.Args = append(msg.Args, Argument{Type: "bool", Value: true})
		case 'F':
			msg.Args = append(msg.Args, Argument{Type: "bool", Value: false})
		default:
			return msg, fmt.Errorf("unsupported type tag in packet: %c", tag)
		}
	}

	return msg, nil
}

func decodeString(data []byte, offset int) (string, int, error) {
	if offset >= len(data) {
		return "", offset, errors.New("offset out of range")
	}

	nullIdx := -1
	for i := offset; i < len(data); i++ {
		if data[i] == 0x00 {
			nullIdx = i
			break
		}
	}
	if nullIdx == -1 {
		return "", offset, errors.New("null terminator not found in OSC string")
	}

	str := string(data[offset:nullIdx])
	totalLen := nullIdx - offset + 1
	pad := (4 - (totalLen % 4)) % 4
	nextOffset := nullIdx + 1 + pad

	if nextOffset > len(data) {
		return "", offset, errors.New("insufficient data for string padding")
	}

	return str, nextOffset, nil
}

func decodeBundle(data []byte) ([]Message, error) {
	if len(data) < 16 {
		return nil, errors.New("insufficient data for bundle header")
	}
	offset := 16 // Skip "#bundle\x00" (8 bytes) + NTP timetag (8 bytes)

	var messages []Message
	for offset < len(data) {
		if offset+4 > len(data) {
			return nil, errors.New("insufficient data for bundle element size")
		}
		size := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4

		if size < 0 || offset+size > len(data) {
			return nil, fmt.Errorf("invalid bundle element size: %d", size)
		}

		elementData := data[offset : offset+size]
		offset += size

		elementMessages, err := DecodePacket(elementData)
		if err != nil {
			return nil, fmt.Errorf("failed to decode bundle element: %w", err)
		}
		messages = append(messages, elementMessages...)
	}

	return messages, nil
}

// SendOSC sends an OSC Message to the specified target.
func SendOSC(host string, port int, msg Message) error {
	addrStr := fmt.Sprintf("%s:%d", host, port)
	udpAddr, err := net.ResolveUDPAddr("udp", addrStr)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return fmt.Errorf("failed to dial UDP: %w", err)
	}
	defer conn.Close()

	payload, err := EncodeMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to encode OSC message: %w", err)
	}

	_, err = conn.Write(payload)
	if err != nil {
		return fmt.Errorf("failed to write UDP payload: %w", err)
	}

	return nil
}
