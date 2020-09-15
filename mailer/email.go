package jdchaimailer

import (
	"bytes"
	"html/template"
	"log"
	"net/smtp"
)

var from string
var smtpAuth smtp.Auth

//EmailRequest struct
type EmailRequest struct {
	from    string
	to      []string
	subject string
	body    string
}

//Init will initalize the mailer package with auth
func Init(auth smtp.Auth, mailer string) {
	smtpAuth = auth
	from = mailer
}

//NewEmailRequest constructor
func NewEmailRequest(to []string, from, subject, body string) *EmailRequest {
	return &EmailRequest{
		to:      to,
		from:    from,
		subject: subject,
		body:    body,
	}
}

//SendEmail smtp
func (r *EmailRequest) SendEmail() (bool, error) {
	if smtpAuth == nil {
		log.Println("smtp error: missing service smtp auth!")
	}
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	subject := "Subject: " + r.subject + "!\n"
	msg := []byte(subject + mime + "\n" + r.body)
	addr := "smtp.gmail.com:587"

	if err := smtp.SendMail(addr, smtpAuth, r.from, r.to, msg); err != nil {
		return false, err
	}
	return true, nil
}

//ParseTemplate parse a template for email
func (r *EmailRequest) ParseTemplate(templateFileName string, data interface{}) error {
	t, err := template.ParseFiles(templateFileName)
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	if err = t.Execute(buf, data); err != nil {
		return err
	}
	r.body = buf.String()
	return nil
}

// SendWelcomRegistration sends a registration welcom email to an invited user
func SendWelcomRegistration(group, email, token string) {
	templateData := struct {
		Group string
		URL   string
	}{
		Group: group,
		URL:   token,
	}
	address := []string{email}
	r := NewEmailRequest(address, from, "Welcome to JDScheduler!", "")
	if err := r.ParseTemplate("mailer/welcome.html", templateData); err == nil {
		if _, err := r.SendEmail(); err != nil {
			log.Println("smtp error: " + err.Error())
		}
	} else {
		log.Println("welcome email template parse failed")
	}
}

// SendGroupInvite sends a group invite welcome email
func SendGroupInvite(group, name, email string) {
	templateData := struct {
		Group string
		Name  string
	}{
		Group: group,
		Name:  name,
	}
	address := []string{email}
	r := NewEmailRequest(address, from, "Invited to JDScheduler Group: "+group, "")
	if err := r.ParseTemplate("mailer/groupinvite.html", templateData); err == nil {
		if _, err := r.SendEmail(); err != nil {
			log.Println("smtp error: " + err.Error())
		}
	} else {
		log.Println("group invite email template parse failed")
	}
}
