package utils

import (
	"crypto/tls"
	"fmt"
	"gopkg.in/gomail.v2"
	"netops/conf"
)

func SendMail(subject string, body string, filename string) error {
	data := conf.Config.Email
	email := gomail.NewMessage()
	email.SetHeader("From", data.Sender)                             // 发件人
	email.SetHeader("To", data.To...)                                // 发送给多个用户
	email.SetHeader("Subject", fmt.Sprintf("Netops: <%s>", subject)) // 邮件主题
	email.SetBody("text/html", body)                                 // 邮件正文

	// 添加附件
	if len(filename) > 0 {
		email.Attach(filename)
	}

	d := gomail.NewDialer(data.Host, data.Port, data.Username, data.Password)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	err := d.DialAndSend(email)
	return err
}
