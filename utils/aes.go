package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
)

func AesEncrypt(text string, key string) (result string, err error) {
	if text == "" {
		return "", nil
	}
	// 转成字节数组
	textBytes := []byte(text)
	k := []byte(key)

	// 分组密钥
	// NewCipher该函数限制了输入k的长度必须为16,24或者32
	block, err := aes.NewCipher(k)
	if err != nil {
		return
	}
	// 获取密钥的长度
	blockSize := block.BlockSize()
	// 补全码
	textBytes = PKCS7Padding(textBytes, blockSize)
	// 加密模式
	blockMode := cipher.NewCBCEncrypter(block, k[:blockSize])
	// 创建数组
	results := make([]byte, len(textBytes))
	blockMode.CryptBlocks(results, textBytes)
	result = base64.StdEncoding.EncodeToString(results)
	return
}

func AesDecrypt(text string, key string) (result string, err error) {
	if text == "" {
		return "", nil
	}
	// 转成字节数组
	textByte, _ := base64.StdEncoding.DecodeString(text)
	k := []byte(key)
	// 分组秘钥
	block, err := aes.NewCipher(k)
	if err != nil {
		return
	}
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 加密模式
	blockMode := cipher.NewCBCDecrypter(block, k[:blockSize])
	// 创建数组
	orig := make([]byte, len(textByte))
	// 解密
	blockMode.CryptBlocks(orig, textByte)
	// 去补全码
	orig = PKCS7UnPadding(orig)
	result = string(orig)
	return
}

// PKCS7Padding 补码 AES加密数据块分组长度必须为128bit(byte[16])，密钥长度可以是128bit(byte[16])、192bit(byte[24])、256bit(byte[32])中的任意一个
func PKCS7Padding(cipherText []byte, blockSize int) []byte {
	padding := blockSize - len(cipherText)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(cipherText, padText...)
}

// PKCS7UnPadding 去码
func PKCS7UnPadding(textBytes []byte) []byte {
	length := len(textBytes)
	unPadding := int(textBytes[length-1])
	return textBytes[:(length - unPadding)]
}
