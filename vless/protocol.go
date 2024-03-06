package vless

import (
	"encoding/binary"
	"io"

	"github.com/sagernet/sing-vmess"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/buf"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/sing/common/rw"
)

const Version = 0

type Request struct {
	UUID        []byte
	Command     byte
	Destination M.Socksaddr
}

func WriteRequest(writer io.Writer, request Request, payload []byte) error {
	var requestLen int
	requestLen += 1  // version
	requestLen += 16 // uuid
	requestLen += 1  // protobuf length
	requestLen += 1  // command
	if request.Command != vmess.CommandMux {
		requestLen += vmess.AddressSerializer.AddrPortLen(request.Destination)
	}
	requestLen += len(payload)
	buffer := buf.NewSize(requestLen)
	defer buffer.Release()
	common.Must(
		buffer.WriteByte(Version),
		common.Error(buffer.Write(request.UUID)),
		buffer.WriteByte(0),
		buffer.WriteByte(request.Command),
	)

	if request.Command != vmess.CommandMux {
		err := vmess.AddressSerializer.WriteAddrPort(buffer, request.Destination)
		if err != nil {
			return err
		}
	}

	common.Must1(buffer.Write(payload))
	return common.Error(writer.Write(buffer.Bytes()))
}

func WritePacketRequest(writer io.Writer, request Request, payload []byte) error {
	var requestLen int
	requestLen += 1  // version
	requestLen += 16 // uuid
	requestLen += 1  // protobuf length
	requestLen += 1  // command
	requestLen += vmess.AddressSerializer.AddrPortLen(request.Destination)
	if len(payload) > 0 {
		requestLen += 2
		requestLen += len(payload)
	}
	buffer := buf.NewSize(requestLen)
	defer buffer.Release()
	common.Must(
		buffer.WriteByte(Version),
		common.Error(buffer.Write(request.UUID)),
		buffer.WriteByte(0),
		buffer.WriteByte(vmess.CommandUDP),
	)
	err := vmess.AddressSerializer.WriteAddrPort(buffer, request.Destination)
	if err != nil {
		return err
	}
	common.Must(
		binary.Write(buffer, binary.BigEndian, uint16(len(payload))),
		common.Error(buffer.Write(payload)),
	)
	return common.Error(writer.Write(buffer.Bytes()))
}

func ReadResponse(reader io.Reader) error {
	version, err := rw.ReadByte(reader)
	if err != nil {
		return err
	}
	if version != Version {
		return E.New("unknown version: ", version)
	}
	protobufLength, err := rw.ReadByte(reader)
	if err != nil {
		return err
	}
	if protobufLength > 0 {
		err = rw.SkipN(reader, int(protobufLength))
		if err != nil {
			return err
		}
	}
	return nil
}
