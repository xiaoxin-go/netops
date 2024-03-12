package utils

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	JiraTimeFormat  = "2006-01-02T15:04:05.000+0800"
	LocalTimeFormat = "2006-01-02 15:04:05"
)

// UniqueId 生成唯一日志ID
func UniqueId() string {
	b := make([]byte, 48)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	return GetMd5String(base64.URLEncoding.EncodeToString(b))
}
func GetMd5String(s string) string {
	return GetMd5Bytes([]byte(s))
}

func GetMd5Bytes(bs []byte) string {
	h := md5.New()
	h.Write(bs)
	return hex.EncodeToString(h.Sum(nil))
}

// UTCTOTime 字符串转时间
func UTCTOTime(timeStr, format string) (time.Time, error) {
	return time.ParseInLocation(format, timeStr, time.Local)
}

// LocalTimeToString 时间转字符串
func LocalTimeToString() string {
	return time.Now().Format(LocalTimeFormat)
}

func GetFileMd5(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("打开文件: <%s> 发生异常: <%s>", path, err.Error())
	}
	m := md5.New()
	if _, err := io.Copy(m, file); err != nil {
		return "", fmt.Errorf("copy文件: <%s> 发生异常: <%s>", path, err.Error())
	}
	return hex.EncodeToString(m.Sum(nil)), nil
}
