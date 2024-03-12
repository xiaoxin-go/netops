package utils

import (
	"fmt"
	"github.com/minio/minio-go"
	"io/ioutil"
	"netops/conf"
)

type Minio struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Secure    bool
	client    *minio.Client
	error     error
}

func NewMinioDefault() *Minio {
	return NewMinio(conf.Config.Minio.Endpoint, conf.Config.Minio.AccessKey, conf.Config.Minio.SecretKey, conf.Config.Minio.Secure)
}

func NewMinio(endpoint, accessKey, secretKey string, secure bool) *Minio {
	m := &Minio{
		Endpoint:  endpoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Secure:    secure,
	}
	m.Connect()
	return m
}

func (m *Minio) Connect() {
	client, err := minio.New(m.Endpoint, m.AccessKey, m.SecretKey, m.Secure)
	if err != nil {
		m.error = fmt.Errorf("连接minio服务器:<%s>异常:  <%s>", m.SecretKey, err.Error())
	}
	m.client = client
}

func (m *Minio) UploadFile(bucket, filename, filePath string) error {
	// filename: 上传到minio的文件名
	// filePath: 本地的文件路径
	if m.error != nil {
		return m.error
	}
	result, err := m.client.BucketExists(bucket)
	if err != nil {
		return fmt.Errorf("校验bucket<%s>是否存在异常: <%s>", bucket, err.Error())
	}
	if !result {
		if err := m.client.MakeBucket(bucket, ""); err != nil {
			return fmt.Errorf("创建bucket<%s>发生异常: <%s>", bucket, err.Error())
		}
	}
	opts := minio.PutObjectOptions{ContentType: "application/zip"}
	if _, err := m.client.FPutObject(bucket, filename, filePath, opts); err != nil {
		return fmt.Errorf("文件:<%s>上传到minio发生异常: <%s>", filePath, err.Error())
	}
	return nil
}

func (m *Minio) DownloadFile(bucket, filename, filePath, md5Str string) error {
	if m.error != nil {
		return m.error
	}
	if err := m.client.FGetObject(bucket, filename, filePath, minio.GetObjectOptions{}); err != nil {
		return fmt.Errorf("从minio下获取文件<%s>发生异常: <%s>", filename, err.Error())
	}
	if md5, err := GetFileMd5(filePath); err != nil {
		return err
	} else {
		if md5Str != "" && md5 != md5Str {
			return fmt.Errorf("文件md5:<%s>与<%s>不一致", md5, md5Str)
		}
	}
	return nil
}

func (m *Minio) GetFile(bucket, filename string) ([]byte, error) {
	if m.error != nil {
		return nil, m.error
	}
	object, err := m.client.GetObject(bucket, filename, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("从minio获取文件异常: <%s>", err.Error())
	}
	b, err := ioutil.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("从minio下载文件后读取文件数据异常: <%s>", err.Error())
	}
	return b, nil
}
