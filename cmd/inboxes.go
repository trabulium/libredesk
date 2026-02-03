package main

import (
	"crypto/tls"
	"fmt"
	smtplib "net/smtp"
	"time"
	"encoding/json"
	"net/mail"
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/email/oauth"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetInboxes returns all inboxes
func handleGetInboxes(r *fastglue.Request) error {
	var app = r.Context.(*App)
	inboxes, err := app.inbox.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	for i := range inboxes {
		if err := inboxes[i].ClearPasswords(); err != nil {
			app.lo.Error("error clearing inbox passwords from response", "error", err)
			return envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.inbox}"), nil)
		}
	}
	return r.SendEnvelope(inboxes)
}

// handleGetInbox returns an inbox by ID
func handleGetInbox(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	inbox, err := app.inbox.GetDBRecord(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := inbox.ClearPasswords(); err != nil {
		app.lo.Error("error clearing inbox passwords from response", "error", err)
		return envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.inbox}"), nil)
	}
	return r.SendEnvelope(inbox)
}

// handleCreateInbox creates a new inbox
func handleCreateInbox(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		inbox = imodels.Inbox{}
	)
	if err := r.Decode(&inbox, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}

	// Trim whitespace from inbox fields and config.
	if err := trimInboxFields(&inbox); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "config"), err.Error(), envelope.InputError)
	}

	createdInbox, err := app.inbox.Create(inbox)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := validateInbox(app, createdInbox); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := reloadInboxes(app); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.couldNotReload", "name", "{globals.terms.inbox}"), nil, envelope.GeneralError)
	}

	// Clear passwords before returning.
	if err := createdInbox.ClearPasswords(); err != nil {
		app.lo.Error("error clearing inbox passwords from response", "error", err)
		return envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.inbox}"), nil)
	}

	return r.SendEnvelope(createdInbox)
}

// handleUpdateInbox updates an inbox
func handleUpdateInbox(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		inbox = imodels.Inbox{}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&inbox, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}

	// Trim whitespace from inbox fields and config.
	if err := trimInboxFields(&inbox); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "config"), err.Error(), envelope.InputError)
	}

	if err := validateInbox(app, inbox); err != nil {
		return sendErrorEnvelope(r, err)
	}

	updatedInbox, err := app.inbox.Update(id, inbox)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := reloadInboxes(app); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.couldNotReload", "name", "{globals.terms.inbox}"), nil, envelope.GeneralError)
	}

	// Clear passwords before returning.
	if err := updatedInbox.ClearPasswords(); err != nil {
		app.lo.Error("error clearing inbox passwords from response", "error", err)
		return envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.inbox}"), nil)
	}

	return r.SendEnvelope(updatedInbox)
}

// handleToggleInbox toggles an inbox
func handleToggleInbox(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	toggledInbox, err := app.inbox.Toggle(id)
	if err != nil {
		return err
	}

	if err := reloadInboxes(app); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.couldNotReload", "name", "{globals.terms.inbox}"), nil, envelope.GeneralError)
	}

	// Clear passwords before returning
	if err := toggledInbox.ClearPasswords(); err != nil {
		app.lo.Error("error clearing inbox passwords from response", "error", err)
		return envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.inbox}"), nil)
	}

	return r.SendEnvelope(toggledInbox)
}

// handleDeleteInbox deletes an inbox
func handleDeleteInbox(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	err := app.inbox.SoftDelete(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := reloadInboxes(app); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.couldNotReload", "name", "{globals.terms.inbox}"), nil, envelope.GeneralError)
	}
	return r.SendEnvelope(true)
}

// validateInbox validates the inbox
func validateInbox(app *App, inb imodels.Inbox) error {
	// Validate from address.
	if _, err := mail.ParseAddress(inb.From); err != nil {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalidFromAddress"), nil)
	}
	if len(inb.Config) == 0 {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "config"), nil)
	}
	if inb.Name == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "name"), nil)
	}
	if inb.Channel == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "channel"), nil)
	}

	// Validate email channel config.
	if inb.Channel == inbox.ChannelEmail {
		if err := validateEmailConfig(app, inb.Config); err != nil {
			return err
		}
	}
	return nil
}

// validateEmailConfig validates the email inbox configuration.
func validateEmailConfig(app *App, configJSON json.RawMessage) error {
	var cfg imodels.Config
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "config"), nil)
	}

	// Validate auth_type.
	if cfg.AuthType != "" && cfg.AuthType != imodels.AuthTypePassword && cfg.AuthType != imodels.AuthTypeOAuth2 {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "auth_type"), nil)
	}

	// Validate OAuth config if auth_type is oauth2.
	if cfg.AuthType == imodels.AuthTypeOAuth2 {
		if cfg.OAuth == nil {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "oauth"), nil)
		}
		if cfg.OAuth.Provider != string(oauth.ProviderGoogle) && cfg.OAuth.Provider != string(oauth.ProviderMicrosoft) {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "oauth.provider"), nil)
		}
		if cfg.OAuth.ClientID == "" {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "oauth.client_id"), nil)
		}
	}

	// Validate SMTP configs.
	for i, smtp := range cfg.SMTP {
		if smtp.Host == "" {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "smtp.host"), nil)
		}
		if smtp.Port <= 0 {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "smtp.port"), nil)
		}
		// Validate auth_protocol for password auth.
		if cfg.AuthType != imodels.AuthTypeOAuth2 {
			validAuthProtocols := map[string]bool{"": true, "none": true, "plain": true, "login": true, "cram": true}
			if !validAuthProtocols[cfg.SMTP[i].AuthProtocol] {
				return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "smtp.auth_protocol"), nil)
			}
		}
	}

	// Validate IMAP configs.
	for _, imap := range cfg.IMAP {
		if imap.Host == "" {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "imap.host"), nil)
		}
		if imap.Port <= 0 {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "imap.port"), nil)
		}
		if imap.Mailbox == "" {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "imap.mailbox"), nil)
		}
		// Validate tls_type.
		validTLSTypes := map[string]bool{"none": true, "starttls": true, "tls": true}
		if !validTLSTypes[imap.TLSType] {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "imap.tls_type"), nil)
		}
	}

	return nil
}

// trimInboxFields trims whitespace from inbox fields and its email config if applicable.
func trimInboxFields(inb *imodels.Inbox) error {
	inb.Name = strings.TrimSpace(inb.Name)
	inb.From = strings.TrimSpace(inb.From)

	// Trim email config fields if this is an email channel.
	if inb.Channel == inbox.ChannelEmail && len(inb.Config) > 0 {
		var cfg imodels.Config
		if err := json.Unmarshal(inb.Config, &cfg); err != nil {
			return err
		}
		trimEmailConfig(&cfg)
		trimmedConfig, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		inb.Config = trimmedConfig
	}
	return nil
}

// trimEmailConfig trims whitespace from email configuration fields.
// Passwords and secrets are intentionally NOT trimmed.
func trimEmailConfig(cfg *imodels.Config) {
	// Trim IMAP configs.
	for i := range cfg.IMAP {
		cfg.IMAP[i].Host = strings.TrimSpace(cfg.IMAP[i].Host)
		cfg.IMAP[i].Username = strings.TrimSpace(cfg.IMAP[i].Username)
		cfg.IMAP[i].Mailbox = strings.TrimSpace(cfg.IMAP[i].Mailbox)
	}

	// Trim SMTP configs.
	for i := range cfg.SMTP {
		cfg.SMTP[i].Host = strings.TrimSpace(cfg.SMTP[i].Host)
		cfg.SMTP[i].Username = strings.TrimSpace(cfg.SMTP[i].Username)
		cfg.SMTP[i].HelloHostname = strings.TrimSpace(cfg.SMTP[i].HelloHostname)
	}

	// Trim OAuth config.
	if cfg.OAuth != nil {
		cfg.OAuth.Provider = strings.TrimSpace(cfg.OAuth.Provider)
		cfg.OAuth.ClientID = strings.TrimSpace(cfg.OAuth.ClientID)
		cfg.OAuth.TenantID = strings.TrimSpace(cfg.OAuth.TenantID)
	}
}

// TestInboxRequest represents the request body for testing inbox connection.
type TestInboxRequest struct {
	IMAP      *imodels.IMAPConfig `json:"imap"`
	SMTP      *imodels.SMTPConfig `json:"smtp"`
	AuthType  string              `json:"auth_type"`
	TestEmail string              `json:"test_email"`
}

// TestInboxResponse represents the response for testing inbox connection.
type TestInboxResponse struct {
	Success  bool     `json:"success"`
	IMAPLogs []string `json:"imap_logs"`
	SMTPLogs []string `json:"smtp_logs"`
}

// handleTestInboxConnection tests IMAP and/or SMTP connection with the provided config.
func handleTestInboxConnection(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = TestInboxRequest{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	resp := TestInboxResponse{Success: true}

	// Test IMAP if config provided.
	if req.IMAP != nil && req.IMAP.Host != "" {
		imapLogs, imapOK := testIMAPConnection(req.IMAP)
		resp.IMAPLogs = imapLogs
		if !imapOK {
			resp.Success = false
		}
	}

	// Test SMTP if config provided.
	if req.SMTP != nil && req.SMTP.Host != "" {
		smtpLogs, smtpOK := testSMTPConnection(req.SMTP, req.TestEmail)
		resp.SMTPLogs = smtpLogs
		if !smtpOK {
			resp.Success = false
		}
	}

	return r.SendEnvelope(resp)
}

func testIMAPConnection(cfg *imodels.IMAPConfig) ([]string, bool) {
	logs := []string{}
	addLog := func(msg string) {
		logs = append(logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg))
	}

	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	addLog(fmt.Sprintf("Connecting to IMAP server: %s", address))

	imapOptions := &imapclient.Options{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: cfg.TLSSkipVerify,
		},
	}

	var client *imapclient.Client
	var err error

	switch cfg.TLSType {
	case "none":
		addLog("Using plain connection (no TLS)")
		client, err = imapclient.DialInsecure(address, imapOptions)
	case "starttls":
		addLog("Using STARTTLS connection")
		client, err = imapclient.DialStartTLS(address, imapOptions)
	case "tls":
		addLog("Using SSL/TLS connection")
		client, err = imapclient.DialTLS(address, imapOptions)
	default:
		addLog(fmt.Sprintf("Unknown TLS type: %s", cfg.TLSType))
		return logs, false
	}

	if err != nil {
		addLog(fmt.Sprintf("Connection failed: %v", err))
		return logs, false
	}
	defer client.Logout()
	addLog("Connected successfully")

	// Authenticate.
	addLog(fmt.Sprintf("Authenticating as: %s", cfg.Username))
	if err := client.Login(cfg.Username, cfg.Password).Wait(); err != nil {
		addLog(fmt.Sprintf("Authentication failed: %v", err))
		return logs, false
	}
	addLog("Authentication successful")

	// Select mailbox.
	addLog(fmt.Sprintf("Selecting mailbox: %s", cfg.Mailbox))
	mbox, err := client.Select(cfg.Mailbox, &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		addLog(fmt.Sprintf("Failed to select mailbox: %v", err))
		return logs, false
	}
	addLog(fmt.Sprintf("Mailbox selected - %d messages", mbox.NumMessages))
	addLog("IMAP test completed successfully!")

	return logs, true
}

func testSMTPConnection(cfg *imodels.SMTPConfig, testEmail string) ([]string, bool) {
	logs := []string{}
	addLog := func(msg string) {
		logs = append(logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg))
	}

	serverAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	addLog(fmt.Sprintf("Connecting to SMTP server: %s", serverAddr))

	tlsConfig := &tls.Config{
		ServerName:         cfg.Host,
		InsecureSkipVerify: cfg.TLSSkipVerify,
	}

	var client *smtplib.Client

	switch cfg.TLSType {
	case "tls":
		addLog("Using SSL/TLS connection")
		conn, err := tls.Dial("tcp", serverAddr, tlsConfig)
		if err != nil {
			addLog(fmt.Sprintf("TLS connection failed: %v", err))
			return logs, false
		}
		defer conn.Close()
		var cerr error
		client, cerr = smtplib.NewClient(conn, cfg.Host)
		if cerr != nil {
			addLog(fmt.Sprintf("Failed to create SMTP client: %v", cerr))
			return logs, false
		}
	default:
		addLog("Using plain connection")
		var err error
		client, err = smtplib.Dial(serverAddr)
		if err != nil {
			addLog(fmt.Sprintf("Connection failed: %v", err))
			return logs, false
		}
	}
	defer client.Close()
	addLog("Connected successfully")

	// Send EHLO.
	hostname := cfg.HelloHostname
	if hostname == "" {
		hostname = "localhost"
	}
	addLog(fmt.Sprintf("Sending EHLO %s", hostname))
	if err := client.Hello(hostname); err != nil {
		addLog(fmt.Sprintf("EHLO failed: %v", err))
		return logs, false
	}

	// STARTTLS if required.
	if cfg.TLSType == "starttls" {
		addLog("Starting TLS (STARTTLS)")
		if err := client.StartTLS(tlsConfig); err != nil {
			addLog(fmt.Sprintf("STARTTLS failed: %v", err))
			return logs, false
		}
		addLog("TLS connection established")
	}

	// Authenticate if credentials provided.
	if cfg.Username != "" && cfg.Password != "" {
		addLog(fmt.Sprintf("Authenticating as %s using %s", cfg.Username, cfg.AuthProtocol))
		var auth smtplib.Auth
		switch cfg.AuthProtocol {
		case "plain":
			auth = smtplib.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		case "login":
			auth = &loginAuth{username: cfg.Username, password: cfg.Password}
		case "cram":
			auth = smtplib.CRAMMD5Auth(cfg.Username, cfg.Password)
		case "none", "":
			addLog("No authentication required")
		}
		if auth != nil {
			if err := client.Auth(auth); err != nil {
				addLog(fmt.Sprintf("Authentication failed: %v", err))
				return logs, false
			}
			addLog("Authentication successful")
		}
	}

	// If test email provided, send a test message.
	if testEmail != "" {
		fromAddr := cfg.Username
		addLog(fmt.Sprintf("Sending test email to %s", testEmail))
		if err := client.Mail(fromAddr); err != nil {
			addLog(fmt.Sprintf("MAIL FROM failed: %v", err))
			return logs, false
		}
		if err := client.Rcpt(testEmail); err != nil {
			addLog(fmt.Sprintf("RCPT TO failed: %v", err))
			return logs, false
		}
		w, err := client.Data()
		if err != nil {
			addLog(fmt.Sprintf("DATA command failed: %v", err))
			return logs, false
		}
		msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: LibreDesk Inbox SMTP Test\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nThis is a test email from LibreDesk inbox SMTP configuration.\r\nSent at: %s",
			fromAddr, testEmail, time.Now().Format(time.RFC1123))
		if _, err := w.Write([]byte(msg)); err != nil {
			addLog(fmt.Sprintf("Failed to write message: %v", err))
			return logs, false
		}
		if err := w.Close(); err != nil {
			addLog(fmt.Sprintf("Failed to close message: %v", err))
			return logs, false
		}
		addLog("Test email sent successfully!")
	}

	addLog("SMTP test completed successfully!")
	client.Quit()
	return logs, true
}
