package ecommerce

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/zerodha/logf"
)

// Manager handles ecommerce provider operations with multi-stage context gathering
type Manager struct {
	provider Provider
	lo       logf.Logger
}

// NewManager creates a new ecommerce manager
func NewManager(provider Provider, lo logf.Logger) *Manager {
	return &Manager{provider: provider, lo: lo}
}

// IsConfigured returns true if a provider is configured
func (m *Manager) IsConfigured() bool {
	return m.provider != nil
}

// GatherFullContext performs multi-stage context gathering for AI prompt
// Stage 1: Fetch customer + recent orders by email
// Stage 2: Scan all provided messages for order numbers
// Stage 3: Fetch full details for mentioned orders
func (m *Manager) GatherFullContext(ctx context.Context, email string, messages []string, maxOrders int) (*EcommerceContext, error) {
	if m.provider == nil {
		return nil, nil
	}

	result := &EcommerceContext{}

	// Stage 1: Fetch customer and recent orders
	customer, err := m.provider.GetCustomerByEmail(ctx, email)
	if err != nil && err != ErrNotFound {
		m.lo.Warn("failed to get customer", "email", email, "error", err)
	} else if err == nil {
		result.Customer = customer
	}

	orders, err := m.provider.GetOrdersByEmail(ctx, email, maxOrders)
	if err != nil && err != ErrNotFound {
		m.lo.Warn("failed to get orders", "email", email, "error", err)
	} else {
		result.RecentOrders = orders
	}

	// Stage 2: Scan ALL messages for order numbers
	var foundOrderNumbers []string
	for _, msg := range messages {
		nums := extractAllOrderNumbers(msg)
		foundOrderNumbers = append(foundOrderNumbers, nums...)
	}

	// Deduplicate
	seen := make(map[string]bool)
	var uniqueOrders []string
	for _, num := range foundOrderNumbers {
		if !seen[num] {
			seen[num] = true
			uniqueOrders = append(uniqueOrders, num)
		}
	}

	// Stage 3: Fetch full details for mentioned orders (limit to first 2)
	for i, orderNum := range uniqueOrders {
		if i >= 2 {
			break
		}
		order, err := m.provider.GetOrderByNumber(ctx, orderNum)
		if err == nil {
			result.MatchedOrders = append(result.MatchedOrders, order)
			m.lo.Debug("found order in conversation", "order_number", orderNum)
		} else if err != ErrNotFound {
			m.lo.Warn("failed to lookup order", "order_number", orderNum, "error", err)
		}
	}

	return result, nil
}

// FormatContextForPrompt formats ecommerce context as text for AI prompt
func (m *Manager) FormatContextForPrompt(eCtx *EcommerceContext) string {
	if eCtx == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n## Customer Ecommerce Data\n\n")

	if eCtx.Customer != nil {
		sb.WriteString(fmt.Sprintf("**Customer:** %s %s (%s)\n",
			eCtx.Customer.FirstName, eCtx.Customer.LastName, eCtx.Customer.Email))
		if eCtx.Customer.Telephone != "" {
			sb.WriteString(fmt.Sprintf("**Phone:** %s\n", eCtx.Customer.Telephone))
		}
		if !eCtx.Customer.CreatedAt.IsZero() {
			sb.WriteString(fmt.Sprintf("**Customer since:** %s\n", eCtx.Customer.CreatedAt.Format("2006-01-02")))
		}
	}

	// Show matched orders with FULL details
	if len(eCtx.MatchedOrders) > 0 {
		sb.WriteString("\n**Orders Mentioned in Conversation:**\n")
		for _, order := range eCtx.MatchedOrders {
			sb.WriteString(formatOrderFull(order))
			sb.WriteString("\n")
		}
	}

	// Show recent orders as summary only
	if len(eCtx.RecentOrders) > 0 {
		sb.WriteString("\n**Recent Orders (Summary):**\n")
		for _, order := range eCtx.RecentOrders {
			// Skip if already shown in matched orders
			alreadyShown := false
			for _, matched := range eCtx.MatchedOrders {
				if matched.IncrementID == order.IncrementID {
					alreadyShown = true
					break
				}
			}
			if !alreadyShown {
				sb.WriteString(formatOrderSummary(&order))
			}
		}
	}

	return sb.String()
}

func formatOrderFull(o *Order) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### Order #%s\n", o.IncrementID))
	sb.WriteString(fmt.Sprintf("- **Status:** %s\n", o.Status))
	sb.WriteString(fmt.Sprintf("- **Date:** %s\n", o.CreatedAt.Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("- **Total:** $%.2f\n", o.GrandTotal))

	if len(o.Items) > 0 {
		sb.WriteString("- **Items:**\n")
		for _, item := range o.Items {
			sb.WriteString(fmt.Sprintf("  - %s (SKU: %s) x%d @ $%.2f\n",
				item.Name, item.SKU, item.Qty, item.Price))
		}
	}

	if len(o.Shipments) > 0 {
		sb.WriteString("- **Shipments:**\n")
		for _, ship := range o.Shipments {
			sb.WriteString(fmt.Sprintf("  - **%s** Tracking: %s\n", ship.Carrier, ship.TrackingNumber))
			if ship.Status != "" {
				sb.WriteString(fmt.Sprintf("    Status: %s\n", ship.Status))
			}
		}
	}

	if o.ShippingAddress != nil {
		sb.WriteString(fmt.Sprintf("- **Shipping to:** %s %s, %s, %s %s\n",
			o.ShippingAddress.FirstName, o.ShippingAddress.LastName,
			o.ShippingAddress.City, o.ShippingAddress.Region, o.ShippingAddress.PostCode))
	}

	return sb.String()
}

func formatOrderSummary(o *Order) string {
	return fmt.Sprintf("- #%s | %s | $%.2f | %s\n",
		o.IncrementID, o.Status, o.GrandTotal, o.CreatedAt.Format("2006-01-02"))
}

// Order number patterns for Magento-style IDs (100xxxxxx)
var (
	orderPrefixRegex     = regexp.MustCompile(`(?i)(?:order|#|number)[:\s#]*(\d{9,12})`)
	standaloneOrderRegex = regexp.MustCompile(`\b(1\d{8,11})\b`)
)

func extractAllOrderNumbers(text string) []string {
	var results []string

	// First try prefixed patterns (higher confidence)
	matches := orderPrefixRegex.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) > 1 {
			results = append(results, match[1])
		}
	}

	// Then try standalone numbers
	matches = standaloneOrderRegex.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) > 1 {
			results = append(results, match[1])
		}
	}

	return results
}
