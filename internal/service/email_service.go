package service

import (
	"context"
	"fmt"
	"net/smtp"
	"net/url"
	"user-auth-exam/infrastructure/config"
)

type EmailService interface {
	SendOTP(c context.Context, toEmail, otp string) error
	SendPasswordReset(c context.Context, toEmail, token string) error
}

type emailServiceImpl struct {
	CfgEmail    *config.EmailConfig
	AppUrl *config.AppConfig
}

func NewEmailServiceImpl(cfgEmail *config.EmailConfig, appUrl *config.AppConfig) EmailService {
	return &emailServiceImpl{
		CfgEmail: cfgEmail,
		AppUrl:   appUrl,
	}
}

func (e *emailServiceImpl) SendOTP(c context.Context, toEmail, otp string) error {
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

func (e *emailServiceImpl) SendPasswordReset(c context.Context, toEmail, token string) error {
	resetURL := fmt.Sprintf("%s/api/v1/auth/reset-password?token=%s", e.AppUrl.FrontendUrl, url.QueryEscape(token))

	subject := "Reset Your Password"

	body := fmt.Sprintf(
		"Hello,\r\n\r\n"+
			"We received a request to reset your password.\r\n\r\n"+
			"Please click the link below to reset your password:\r\n"+
			"%s\r\n\r\n"+
			"This link will expire in 10 minutes.\r\n\r\n"+
			"If you did not request a password reset, please ignore this email.\r\n",
		resetURL,
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
