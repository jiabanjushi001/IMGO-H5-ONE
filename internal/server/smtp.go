package server

import (
	"crypto/tls"
	"errors"
	"net/smtp"
)

func startSMTP(c *smtp.Client, host, user, password string) error {
	if ok, _ := c.Extension("STARTTLS"); !ok {
		return errors.New("SMTP 必须支持 STARTTLS")
	}
	if e := c.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); e != nil {
		return e
	}
	if user != "" {
		return c.Auth(smtp.PlainAuth("", user, password, host))
	}
	return nil
}
