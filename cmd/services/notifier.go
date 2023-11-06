package services

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

type EmailNotifier struct {
	mailbox string
	address string
	auth    smtp.Auth
}

func GetEmailNotifier(username, password, host, port, mailbox string) (EmailNotifier, error) {
	auth := smtp.PlainAuth("", username, password, host)
	return EmailNotifier{
		mailbox: mailbox,
		address: fmt.Sprintf("%s:%s", host, port),
		auth:    auth}, nil
}

func (e EmailNotifier) SendEmail(recipients []string, emailSubject, messageBody, mailbox string) {
	from := mailbox

	msg := fmt.Sprintf("To: %s \r\n", strings.Join(recipients, ",")) +
		fmt.Sprintf("Subject: %s \r\n\r\n", emailSubject) +
		fmt.Sprintf("%s \r\n", messageBody)

	err := smtp.SendMail(e.address, e.auth, from, recipients, []byte(msg))
	if err != nil {
		log.Printf("Warning: SendEmail - error while sending email %v", err)
	} else {
		log.Println("Email sent successfully")
	}
}
