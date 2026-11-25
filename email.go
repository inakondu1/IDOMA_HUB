package main

import (
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
)

func sendPasswordResetEmail(to, resetLink string) error {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	username := strings.TrimSpace(os.Getenv("SMTP_USER"))
	password := os.Getenv("SMTP_PASSWORD")
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))

	if host == "" || port == "" || username == "" || password == "" || from == "" {
		return fmt.Errorf("SMTP configuration is incomplete")
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return fmt.Errorf("invalid SMTP port")
	}

	parsedFrom, err := mail.ParseAddress(from)
	if err != nil {
		return fmt.Errorf("invalid sender address")
	}

	parsedTo, err := mail.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("invalid recipient address")
	}

	if !strings.HasPrefix(resetLink, "https://") {
		return fmt.Errorf("password reset link must use HTTPS")
	}

	auth := smtp.PlainAuth("", username, password, host)

	subject := "Subject: IDOMA HUB Password Reset\r\n"
	headers := "MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"From: " + parsedFrom.String() + "\r\n" +
		"To: " + parsedTo.String() + "\r\n" +
		"X-Content-Type-Options: nosniff\r\n\r\n"

	body := "Hello,\n\n" +
		"We received a request to reset your IDOMA HUB password.\n\n" +
		"Use the link below to choose a new password:\n\n" +
		resetLink + "\n\n" +
		"This link expires in 30 minutes and can only be used once.\n\n" +
		"If you did not request this, you can ignore this email.\n\n" +
		"IDOMA HUB Team"

	message := []byte(subject + headers + body)

	address := net.JoinHostPort(host, strconv.Itoa(portNumber))

	return smtp.SendMail(
		address,
		auth,
		parsedFrom.Address,
		[]string{parsedTo.Address},
		message,
	)
}
