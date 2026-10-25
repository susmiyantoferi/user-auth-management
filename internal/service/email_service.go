package service

import (
	"context"
	"fmt"
	"net/smtp"
	"user-auth-exam/infrastructure/config"
)

type EmailService interface {
	SendEmail(c context.Context, toEmail, otp string) error
}

type emailServiceImpl struct {
	CfgEmail *config.EmailConfig
}

func NewEmailServiceImpl(cfgEmail *config.EmailConfig) EmailService {
	return &emailServiceImpl{
		CfgEmail: cfgEmail,
	}
}

func (e *emailServiceImpl) SendEmail(c context.Context, toEmail, otp string) error {
	subject := "Verify Your Email OTP"

	//text email
	body := fmt.Sprintf(
		"Hello,\r\n\r\n"+
			"Your verification code is: %s\r\n\r\n"+
			"This code will expire in 5 minutes.\r\n\r\n"+
			"If you did not create this account, please ignore this email.\r\n",
		otp,
	)

	//prepare send email
	message := []byte(
		"From: " + e.CfgEmail.From + "\r\n" +
			"To: " + toEmail + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" +
			body,
	)

	//auth untuk email
	auth := smtp.PlainAuth(
		"",
		e.CfgEmail.Username,
		e.CfgEmail.Password,
		e.CfgEmail.Host,
	)

	addr := fmt.Sprintf("%s:%d", e.CfgEmail.Host, e.CfgEmail.Port)

	//send email
	if err := smtp.SendMail(addr, auth, e.CfgEmail.From, []string{toEmail}, message); err != nil {
		return err
	}

	return nil
}
