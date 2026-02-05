package main

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetInboxSignature returns the processed signature for an inbox with placeholders replaced
func handleGetInboxSignature(r *fastglue.Request) error {
	app := r.Context.(*App)

	inboxID, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid inbox ID", nil, envelope.InputError)
	}

	conversationUUID := string(r.RequestCtx.QueryArgs().Peek("conversation_uuid"))

	// Get inbox from database (has Config field)
	inbox, err := app.inbox.GetDBRecord(inboxID)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Inbox not found", nil, envelope.NotFoundError)
	}

	// Parse inbox config
	var config struct {
		Signature string `json:"signature"`
	}
	if err := json.Unmarshal(inbox.Config, &config); err != nil {
		return r.SendEnvelope(map[string]string{"signature": ""})
	}

	if config.Signature == "" {
		return r.SendEnvelope(map[string]string{"signature": ""})
	}

	signature := config.Signature

	// Replace inbox placeholders
	signature = strings.ReplaceAll(signature, "{{inbox.name}}", inbox.Name)

	// Replace agent placeholders from auth context
	if user, ok := r.RequestCtx.UserValue("user").(*models.User); ok && user != nil {
		signature = strings.ReplaceAll(signature, "{{agent.first_name}}", user.FirstName)
		signature = strings.ReplaceAll(signature, "{{agent.last_name}}", user.LastName)
		signature = strings.ReplaceAll(signature, "{{agent.full_name}}", user.FirstName+" "+user.LastName)
		signature = strings.ReplaceAll(signature, "{{agent.email}}", user.Email.String)
	}

	// Replace customer placeholders if conversation UUID provided
	if conversationUUID != "" {
		conv, err := app.conversation.GetConversation(0, conversationUUID, "")
		if err == nil && conv.Contact.FirstName != "" {
			signature = strings.ReplaceAll(signature, "{{customer.first_name}}", conv.Contact.FirstName)
			signature = strings.ReplaceAll(signature, "{{customer.last_name}}", conv.Contact.LastName)
		}
	}

	return r.SendEnvelope(map[string]string{"signature": signature})
}
