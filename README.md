# Libredesk (mageaustralia Fork)

This is a maintained fork of [Libredesk](https://github.com/abhinavxd/libredesk), an open-source, self-hosted customer support desk.

We run Libredesk in production and love the project. This fork exists because we need to ship features at the pace our business demands, and the upstream PR review cycle doesn't always align with that timeline. Rather than pressure the maintainers, we maintain our own fork with the features we need.

We're not trying to replace or compete with upstream Libredesk — we actively track releases and rebase our changes onto new versions as they come out. If any of our additions are useful to the broader project, we're happy to contribute them back.

**Upstream**: [abhinavxd/libredesk](https://github.com/abhinavxd/libredesk) | [libredesk.io](https://libredesk.io) | [Live demo](https://demo.libredesk.io/)  
**Base version**: v1.0.1

---

## Fork Features

Everything from upstream Libredesk is included. The following are additions in this fork.

**Latest** — Bulk Actions with conversation selection, and inline agent/team assignment directly from the conversation list.

### Bulk Actions & Conversation Selection

Select multiple conversations from the list and perform bulk operations — no more opening each ticket individually to triage.

- **Per-row checkboxes** on the conversation list
- **Shift+click** range selection (click one, hold shift, click another to select all in between)
- **Select All** toggle in the bulk action toolbar
- **Bulk Assign** to any agent or team via dropdown
- **Bulk Status** change (Open, Replied, Resolved, Closed)
- **Bulk Priority** change (Urgent, High, Medium, Low, None)
- Toast notifications with success/error counts

### Quick-Assign Dropdowns on Conversation List

Each conversation row shows the assigned agent and team with inline dropdown menus for reassignment — no need to open the conversation.

- Agent assignment shown with user icon (orange "Unassigned" when empty)
- Team assignment shown with team icon
- Compact 2x2 grid layout alongside timestamp and unread badge
- Dropdown menus with full agent/team lists for quick reassignment

### OpenRouter AI Provider

Support for [OpenRouter](https://openrouter.ai/) as an AI provider, giving access to 100+ models (GPT-4o, Claude, Llama, Mistral, etc.) through a single API key.

### RAG AI Assistant Enhancements

Improvements to the built-in RAG AI assistant:

- **Knowledge Sources UI**: Admin page to manage knowledge sources (webpages, macros)
- **Context limiting**: Conversations trimmed to last 10 messages / 6000 chars to prevent timeouts on long threads
- **Ecommerce context injection**: Order and customer data included in AI prompts alongside knowledge base results
- **Extended timeouts**: AI provider HTTP timeouts increased to 60s for large prompts

### Ecommerce Integration (Maho Commerce)

Pull customer and order data from a Maho Commerce (Magento-compatible) store into AI-generated responses:

- Customer lookup by email
- Recent order fetching with items, prices, quantities
- Conversation scanning for order numbers with automatic detail retrieval
- Order status history and shipment tracking with carrier-specific URLs
- Supported carriers: Australia Post, Couriers Please, Team Global Express

### Freshdesk Theme

An alternative UI theme inspired by Freshdesk, selectable via a theme switcher in the sidebar.

- Teal colour palette with dark sidebar
- Conversation list hides when a ticket is open (full-width detail view)
- Sidebar collapsed by default
- Collapsible reply box with unified scrolling
- Theme persists via localStorage

### Conversation List Enhancements

- **Subject, ticket number, status, and priority** displayed on each row
- **Previous Conversations accordion** defaults to open
- **Conversation status and priority badges** with colour-coded indicators

### Email & Message Improvements

- **Inline image rendering** in conversation messages
- **Email HTML sanitisation** for incoming messages — cleaner rendering with tightened layout
- **Per-email remove buttons** on TO, CC, and BCC fields
- **Agent name in email From header** instead of generic inbox name

### Full-Width Layout Toggle

Toggle between split list/detail view and full-width conversation view. Messages render at full width for better readability on wide screens.

### Auto-Assign on Reply

Per-inbox setting that automatically assigns a conversation to the agent who replies, if it's currently unassigned.

### Per-Inbox Email Signatures

Each inbox can have its own email signature with dynamic placeholders, configured in inbox settings.

### Connection Testing

- **IMAP connection test** with debug logs in inbox settings
- **SMTP test** for email notification settings

### Multimodal AI (Image Support)

Conversation attachments (images) are extracted, resized to 500x500, and included as base64 in AI prompts for multimodal models that support vision.

### Other UI Customisations

- **Ticket ID in header**: Shows contact name, reference number, and subject (e.g., "John Smith #105 - Order enquiry")
- **Simplified sidebar name**: Contact name only in sidebar to avoid overflow
- **Self-assign notification suppression**: Assigning to yourself doesn't trigger a notification

---

## Installation

This fork is designed for self-hosting with local Docker builds. It is **not** published to Docker Hub.

### Docker (Recommended)

```shell
git clone https://github.com/mageaustralia/libredesk/.git
cd libredesk

cp config.sample.toml config.toml
# Edit config.toml as needed

docker compose up -d

# Set the System user password
docker exec -it libredesk_app ./libredesk --set-system-user-password
```

Go to `http://localhost:9000` and login with username `System` and the password you set.

### AI-Powered Responses (RAG)

The AI assistant uses PostgreSQL with pgvector for semantic search.

**Docker:** Already included — uses `pgvector/pgvector:pg17` image.

**Manual install:** Install the pgvector extension:
- Ubuntu/Debian: `apt install postgresql-17-pgvector`
- Or compile from [pgvector/pgvector](https://github.com/pgvector/pgvector)

The extension is automatically enabled during database migration.

---

## Keeping Up with Upstream

When a new upstream version is released:

```shell
git fetch origin --tags
git checkout -b feature/openrouter-vX.Y.Z vX.Y.Z
git cherry-pick <your-custom-commits>
# Resolve any conflicts, rebuild, deploy
```

---

## Contributing

For contributions to the core project, see [upstream Libredesk](https://github.com/abhinavxd/libredesk). For issues specific to this fork's features, open an issue on [mageaustralia/libredesk](https://github.com/mageaustralia/libredesk/).

The backend is written in Go and the frontend is Vue.js 3 with Shadcn for UI components. See [developer setup docs](https://docs.libredesk.io/contributing/developer-setup).
