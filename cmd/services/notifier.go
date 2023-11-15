package services

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

type Notifier interface {
	Notify(messageSubject, messageBody string)
}

func SendNotification(n Notifier, messageSubject, messageBody string) {
	n.Notify(messageSubject, messageBody)
}

type EmailNotifier struct {
	recipients []string
	mailbox    string
	address    string
	auth       smtp.Auth
}

func GetEmailNotifier(recipients []string, username, password, host, port, mailbox string) (EmailNotifier, error) {
	auth := smtp.PlainAuth("", username, password, host)
	return EmailNotifier{
		recipients: recipients,
		mailbox:    mailbox,
		address:    fmt.Sprintf("%s:%s", host, port),
		auth:       auth}, nil
}

func (e EmailNotifier) Notify(messageSubject, messageBody string) {
	from := e.mailbox

	msg := fmt.Sprintf("To: %s \r\n", strings.Join(e.recipients, ",")) +
		fmt.Sprintf("Subject: %s \r\n\r\n", messageSubject) +
		fmt.Sprintf("%s \r\n", messageBody)

	err := smtp.SendMail(e.address, e.auth, from, e.recipients, []byte(msg))
	if err != nil {
		log.Printf("Warning: SendEmail - error while sending email %v", err)
	} else {
		log.Println("Email sent successfully")
	}
}
