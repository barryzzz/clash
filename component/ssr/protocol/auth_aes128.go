package protocol

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"io"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/Dreamacro/clash/common/parcel"
	"github.com/Dreamacro/clash/common/pool"
	"github.com/Dreamacro/clash/component/ssr/digest"
	"github.com/Dreamacro/clash/component/ssr/utils"

	"github.com/Dreamacro/go-shadowsocks2/core"
	"github.com/Dreamacro/go-shadowsocks2/shadowstream"
)

const authAes128Md5 = "auth_aes128_md5"
const authAes128Sha1 = "auth_aes128_sha1"

// rand byte + header hmac + user id + encrypted header + rand hmac (+ data ) + data hmac
const authAes128HeaderPaddingSize = 1 + 6 + 4 + 16 + 4 + 4
const authAes128BlockSize = 4096

type authAes128 struct {
	// shadowscksr auth context
	mutex        sync.Mutex
	clientID     uint32
	connectionID uint32

	// shadowsocks.Kdf(password)
	PasswordKey []byte

	// shadowsocksr auth key & password
	UserId  uint32
	UserKey []byte

	Salt string
	Hmac digest.HmacMethod
}

type authAes128Conn struct {
	*shadowstream.Conn
	*authAes128

	headerSent     bool
	sendBlockID    uint32
	receiveBlockID uint32

	remainBytes       []byte
	remainBytesOffset int
}

func (a *authAes128) StreamConn(conn net.Conn) (net.Conn, error) {
	stream, ok := conn.(*shadowstream.Conn)
	if !ok {
		return nil, ErrUnsupportedCipher
	}

	return &authAes128Conn{
		Conn:           stream,
		authAes128:     a,
		headerSent:     false,
		sendBlockID:    1,
		receiveBlockID: 1,
	}, nil
}

func (a *authAes128) PacketConn(net.PacketConn) (net.PacketConn, error) {
	return nil, ErrNotImplement
}

func (a *authAes128) obtainAuthID() (uint32, uint32) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.clientID == 0 || a.connectionID > 0xFF000000 {
		a.clientID = 1 + rand.Uint32()
		a.connectionID = a.clientID & 0x00FFFFFF
	}

	client := a.clientID
	conn := a.connectionID

	a.connectionID++

	return client, conn
}

func (c *authAes128Conn) Write(buf []byte) (int, error) {
	if !c.headerSent {
		authSize := utils.MinInt(1200, len(buf))

		if err := c.writeAuthData(buf[:authSize]); err != nil {
			return 0, err
		}

		c.headerSent = true
		buf = buf[authSize:]
	}

	return len(buf), utils.ForEachBlock(buf, authAes128BlockSize, c.writeBlockData)
}

func (c *authAes128Conn) writeAuthData(authData []byte) error {
	randSize := 0
	if len(authData) > 400 {
		randSize = rand.Intn(512)
	} else {
		randSize = rand.Intn(1024)
	}

	// build auth header

	clientId, connId := c.obtainAuthID()
	blockSize := randSize + len(authData) + authAes128HeaderPaddingSize

	// alloc header data and encrypted header data
	// header data: 0 - 16 bytes
	// encrypted : 16 - 32 bytes

	p := parcel.Pack(make([]byte, 16*2), binary.LittleEndian)

	p.UInt32(uint32(time.Now().Unix())) // 4 bytes
	p.UInt32(clientId)                  // 4 bytes
	p.UInt32(connId)                    // 4 bytes
	p.UInt16(uint16(blockSize))         // 2 bytes
	p.UInt16(uint16(randSize))          // 2 bytes

	header := p.Get()

	// initialize header encrypter

	encryptPassword := base64.StdEncoding.EncodeToString(c.UserKey) + c.Salt
	encryptKey := core.Kdf(encryptPassword, 16)

	encryptBlock, err := aes.NewCipher(encryptKey)
	if err != nil {
		return err
	}

	// source 0-16 bytes
	// dest   16-32 bytes
	cipher.NewCBCEncrypter(encryptBlock, make([]byte, 16)).CryptBlocks(header[16:], header[:16])

	block := pool.Get(blockSize)
	defer pool.Put(block)

	p = parcel.Pack(block, binary.LittleEndian)

	p.Byte(byte(rand.Uint32()))                            // 1 byte
	p.Skip(6)                                              // 6 bytes  padding for first byte hmac
	p.UInt32(c.UserId)                                     // 4 bytes
	p.Bytes(header[16:])                                   // 16 bytes encrypted header
	p.Skip(4)                                              // 4 bytes  padding for header hmac
	p.NextBytes(randSize, func(i []byte) { rand.Read(i) }) // fill random block
	p.Bytes(authData)                                      // data

	// generate data hmac

	iv, err := c.Conn.ObtainWriteIV()
	if err != nil {
		return err
	}

	headerHmacKey := append(append(make([]byte, 0, len(iv)+len(c.PasswordKey)), iv...), c.PasswordKey...)

	headerHmac := c.Hmac(headerHmacKey, block[:1])
	copy(block[1:7], headerHmac) // random byte hmac

	randHeaderHmac := c.Hmac(headerHmacKey, block[7:27])
	copy(block[27:27+4], randHeaderHmac) // header hmac

	dataHmac := c.Hmac(c.UserKey, block[:len(block)-4])
	copy(block[len(block)-4:], dataHmac) // data block hmac

	_, err = c.Conn.Write(block)

	return err
}

func (c *authAes128Conn) writeBlockData(data []byte) error {
	randSize := 1
	if len(data) <= 1200 {
		if c.sendBlockID > 4 {
			randSize += rand.Intn(32)
		} else {
			if len(data) > 900 {
				randSize += rand.Intn(128)
			} else {
				randSize += rand.Intn(512)
			}
		}
	}

	// build block data

	block := pool.Get(randSize + len(data) + 8) // 8 bytes header & hmac
	defer pool.Put(block)

	p := parcel.Pack(block, binary.LittleEndian)

	p.UInt16(uint16(len(block)))                                   // 2 bytes
	p.Skip(2)                                                      // 2 bytes hmac(block length)
	p.NextBytes(randSize, func(bytes []byte) { rand.Read(bytes) }) // randSize bytes, fill random block
	p.Bytes(data)                                                  // data
	p.Skip(4)                                                      // hmac(data)

	// set rand size

	if randSize < 128 {
		block[4] = byte(randSize)
	} else {
		block[4] = 0xFF
		binary.LittleEndian.PutUint16(block[5:], uint16(randSize))
	}

	// generate data hmac

	hmacKey := parcel.Pack(make([]byte, len(c.UserKey)+4), binary.LittleEndian)
	hmacKey.Bytes(c.UserKey)
	hmacKey.UInt32(c.sendBlockID)

	lengthHmac := c.Hmac(hmacKey.Get(), block[0:2])
	copy(block[2:4], lengthHmac)

	blockHmac := c.Hmac(hmacKey.Get(), block[:len(block)-4])
	copy(block[len(block)-4:], blockHmac)

	_, err := c.Conn.Write(block)

	return err
}

func (c *authAes128Conn) Read(output []byte) (int, error) {
	remain := c.remainBytes

	if remain != nil {
		if len(remain)-c.remainBytesOffset > 0 {
			n := copy(output, remain)
			c.remainBytesOffset += n
		} else {
			pool.Put(remain)
			c.remainBytes = nil
		}
	}

	hmacKey := parcel.Pack(make([]byte, len(c.UserKey)+4), binary.LittleEndian)

	hmacKey.Bytes(c.UserKey)
	hmacKey.UInt32(c.receiveBlockID)

	if len(output) < 5 {
		return 0, io.ErrShortBuffer
	}

	// unpack header

	header := parcel.Unpack(output, binary.LittleEndian)
	if _, err := io.ReadFull(c.Conn, header.Get()[:5]); err != nil {
		return 0, err
	}

	blockLength := header.UInt16()      // 2 bytes, block length
	lengthHmac := header.Bytes(2)       // 2 bytes, hmac(block length)
	randLength := uint16(header.Byte()) // rand size

	// valid length & length hmac

	h := c.Hmac(hmacKey.Get(), header.Get()[:2])
	if bytes.Compare(h[0:2], lengthHmac[0:2]) != 0 {
		return 0, ErrAuthFailure
	}

	c.receiveBlockID++

	dataLength := blockLength - 4 - 4 - 1 // block length - header - hmac - random size

	if randLength >= 255 {
		if err := binary.Read(c.Conn, binary.LittleEndian, &randLength); err != nil {
			return 0, err
		}

		randLength -= 1
		dataLength -= randLength

		randLength -= 2
	} else {
		randLength -= 1
		dataLength -= randLength
	}

	// read random block

	for randLength > 0 {
		size := utils.MinInt(int(randLength), len(output))

		// reuse output buffer
		if _, err := io.ReadFull(c.Conn, output[:size]); err != nil {
			return 0, err
		}

		randLength -= uint16(size)
	}

	// read data block

	readLength := utils.MinInt(len(output), int(dataLength))

	output = output[:readLength] // data block

	if _, err := io.ReadFull(c.Conn, output); err != nil {
		return 0, err
	}

	if int(dataLength) > readLength {
		remain = pool.Get(int(dataLength) - readLength)

		if _, err := io.ReadFull(c.Conn, remain); err != nil {
			return 0, err
		}

		c.remainBytes = remain
	}

	// ignore data block hmac

	if _, err := io.ReadFull(c.Conn, make([]byte, 4)); err != nil {
		return 0, err
	}

	return len(output), nil
}

func newAuthAes128(digestMethod digest.Method, hmacMethod digest.HmacMethod, salt string, passwordKey []byte, userId uint32, userKey string) Protocol {
	var hashedUserKey []byte

	if userId == 0 {
		userId = rand.Uint32()
	}

	if userKey != "" {
		hashedUserKey = digestMethod([]byte(userKey))
	} else {
		hashedUserKey = passwordKey
	}

	return &authAes128{
		PasswordKey: passwordKey,
		UserId:      userId,
		UserKey:     hashedUserKey,
		Hmac:        hmacMethod,
		Salt:        salt,
	}
}

func NewAuthAes128Sha1(passwordKey []byte, userId uint32, userKey string) Protocol {
	return newAuthAes128(digest.SHA1Sum, digest.HmacSHA1, authAes128Sha1, passwordKey, userId, userKey)
}

func NewAuthAes128Md5(passwordKey []byte, userId uint32, userKey string) Protocol {
	return newAuthAes128(digest.MD5Sum, digest.HmacMD5, authAes128Md5, passwordKey, userId, userKey)
}
