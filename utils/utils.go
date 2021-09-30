package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	mathrand "math/rand"
	"reflect"
	"runtime"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/zenazn/pkcs7pad"
)

func MD5(src string) string {
	h := md5.New()
	_, _ = h.Write([]byte(src))
	return hex.EncodeToString(h.Sum([]byte("")))
}

func Sha256(str, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	io.WriteString(h, str)
	return hex.EncodeToString(h.Sum(nil))
}

func Sign(url, body, app_secret, token string) string {
	var hash_key = app_secret + token
	return Sha256(url+body, hash_key)
}

func UUID() string {
	return uuid.NewV4().String()
}

func JsonMarshalString(v interface{}) string {
	return string(JsonMarshalByte(v))
}

func JsonMarshalByte(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		//logger.Error("utils.JsonMarshalByte:" + err.Error())
		return nil
	}

	return data
}

func JoinURLPath(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// RandString 生成随机字符串
func RandString(len int) string {
	r := mathrand.New(mathrand.NewSource(time.Now().Unix()))
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		b := r.Intn(26) + 65
		bytes[i] = byte(b)
	}
	return string(bytes)
}

func GenOrderId() string {
	tmStr := time.Now().Format("060102T150405MST")
	tailfix := RandString(6)
	return tmStr + tailfix
}

func dbFields(values interface{}) []string {
	v := reflect.ValueOf(values)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	fields := []string{}
	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			_ = f
			field := v.Type().Field(i).Tag.Get("db")
			if field != "" {
				fields = append(fields, field)
			}
		}
		return fields
	}
	if v.Kind() == reflect.Map {
		for _, keyv := range v.MapKeys() {
			fields = append(fields, keyv.String())
		}
		return fields
	}
	panic(fmt.Errorf("dbFields requires a struct or a map, found: %s", v.Kind().String()))
}

// 函数执行时间
// defer Elapsed.Stop()
type elapsedTime struct {
	start time.Time
}

func (e *elapsedTime) Stop() string {
	elapsed := time.Now().Sub(e.start)
	pc, _, _, _ := runtime.Caller(1)
	f := runtime.FuncForPC(pc)
	return fmt.Sprintf(f.Name()+"耗时:%v", elapsed)
}

func Elapsed() interface {
	Stop() string
} {
	var e elapsedTime
	e.start = time.Now()
	return &e
}

func WaitRandMS(minMS int, maxMS int) {
	if minMS >= maxMS {
		time.Sleep(time.Duration(minMS) * time.Millisecond)
		return
	}

	step := 100
	x := minMS + (rand.Intn((maxMS-minMS)*step))/step
	wait := time.Duration(x) * time.Millisecond
	time.Sleep(wait)
}

func InSliceString(str string, arr []string) bool {
	for _, s := range arr {
		if strings.Contains(str, s) {
			return true
		}
	}

	return false
}

func FormatUnixTime(timestamp int64, layout string) string {
	if timestamp <= 0 {
		return ""
	}

	tm := time.Unix(timestamp, 0)
	return tm.Format(layout)
}

func FormatMilliSecond(timestamp int64, layout string) string {
	if timestamp <= 0 {
		return ""
	}

	tm := time.Unix(timestamp/1000, timestamp%1000*1e6)
	return tm.Format(layout)
}

var commonIV = []byte{0x5a, 0xe3, 0xf0, 0x46, 0xcc, 0x11, 0xb4, 0x45, 0x09, 0x04, 0x47, 0x58, 0x00, 0xbf, 0x88, 0xd5}

func AesEncrypt(src, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// src = PKCS7Pad(src)
	src = pkcs7pad.Pad(src, aes.BlockSize)
	dst := make([]byte, len(src))
	blockMode := cipher.NewCBCEncrypter(block, commonIV)
	blockMode.CryptBlocks(dst, src)

	return dst, nil
}

func AesDecrypt(src, key []byte) ([]byte, error) {
	//logger := server.SugarLogger("AesDecrypt")
	if len(src)%16 != 0 {
		//logger.Errorf("data is can not exact division 16")
		return nil, errors.New("data is can not exact division 16")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	dst := make([]byte, len(src))
	blockMode := cipher.NewCBCDecrypter(block, commonIV)
	blockMode.CryptBlocks(dst, src)
	dst2 := make([]byte, len(src))
	for i, _ := range dst2 {
		if i < len(dst) {
			dst2[i] = dst[i]
		}
	}

	dst, err = pkcs7pad.Unpad(dst)
	if err != nil {
		//logger.Errorf("error unpadding bytes: %s, error: %s",hex.EncodeToString(dst2), err.Error())

		if dst2 = PKCS7Unpad(dst2); dst2 != nil && len(dst2) > 0 {
			return dst2, nil
		}

		return nil, err
	}

	return dst, nil
}

// PKCS7Unpad() removes any potential PKCS7 padding added.
func PKCS7Unpad(data []byte) []byte {
	dataLen := len(data)
	// Edge case
	if dataLen == 0 {
		return nil
	}
	// the last byte indicates the length of the padding to remove
	paddingLen := int(data[dataLen-1])

	// padding length can only be between 1-15
	if paddingLen < 16 {
		return data[:dataLen-paddingLen]
	}
	return data
}

// PKCS7Pad() pads an byte array to be a multiple of 16
// http://tools.ietf.org/html/rfc5652#section-6.3
func PKCS7Pad(data []byte) []byte {
	dataLen := len(data)

	var validLen int
	if dataLen%16 == 0 {
		validLen = dataLen
	} else {
		validLen = int(dataLen/16+1) * 16
	}

	paddingLen := validLen - dataLen
	// The length of the padding is used as the byte we will
	// append as a pad.
	bitCode := byte(paddingLen)
	padding := make([]byte, paddingLen)
	for i := 0; i < paddingLen; i++ {
		padding[i] = bitCode
	}
	return append(data, padding...)
}

func IsLanIp(ip string) bool {
	ipAddr := strings.Split(ip, `.`)
	if len(ipAddr) < 4 {
		return false
	}

	pre := ipAddr[0] + "."
	if pre == "10." || pre == "127." || pre == "192." {
		return true
	}
	return false
}
