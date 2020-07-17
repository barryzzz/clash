// from https://github.com/sh4d0wfiend/go-shadowsocksr/blob/master/tools/encrypt.go

package digest

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
)

type HmacMethod func(key []byte, data []byte) []byte
type Method func(data []byte) []byte

func HmacMD5(key []byte, data []byte) []byte {
	hmacMD5 := hmac.New(md5.New, key)
	hmacMD5.Write(data)
	return hmacMD5.Sum(nil)[:10]
}

func HmacSHA1(key []byte, data []byte) []byte {
	hmacSHA1 := hmac.New(sha1.New, key)
	hmacSHA1.Write(data)
	return hmacSHA1.Sum(nil)[:10]
}

func MD5Sum(d []byte) []byte {
	h := md5.New()
	h.Write(d)
	return h.Sum(nil)
}

func SHA1Sum(d []byte) []byte {
	h := sha1.New()
	h.Write(d)
	return h.Sum(nil)
}
