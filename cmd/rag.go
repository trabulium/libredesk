package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/ai"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/rag/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// SearchIntent represents a classified search intent from the AI.
type SearchIntent struct {
	Type  string `json:"type"`
	Query string `json:"query"`
}

// SearchClassification is the AI's classification response.
type SearchClassification struct {
	Intents []SearchIntent `json:"intents"`
}

// ExternalSearchHit represents a generic search result from an external search API.
type ExternalSearchHit struct {
	Name            string                 `json:"name"`
	Question        string                 `json:"question"`
	Answer          string                 `json:"answer"`
	URL             string                 `json:"url"`
	Description     string                 `json:"description"`
	MetaDescription string                 `json:"meta_description"`
	BrandID         string                 `json:"brand_id"`
	InStock         int                    `json:"in_stock"`
	ProductCount    int                    `json:"product_count"`
	Price           map[string]interface{} `json:"price"`
	Categories      map[string]interface{} `json:"categories"`
}

// ExternalSearchResponse is the response from an external search API.
type ExternalSearchResponse struct {
	Hits               []ExternalSearchHit `json:"hits"`
	Query              string              `json:"query"`
	EstimatedTotalHits int                 `json:"estimatedTotalHits"`
}

// handleGetRAGSources returns all RAG knowledge sources.
func handleGetRAGSources(r *fastglue.Request) error {
	app := r.Context.(*App)

	sources, err := app.rag.GetSources()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(sources)
}

// handleGetRAGSource returns a single RAG source by ID.
func handleGetRAGSource(r *fastglue.Request) error {
	app := r.Context.(*App)

	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid source ID", nil, envelope.InputError)
	}

	source, err := app.rag.GetSource(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(source)
}

// handleCreateRAGSource creates a new knowledge source.
func handleCreateRAGSource(r *fastglue.Request) error {
	app := r.Context.(*App)

	var req struct {
		Name       string          `json:"name"`
		SourceType string          `json:"source_type"`
		Config     json.RawMessage `json:"config"`
		Enabled    bool            `json:"enabled"`
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	// Validate
	if strings.TrimSpace(req.Name) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Name is required", nil, envelope.InputError)
	}
	if req.SourceType != "macro" && req.SourceType != "webpage" && req.SourceType != "custom" && req.SourceType != "file" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid source type", nil, envelope.InputError)
	}

	source, err := app.rag.CreateSource(req.Name, req.SourceType, req.Config, req.Enabled)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(source)
}

// handleUpdateRAGSource updates a knowledge source.
func handleUpdateRAGSource(r *fastglue.Request) error {
	app := r.Context.(*App)

	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid source ID", nil, envelope.InputError)
	}

	var req struct {
		Name    string          `json:"name"`
		Config  json.RawMessage `json:"config"`
		Enabled bool            `json:"enabled"`
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	source, err := app.rag.UpdateSource(id, req.Name, req.Config, req.Enabled)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(source)
}

// handleDeleteRAGSource deletes a knowledge source.
func handleDeleteRAGSource(r *fastglue.Request) error {
	app := r.Context.(*App)

	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid source ID", nil, envelope.InputError)
	}

	if err := app.rag.DeleteSource(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleSyncRAGSource triggers a sync for a source.
func handleSyncRAGSource(r *fastglue.Request) error {
	app := r.Context.(*App)

	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid source ID", nil, envelope.InputError)
	}

	// Sync in background
	go func() {
		if err := app.ragSync.SyncSourceByID(id); err != nil {
			app.lo.Error("error syncing source", "source_id", id, "error", err)
		}
	}()

	return r.SendEnvelope(map[string]string{"status": "sync_started"})
}

// handleRAGSearch searches the knowledge base.
func handleRAGSearch(r *fastglue.Request) error {
	app := r.Context.(*App)

	var req struct {
		Query     string  `json:"query"`
		Limit     int     `json:"limit"`
		Threshold float64 `json:"threshold"`
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	if strings.TrimSpace(req.Query) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Query is required", nil, envelope.InputError)
	}

	if req.Limit <= 0 {
		req.Limit = 5
	}
	if req.Threshold <= 0 {
		req.Threshold = 0.25
	}

	results, err := app.rag.Search(req.Query, req.Limit, req.Threshold)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(results)
}

// DefaultMaxRAGImages limits the number of images sent to multimodal AI.
const DefaultMaxRAGImages = 3

// handleRAGGenerateResponse generates a response using RAG.
func handleRAGGenerateResponse(r *fastglue.Request) error {
	app := r.Context.(*App)

	var req struct {
		ConversationID   int    `json:"conversation_id"`
		CustomerMessage  string `json:"customer_message"`
		IncludeEcommerce bool   `json:"include_ecommerce"`
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	if strings.TrimSpace(req.CustomerMessage) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Customer message is required", nil, envelope.InputError)
	}

	// Cap conversation context to avoid huge prompts and AI timeouts.
	const maxMessageLen = 6000
	if len(req.CustomerMessage) > maxMessageLen {
		req.CustomerMessage = "[Earlier messages truncated]\n\n" + req.CustomerMessage[len(req.CustomerMessage)-maxMessageLen:]
		app.lo.Info("truncated customer message", "original_len", len(req.CustomerMessage)+maxMessageLen, "truncated_to", len(req.CustomerMessage))
	}

	timerStart := time.Now()

	// Get AI settings
	aiSettings, err := app.setting.GetAISettings()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Use sensible defaults for RAG search
	threshold := aiSettings.SimilarityThreshold
	if threshold <= 0 || threshold > 0.5 {
		threshold = 0.25
	}
	maxChunks := aiSettings.MaxContextChunks
	if maxChunks <= 0 {
		maxChunks = 5
	}

	// Search knowledge base
	results, err := app.rag.Search(req.CustomerMessage, maxChunks, threshold)
	if err != nil {
		app.lo.Warn("RAG search failed, continuing without context", "error", err)
		results = []models.SearchResult{}
	}

	app.lo.Info("TIMING rag_search", "elapsed_ms", time.Since(timerStart).Milliseconds())

	app.lo.Info("RAG generate response", "query", req.CustomerMessage, "results_count", len(results), "threshold", threshold, "include_ecommerce", req.IncludeEcommerce)

	// Extract conversation images for multimodal AI
	var aiImages []ai.ImageContent
	if req.ConversationID > 0 {
		images, err := app.rag.GetConversationImages(req.ConversationID, DefaultMaxRAGImages)
		if err != nil {
			app.lo.Warn("failed to get conversation images, continuing without", "conversation_id", req.ConversationID, "error", err)
		} else if len(images) > 0 {
			for _, img := range images {
				aiImages = append(aiImages, ai.ImageContent{
					URL:      img.DataURL,
					Filename: img.Filename,
				})
			}
			app.lo.Info("conversation images extracted for AI", "conversation_id", req.ConversationID, "count", len(aiImages))
		}
	}

	// Gather ecommerce context if requested and configured
	var ecommerceContext string
	if req.IncludeEcommerce && req.ConversationID > 0 && app.ecommerce != nil && app.ecommerce.IsConfigured() {
		ecommerceContext = app.gatherEcommerceContext(r.RequestCtx, req.ConversationID)
	}

	app.lo.Info("TIMING ecommerce", "elapsed_ms", time.Since(timerStart).Milliseconds())

	// Search external search API if enabled.
	var externalSearchContext string
	if aiSettings.ExternalSearchEnabled && aiSettings.ExternalSearchURL != "" {
		maxSearchResults := aiSettings.ExternalSearchMaxResults
		if maxSearchResults <= 0 {
			maxSearchResults = 3
		}

		intents, err := app.classifySearchIntent(req.CustomerMessage)
		if err != nil {
			app.lo.Warn("External search classification failed, continuing without", "error", err)
		} else {
			app.lo.Info("External search classification", "intents", intents)
			externalSearchContext = app.performExternalSearch(intents, maxSearchResults)
			if externalSearchContext != "" {
				app.lo.Info("External search results added to context", "length", len(externalSearchContext))
			}
		}
	}

	app.lo.Info("TIMING external_search", "elapsed_ms", time.Since(timerStart).Milliseconds())

	// Build context from results
	var contextParts, macroParts []string
	for _, res := range results {
		if strings.HasPrefix(res.SourceRef, "macro_") {
			macroParts = append(macroParts, "- "+res.Title+": "+res.Content)
		} else {
			contextParts = append(contextParts, "## "+res.Title+"\n"+res.Content)
		}
	}

	contextStr := strings.Join(contextParts, "\n\n")
	macrosStr := strings.Join(macroParts, "\n")

	// Build system prompt from template - use default if not set
	systemPrompt := aiSettings.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = `You are a helpful customer support assistant for {{site_name}}. Use the following knowledge base content to answer questions accurately.

Knowledge Base Context:
{{context}}

Customer Question: {{enquiry}}

Provide a helpful, accurate response based on the context above. If the context does not contain relevant information, let the customer know you will need to check and get back to them.`
	}
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{site_name}}", ko.String("app.site_name"))
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{context}}", contextStr)
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{macros}}", macrosStr)
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{enquiry}}", req.CustomerMessage)
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{external_search_results}}", externalSearchContext)

	// Append ecommerce context if available
	if ecommerceContext != "" {
		systemPrompt += ecommerceContext
	}

	// Add note about images if present
	if len(aiImages) > 0 {
		systemPrompt += "\n\nNote: The customer has attached images to this conversation. Please examine them and reference relevant details in your response."
	}

	// Build prompt payload with optional images
	payload := ai.PromptPayload{
		SystemPrompt: systemPrompt,
		UserPrompt:   req.CustomerMessage,
		Images:       aiImages,
	}

	app.lo.Info("TIMING before_ai_completion", "elapsed_ms", time.Since(timerStart).Milliseconds(), "prompt_len", len(systemPrompt))

	// Generate response using the prompt payload with optional images
	response, err := app.ai.CompletionWithPayload(payload)
	app.lo.Info("TIMING ai_completion_done", "elapsed_ms", time.Since(timerStart).Milliseconds())

	if err != nil {
		app.lo.Error("TIMING ai_completion_failed", "elapsed_ms", time.Since(timerStart).Milliseconds(), "error", err)
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(map[string]interface{}{
		"response": response,
		"sources":  results,
	})
}

// gatherEcommerceContext retrieves ecommerce data for the conversation's contact
func (app *App) gatherEcommerceContext(ctx context.Context, conversationID int) string {
	// Get conversation UUID from ID
	uuid, err := app.conversation.GetConversationUUID(conversationID)
	if err != nil {
		app.lo.Warn("failed to get conversation UUID for ecommerce context", "conversation_id", conversationID, "error", err)
		return ""
	}

	// Get conversation with contact info
	conv, err := app.conversation.GetConversation(conversationID, uuid, "")
	if err != nil {
		app.lo.Warn("failed to get conversation for ecommerce context", "conversation_id", conversationID, "error", err)
		return ""
	}

	// Get customer email from contact
	customerEmail := ""
	if conv.Contact.Email.Valid {
		customerEmail = conv.Contact.Email.String
	}
	if customerEmail == "" {
		app.lo.Debug("no email for ecommerce context", "conversation_id", conversationID)
		return ""
	}

	// Get conversation messages for order number scanning
	messages, _, err := app.conversation.GetConversationMessages(uuid, 1, 50, nil, nil)
	if err != nil {
		app.lo.Warn("failed to get messages for ecommerce context", "conversation_id", conversationID, "error", err)
		// Continue without message scanning
	}

	// Extract text content from messages
	var messageTexts []string
	for _, msg := range messages {
		if msg.Content != "" && msg.Content != "" {
			// Strip HTML tags for order number scanning
			text := stripHTML(msg.Content)
			if text != "" {
				messageTexts = append(messageTexts, text)
			}
		}
	}

	app.lo.Info("ecommerce message scan", "db_messages", len(messages), "text_messages", len(messageTexts))

	// Gather ecommerce context using the manager
	eCtx, err := app.ecommerce.GatherFullContext(ctx, customerEmail, messageTexts, 5)
	if err != nil {
		app.lo.Warn("failed to gather ecommerce context", "email", customerEmail, "error", err)
		return ""
	}

	// Format for AI prompt
	formatted := app.ecommerce.FormatContextForPrompt(eCtx)
	if formatted != "" {
		app.lo.Info("ecommerce context added to prompt", "email", customerEmail, "length", len(formatted))
	}

	return formatted
}

// classifySearchIntent uses the AI to classify a customer message into search intents.
func (app *App) classifySearchIntent(message string) ([]SearchIntent, error) {
	classifyPrompt := `Analyze this customer support message and extract search intents.
Return JSON only, no other text.

Message: "` + message + `"

Response format:
{"intents": [{"type": "product", "query": "concise search terms"}, {"type": "category", "query": "concise terms"}, {"type": "faq", "query": "concise terms"}]}

Rules:
- Only include intent types that are relevant to the message
- Keep queries to 2-5 words, suitable for search
- "product" = looking for a specific product, brand, or item
- "category" = browsing a type/category of items
- "faq" = asking about policies, shipping, returns, orders, delivery, etc.
- A message can have multiple intents
- If the message is purely conversational (greetings, thanks) or not related to products/policies, return empty intents: {"intents": []}
- Do NOT wrap in markdown code blocks`

	response, err := app.ai.CompletionWithSystemPrompt("You are a JSON-only classifier. Output valid JSON only, no markdown, no explanation.", classifyPrompt)
	if err != nil {
		return nil, fmt.Errorf("classification failed: %w", err)
	}

	// Clean up response - remove markdown code blocks if present.
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var classification SearchClassification
	if err := json.Unmarshal([]byte(response), &classification); err != nil {
		return nil, fmt.Errorf("failed to parse classification: %w (response: %s)", err, response)
	}

	return classification.Intents, nil
}

// queryExternalSearch queries an external search API endpoint via HTTP POST.
func queryExternalSearch(searchURL, query string, limit int, headers map[string]string) (*ExternalSearchResponse, error) {
	payload := fmt.Sprintf(`{"q":%q,"limit":%d}`, query, limit)
	req, err := http.NewRequest("POST", searchURL, bytes.NewBufferString(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Apply custom headers.
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Default User-Agent if not set.
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "LibreDesk/1.0")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("external search returned status %d: %s", resp.StatusCode, string(body))
	}

	var result ExternalSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// performExternalSearch searches configured external endpoints based on classified intents.
func (app *App) performExternalSearch(intents []SearchIntent, maxResults int) string {
	var sections []string

	aiSettings, err := app.setting.GetAISettings()
	if err != nil {
		app.lo.Warn("Failed to get AI settings for external search", "error", err)
		return ""
	}

	// Parse configured endpoints from settings.
	// Format: JSON object mapping intent type to URL path suffix.
	// e.g. {"product": "/indexes/my_products/search", "category": "/indexes/my_categories/search", "faq": "/indexes/my_faqs/search"}
	endpoints := make(map[string]string)
	if aiSettings.ExternalSearchEndpoints != "" {
		if err := json.Unmarshal([]byte(aiSettings.ExternalSearchEndpoints), &endpoints); err != nil {
			app.lo.Warn("Failed to parse external search endpoints config", "error", err)
			return ""
		}
	}

	if len(endpoints) == 0 {
		app.lo.Warn("No external search endpoints configured")
		return ""
	}

	// Parse custom headers.
	// Format: JSON object of header key-value pairs.
	// e.g. {"User-Agent": "Mozilla/5.0...", "Referer": "https://example.com/"}
	headers := make(map[string]string)
	if aiSettings.ExternalSearchHeaders != "" {
		if err := json.Unmarshal([]byte(aiSettings.ExternalSearchHeaders), &headers); err != nil {
			app.lo.Warn("Failed to parse external search headers config", "error", err)
		}
	}

	baseURL := strings.TrimRight(aiSettings.ExternalSearchURL, "/")

	for _, intent := range intents {
		endpointPath, ok := endpoints[intent.Type]
		if !ok {
			continue
		}

		searchURL := baseURL + endpointPath
		results, err := queryExternalSearch(searchURL, intent.Query, maxResults, headers)
		if err != nil {
			app.lo.Warn("External search query failed", "type", intent.Type, "query", intent.Query, "error", err)
			continue
		}

		if len(results.Hits) == 0 {
			continue
		}

		switch intent.Type {
		case "product":
			var lines []string
			for i, hit := range results.Hits {
				price := ""
				if aud, ok := hit.Price["AUD"]; ok {
					if audMap, ok := aud.(map[string]interface{}); ok {
						if formatted, ok := audMap["default_formated"].(string); ok {
							price = formatted
						}
						if origFormatted, ok := audMap["default_original_formated"].(string); ok {
							price += " (was " + origFormatted + ")"
						}
					}
				}
				stock := "In Stock"
				if hit.InStock == 0 {
					stock = "Out of Stock"
				}
				line := fmt.Sprintf("%d. %s", i+1, hit.Name)
				if hit.BrandID != "" {
					line += " by " + hit.BrandID
				}
				if price != "" {
					line += " - " + price
				}
				line += " - " + stock
				line += "\n   URL: " + hit.URL
				desc := stripHTML(hit.Description)
				if len(desc) > 200 {
					desc = desc[:200] + "..."
				}
				if desc != "" {
					line += "\n   " + strings.TrimSpace(desc)
				}
				lines = append(lines, line)
			}
			sections = append(sections, "=== Product Results (from website) ===\n"+strings.Join(lines, "\n\n"))

		case "category":
			var lines []string
			for i, hit := range results.Hits {
				line := fmt.Sprintf("%d. %s (%d products)", i+1, hit.Name, hit.ProductCount)
				line += "\n   URL: " + hit.URL
				if hit.MetaDescription != "" {
					desc := hit.MetaDescription
					if len(desc) > 200 {
						desc = desc[:200] + "..."
					}
					line += "\n   " + desc
				}
				lines = append(lines, line)
			}
			sections = append(sections, "=== Category Results (from website) ===\n"+strings.Join(lines, "\n\n"))

		case "faq":
			var lines []string
			for i, hit := range results.Hits {
				line := fmt.Sprintf("%d. Q: %s\n   A: %s", i+1, hit.Question, hit.Answer)
				line += "\n   URL: " + hit.URL
				lines = append(lines, line)
			}
			sections = append(sections, "=== FAQ Results (from website) ===\n"+strings.Join(lines, "\n\n"))
		}
	}

	if len(sections) == 0 {
		return ""
	}

	return strings.Join(sections, "\n\n")
}

// stripHTML removes HTML tags from a string.
func stripHTML(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	for strings.Contains(s, "<") {
		start := strings.Index(s, "<")
		end := strings.Index(s[start:], ">")
		if end == -1 {
			break
		}
		s = s[:start] + " " + s[start+end+1:]
	}
	// Collapse multiple spaces
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}

// handleRAGFileUpload handles file uploads for RAG knowledge sources.
func handleRAGFileUpload(r *fastglue.Request) error {
	app := r.Context.(*App)

	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		app.lo.Error("error parsing form data", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid form data", nil, envelope.InputError)
	}

	// Get file
	files, ok := form.File["file"]
	if !ok || len(files) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "No file provided", nil, envelope.InputError)
	}

	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		app.lo.Error("error opening file", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read file", nil, envelope.GeneralError)
	}
	defer file.Close()

	// Read file content
	content := make([]byte, fileHeader.Size)
	if _, err := file.Read(content); err != nil {
		app.lo.Error("error reading file", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read file", nil, envelope.GeneralError)
	}

	// Get file extension and determine type
	filename := fileHeader.Filename
	var fileType string
	if strings.HasSuffix(strings.ToLower(filename), ".txt") {
		fileType = "txt"
	} else if strings.HasSuffix(strings.ToLower(filename), ".csv") {
		fileType = "csv"
	} else if strings.HasSuffix(strings.ToLower(filename), ".json") {
		fileType = "json"
	} else {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Unsupported file type. Only .txt, .csv, and .json files are supported", nil, envelope.InputError)
	}

	// Get name from form or use filename
	var name string
	if names, ok := form.Value["name"]; ok && len(names) > 0 && strings.TrimSpace(names[0]) != "" {
		name = strings.TrimSpace(names[0])
	} else {
		name = filename
	}

	// Check if enabled
	enabled := true
	if enabledVals, ok := form.Value["enabled"]; ok && len(enabledVals) > 0 {
		enabled = enabledVals[0] == "true"
	}

	// Create file config
	config := models.FileConfig{
		Filename: filename,
		FileType: fileType,
		Content:  string(content),
	}
	configJSON, err := json.Marshal(config)
	if err != nil {
		app.lo.Error("error marshaling config", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to process file", nil, envelope.GeneralError)
	}

	// Create the source
	source, err := app.rag.CreateSource(name, "file", configJSON, enabled)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Sync immediately in background
	go func() {
		if err := app.ragSync.SyncSourceByID(source.ID); err != nil {
			app.lo.Error("error syncing file source", "source_id", source.ID, "error", err)
		}
	}()

	return r.SendEnvelope(source)
}
