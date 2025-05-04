package email

import (
	"bytes"
	"fmt"
	"github.com/SarathLUN/go-email-phishing-tools/internal/config" // Adjust path
	"html/template"
	"log"
	"net/smtp"
	"strings"
)

// EmailTemplateData holds the data needed to populate the email template.
type EmailTemplateData struct {
	FullName     string
	TrackingLink string
	Subject      string // Include subject if it's dynamic or needs to be in template scope
}

// Sender defines the interface for sending emails.
type Sender interface {
	Send(toEmail, toName, subject string, templateData EmailTemplateData) error
}

// gmailSender implements the Sender interface using Gmail SMTP.
type gmailSender struct {
	cfg      *config.Config
	template *template.Template
}

// NewGmailSender creates a new sender instance, parsing the template on creation.
func NewGmailSender(cfg *config.Config) (Sender, error) {
	// Parse the template file
	log.Printf("Parsing email template from: %s", cfg.EmailTemplatePath)
	tmpl, err := template.ParseFiles(cfg.EmailTemplatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse email template file '%s': %w", cfg.EmailTemplatePath, err)
	}

	return &gmailSender{
		cfg:      cfg,
		template: tmpl,
	}, nil
}

// Send constructs and sends an email using the configured template and SMTP server.
func (s *gmailSender) Send(toEmail, toName, subject string, templateData EmailTemplateData) error {
	// Ensure template data has subject if needed by template itself
	templateData.Subject = subject

	// Execute the template
	var body bytes.Buffer
	if err := s.template.Execute(&body, templateData); err != nil {
		return fmt.Errorf("failed to execute email template for %s: %w", toEmail, err)
	}

	// Construct email headers and body
	// Use RFC 5322 standard format for headers
	headers := make(map[string]string)
	headers["From"] = s.cfg.SMTPSenderAddress
	//headers["From"] = "HR Department"
	headers["To"] = toEmail // Can use fmt.Sprintf("%s <%s>", toName, toEmail) if desired
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"
	headers["List-Unsubscribe"] = "<mailto:no-reply@passapptech.com?subject=unsubscribe>"

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body.String() // Separate headers from body with empty line

	// Setup SMTP authentication
	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPassword, s.cfg.SMTPHost)

	// SMTP server address
	smtpAddr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)

	// Send the email
	err := smtp.SendMail(smtpAddr, auth, s.cfg.SMTPSenderAddress, []string{toEmail}, []byte(message))
	if err != nil {
		// Log detailed error, but return a slightly simpler one
		log.Printf("SMTP Error for %s: %v", toEmail, err)
		// Check for common SMTP errors if needed (e.g., authentication failure)
		if strings.Contains(err.Error(), "Username and Password not accepted") {
			return fmt.Errorf("SMTP authentication failed for user %s", s.cfg.SMTPUser)
		}
		return fmt.Errorf("failed to send email via SMTP to %s", toEmail)
	}

	log.Printf("Successfully sent email to %s", toEmail)
	return nil
}
