package services

import (
	"fmt"
	"net/smtp"
	"os"
)

type EmailService struct {
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	fromEmail    string
	fromName     string
	frontendURL  string
}

func NewEmailService() *EmailService {
	return &EmailService{
		smtpHost:     os.Getenv("SMTP_HOST"),
		smtpPort:     os.Getenv("SMTP_PORT"),
		smtpUsername: os.Getenv("SMTP_USERNAME"),
		smtpPassword: os.Getenv("SMTP_PASSWORD"),
		fromEmail:    os.Getenv("SMTP_FROM_EMAIL"),
		fromName:     os.Getenv("SMTP_FROM_NAME"),
		frontendURL:  os.Getenv("FRONTEND_URL"),
	}
}

// SendVerificationEmail sends email verification link
func (es *EmailService) SendVerificationEmail(toEmail, fullName, token string) error {
	verifyURL := fmt.Sprintf("%s/auth/verify-email?token=%s", es.frontendURL, token)

	subject := "Verify your InvestorCenter.ai account"
	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif;">
			<h2>Welcome to InvestorCenter.ai, %s!</h2>
			<p>Thanks for signing up. Please verify your email address by clicking the link below:</p>
			<p><a href="%s" style="background-color: #4CAF50; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Verify Email</a></p>
			<p>Or copy and paste this URL into your browser:</p>
			<p>%s</p>
			<p>This link will expire in 24 hours.</p>
			<p>If you didn't create an account, you can safely ignore this email.</p>
		</body>
		</html>
	`, fullName, verifyURL, verifyURL)

	return es.sendEmail(toEmail, subject, body)
}

// SendPasswordResetEmail sends password reset link
func (es *EmailService) SendPasswordResetEmail(toEmail, fullName, token string) error {
	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", es.frontendURL, token)

	subject := "Reset your InvestorCenter.ai password"
	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif;">
			<h2>Password Reset Request</h2>
			<p>Hi %s,</p>
			<p>We received a request to reset your password. Click the link below to reset it:</p>
			<p><a href="%s" style="background-color: #2196F3; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Reset Password</a></p>
			<p>Or copy and paste this URL into your browser:</p>
			<p>%s</p>
			<p>This link will expire in 1 hour.</p>
			<p>If you didn't request a password reset, you can safely ignore this email.</p>
		</body>
		</html>
	`, fullName, resetURL, resetURL)

	return es.sendEmail(toEmail, subject, body)
}

// sendEmail is a helper to send HTML emails via SMTP
func (es *EmailService) sendEmail(to, subject, htmlBody string) error {
	// If SMTP is not configured, skip sending email (for development)
	if es.smtpHost == "" || es.smtpPassword == "" {
		fmt.Printf("SMTP not configured. Skipping email to %s\n", to)
		fmt.Printf("Subject: %s\n", subject)
		return nil
	}

	fmt.Printf("Attempting to send email to %s via %s:%s\n", to, es.smtpHost, es.smtpPort)

	from := fmt.Sprintf("%s <%s>", es.fromName, es.fromEmail)
	msg := []byte(fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", from, to, subject, htmlBody))

	auth := smtp.PlainAuth("", es.smtpUsername, es.smtpPassword, es.smtpHost)
	addr := fmt.Sprintf("%s:%s", es.smtpHost, es.smtpPort)

	err := smtp.SendMail(addr, auth, es.fromEmail, []string{to}, msg)
	if err != nil {
		fmt.Printf("ERROR sending email to %s: %v\n", to, err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Printf("Successfully sent email to %s\n", to)
	return nil
}
