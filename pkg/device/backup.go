package device

import (
	"archive/zip"
	"fmt"
	"go.uber.org/zap"
	"netops/conf"
	netApi2 "netops/grpc_client/protobuf/net_api"
	"netops/model"
	"netops/utils"
	"os"
	"time"
)

// Backup 配置备份
func (b *base) Backup() error {
	l := zap.L().With(zap.String("func", "backup"), zap.Int("device_id", b.DeviceId))
	l.Info("1. 获取设备配置------------>")
	nt := time.Now().Format("200601021504")
	text, e := b.getBackupConfig()
	if e != nil {
		l.Error("调用接口失败", zap.Error(e))
		return e
	}
	back := model.TDeviceBackup{DeviceId: b.DeviceId, Size: len(text), Filename: fmt.Sprintf("netops/%s/%s.zip", b.device.Name, nt)}

	l.Info("2. 写入压缩文件-------------->")
	zipFilename := fmt.Sprintf("data/%s-%s.zip", b.device.Name, nt)
	md5, e := b.zipFile(zipFilename, []byte(text))
	if e != nil {
		l.Error("压缩配置文件失败", zap.Error(e))
		return e
	}
	back.Md5 = md5
	l.Info("3. 上传到minio---------->")
	if e := b.updateMinio(back.Filename, zipFilename); e != nil {
		l.Error("上传minio失败", zap.Error(e))
		return e
	}
	l.Info("4. 保存备份信息---------->")
	if e := back.Create(); e != nil {
		l.Error("保存备份配置失败", zap.Error(e))
		return e
	}
	return nil
}

func (b *base) zipFile(filename string, body []byte) (string, error) {
	zipFile, _ := os.Create(filename)
	defer zipFile.Close()
	w := zip.NewWriter(zipFile)
	defer w.Close()
	fw, e := w.Create(fmt.Sprintf("content.txt"))
	if e != nil {
		return "", fmt.Errorf("创建文件写入zip失败, err: %w", e)
	}
	_, err := fw.Write(body)
	if err != nil {
		return "", fmt.Errorf("写入文件数据失败, err: %w")
	}
	md5, e := utils.GetFileMd5(filename)
	if e != nil {
		return "", fmt.Errorf("读取文件md5值失败, err: %w", e)
	}
	return md5, nil
}

// 上传文件到minio
func (b *base) updateMinio(objectName, filepath string) error {
	m := utils.NewMinioDefault()
	// netops/device_name/filename.zip
	if err := m.UploadFile(conf.Config.Minio.Bucket, objectName, filepath); err != nil {
		return err
	}
	return nil
}

func (b *base) getBackupConfig() (string, error) {
	commands := []*netApi2.Command{
		{Id: 1, Cmd: b.backupCommand},
	}
	result, e := b.send(commands)
	if e != nil {
		return "", e
	}
	return result[0].Result, nil
}

func NewBackupHandler(backupId int) *backupHandler {
	r := &backupHandler{id: backupId}
	r.init()
	return r
}

type backupHandler struct {
	id   int
	data *model.TDeviceBackup
	Err  error
}

func (h *backupHandler) init() {
	backup := model.TDeviceBackup{}
	if e := backup.FirstById(h.id); e != nil {
		h.Err = e
		return
	}
	h.data = &backup
}
func (h *backupHandler) Download() ([]byte, error) {
	if h.Err != nil {
		return nil, h.Err
	}
	l := zap.L().With(zap.String("func", "BackupDownload"), zap.Int("backup_id", h.id))
	l.Info("1. 从minio获取文件-------------------->")
	m := utils.NewMinioDefault()
	result, err := m.GetFile(conf.Config.Minio.Bucket, h.data.Filename)
	if err != nil {
		return nil, err
	}
	l.Info("2. 对比文件md5---------------->")
	if h.verifyMd5(result, h.data.Md5) {
		return nil, fmt.Errorf("文件MD5值不一致")
	}
	return result, nil
}

// 校对MD5值
func (h *backupHandler) verifyMd5(body []byte, md5 string) bool {
	return utils.GetMd5Bytes(body) == md5
}
