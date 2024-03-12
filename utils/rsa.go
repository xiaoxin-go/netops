package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
)

func GenerateKey() (publicStr, privateStr string, err error) {
	// 2. 生成private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return
	}
	derPrivateKey := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derPrivateKey,
	}
	// 3. 获取private字符串
	privateBytes := pem.EncodeToMemory(block)
	// 4. 生成public key
	publicKey := &privateKey.PublicKey
	derPublicKey, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return
	}
	block = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derPublicKey,
	}
	//5. 获取public字符串
	publicBytes := pem.EncodeToMemory(block)
	privateStr = string(privateBytes)
	publicStr = string(publicBytes)
	return
}

func RsaEncrypt(password, publicPem string) (result string, err error) {
	// 加密
	block, _ := pem.Decode([]byte(publicPem))
	if block == nil {
		return
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return
	}
	pub := pubInterface.(*rsa.PublicKey)
	cipherText, err := rsa.EncryptPKCS1v15(rand.Reader, pub, []byte(password))
	if err != nil {
		return
	}
	result = base64.StdEncoding.EncodeToString(cipherText)
	return
}

func RsaDecrypt(password, privatePem string) (result string, err error) {
	passwordBytes, err := base64.StdEncoding.DecodeString(password)
	if err != nil {
		return
	}
	block, _ := pem.Decode([]byte(privatePem))
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return
	}
	passByte, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, passwordBytes)
	if err != nil {
		return
	}
	result = string(passByte)
	return
}
