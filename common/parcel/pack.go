package parcel

import "encoding/binary"

type PackParcel struct {
	block  []byte
	offset int
	order  binary.ByteOrder
}

func Pack(block []byte, byteOrder binary.ByteOrder) *PackParcel {
	return &PackParcel{
		block:  block,
		offset: 0,
		order:  byteOrder,
	}
}

func (p *PackParcel) Byte(b byte) {
	p.block[p.offset] = b
	p.offset += 1
}

func (p *PackParcel) Bytes(bytes []byte) {
	copy(p.block[p.offset:], bytes)
	p.offset += len(bytes)
}

func (p *PackParcel) UInt16(value uint16) {
	p.order.PutUint16(p.block[p.offset:], value)
	p.offset += 2
}

func (p *PackParcel) UInt32(value uint32) {
	p.order.PutUint32(p.block[p.offset:], value)
	p.offset += 4
}

func (p *PackParcel) UInt64(value uint64) {
	p.order.PutUint64(p.block[p.offset:], value)
	p.offset += 8
}

func (p *PackParcel) Skip(offset int) {
	p.offset += offset
}

func (p *PackParcel) NextBytes(n int, h func([]byte)) {
	h(p.block[p.offset : p.offset+n])
	p.offset += n
}

func (p *PackParcel) Get() []byte {
	return p.block
}
