package message

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// NOTE 这里是应用层协议，所以使用主机字节序即可

// 序列化消息为字节流
func Marshal(msg interface{}) ([]byte, error) {
	switch m := msg.(type) {
	case *Packet:
		return marshalPacket(m)
	case *TLVMsg:
		return marshalTLVMsg(m)
	case *SeqedMsg:
		return marshalSeqedMsg(m)
	case *SeqedTLVMsg:
		return marshalSeqedTLVMsg(m)
	default:
		return nil, fmt.Errorf("unsupported message type: %T", msg)
	}
}

// 反序列化字节流为消息
// NOTE 这里必须要求msg为指针类型，否则下面的类型断言过不去
func Unmarshal(data []byte, msg interface{}, readBody bool) error {
	switch m := msg.(type) {
	case *Packet:
		return unmarshalPacket(data, m, readBody)
	case *TLVMsg:
		return unmarshalTLVMsg(data, m, readBody)
	case *SeqedMsg:
		return unmarshalSeqedMsg(data, m, readBody)
	case *SeqedTLVMsg:
		return unmarshalSeqedTLVMsg(data, m, readBody)
	default:
		return fmt.Errorf("unsupported message type: %T", msg)
	}
}

// 以下是类型特化的具体实现

func marshalPacket(p *Packet) ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})
	// 写bodyLen
	if err := binary.Write(buffer, binary.NativeEndian, p.bodyLen); err != nil {
		return nil, err
	}
	// 写body
	if err := binary.Write(buffer, binary.NativeEndian, p.body); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func unmarshalPacket(data []byte, p *Packet, readBody bool) error {
	reader := bytes.NewReader(data)
	// 读bodyLen
	if err := binary.Read(reader, binary.NativeEndian, &p.bodyLen); err != nil {
		return err
	}
	if readBody {
		// 读body
		// 分配足够的空间来存储body
		p.body = make([]byte, p.bodyLen)
		return binary.Read(reader, binary.NativeEndian, p.body)
	}
	return nil
}

func marshalTLVMsg(t *TLVMsg) ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})
	// 写tag
	if err := binary.Write(buffer, binary.NativeEndian, t.tag); err != nil {
		return nil, err
	}
	// 写bodyLen
	if err := binary.Write(buffer, binary.NativeEndian, t.bodyLen); err != nil {
		return nil, err
	}
	// 写body
	if err := binary.Write(buffer, binary.NativeEndian, t.body); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func unmarshalTLVMsg(data []byte, t *TLVMsg, readBody bool) error {
	reader := bytes.NewReader(data)
	// 读tag
	if err := binary.Read(reader, binary.NativeEndian, &t.tag); err != nil {
		return err
	}
	// 读bodyLen
	if err := binary.Read(reader, binary.NativeEndian, &t.bodyLen); err != nil {
		return err
	}
	if readBody {
		// 读body
		// 分配足够的空间来存储body
		t.body = make([]byte, t.bodyLen)
		return binary.Read(reader, binary.NativeEndian, t.body)
	}
	return nil
}

func marshalSeqedMsg(s *SeqedMsg) ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})
	// 写serial
	if err := binary.Write(buffer, binary.NativeEndian, s.serial); err != nil {
		return nil, err
	}
	// 写bodyLen
	if err := binary.Write(buffer, binary.NativeEndian, s.bodyLen); err != nil {
		return nil, err
	}
	// 写body
	if err := binary.Write(buffer, binary.NativeEndian, s.body); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func unmarshalSeqedMsg(data []byte, s *SeqedMsg, readBody bool) error {
	reader := bytes.NewReader(data)
	// 读serial
	if err := binary.Read(reader, binary.NativeEndian, &s.serial); err != nil {
		return err
	}
	// 读bodyLen
	if err := binary.Read(reader, binary.NativeEndian, &s.bodyLen); err != nil {
		return err
	}
	if readBody {
		// 读body
		// 分配足够的空间来存储body
		s.body = make([]byte, s.bodyLen)
		return binary.Read(reader, binary.NativeEndian, s.body)
	}
	return nil
}

func marshalSeqedTLVMsg(s *SeqedTLVMsg) ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})
	// 写serial
	if err := binary.Write(buffer, binary.NativeEndian, s.serial); err != nil {
		return nil, err
	}
	// 写tag
	if err := binary.Write(buffer, binary.NativeEndian, s.tag); err != nil {
		return nil, err
	}
	// 写bodyLen
	if err := binary.Write(buffer, binary.NativeEndian, s.bodyLen); err != nil {
		return nil, err
	}
	// 写body
	if err := binary.Write(buffer, binary.NativeEndian, s.body); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func unmarshalSeqedTLVMsg(data []byte, s *SeqedTLVMsg, readBody bool) error {
	reader := bytes.NewReader(data)
	// 读serial
	if err := binary.Read(reader, binary.NativeEndian, &s.serial); err != nil {
		return err
	}
	// 读tag
	if err := binary.Read(reader, binary.NativeEndian, &s.tag); err != nil {
		return err
	}
	//读bodyLen
	if err := binary.Read(reader, binary.NativeEndian, &s.bodyLen); err != nil {
		return err
	}
	if readBody {
		// 读body
		// 分配足够的空间来存储body
		s.body = make([]byte, s.bodyLen)
		return binary.Read(reader, binary.NativeEndian, s.body)
	}
	return nil
}

func UmarshalBodyOnly(bodyData []byte, bodyLen int, p IPacket) error {
	reader := bytes.NewBuffer(bodyData)
	// 分配足够的空间来存储body
	body := make([]byte, bodyLen)
	if err := binary.Read(reader, binary.NativeEndian, body); err != nil {
		return err
	}
	p.SetBody(body)
	return nil
}
