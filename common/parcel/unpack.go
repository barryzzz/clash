package parcel

import "encoding/binary"

type UnpackParcel struct {
	block  []byte
	offset int
	order  binary.ByteOrder
}

func Unpack(block []byte, order binary.ByteOrder) *UnpackParcel {
	return &UnpackParcel{
		block:  block,
		offset: 0,
		order:  order,
	}
}

func (p *UnpackParcel) Byte() byte {
	b := p.block[p.offset]
	p.offset += 1

	return b
}

func (p *UnpackParcel) Bytes(n int) []byte {
	r := p.block[p.offset : p.offset+n]
	p.offset += n

	return r
}

func (p *UnpackParcel) UInt16() uint16 {
	value := p.order.Uint16(p.block[p.offset:])
	p.offset += 2

	return value
}

func (p *UnpackParcel) UInt32() uint32 {
	value := p.order.Uint32(p.block[p.offset:])
	p.offset += 4

	return value
}

func (p *UnpackParcel) UInt64() uint64 {
	value := p.order.Uint64(p.block[p.offset:])
	p.offset += 8

	return value
}

func (p *UnpackParcel) Skip(offset int) {
	p.offset += offset
}

func (p *UnpackParcel) Get() []byte {
	return p.block
}
