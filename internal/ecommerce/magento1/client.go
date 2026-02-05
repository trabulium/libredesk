package magento1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/abhinavxd/libredesk/internal/ecommerce"
)

// Client implements the ecommerce.Provider interface for Magento 1 / Maho Commerce
type Client struct {
	baseURL string
	auth    *authClient
	http    *http.Client
}

// New creates a new Magento 1 client from the given configuration
func New(config ecommerce.ProviderConfig) (*Client, error) {
	if config.BaseURL == "" || config.ClientID == "" || config.ClientSecret == "" {
		return nil, fmt.Errorf("magento1: baseURL, clientID, and clientSecret are required")
	}
	return &Client{
		baseURL: config.BaseURL,
		auth:    newAuthClient(config.BaseURL, config.ClientID, config.ClientSecret),
		http:    &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Name returns the provider identifier
func (c *Client) Name() string { return "magento1" }

// doRequest makes an authenticated request to the Magento API
func (c *Client) doRequest(ctx context.Context, endpoint string, params url.Values) (*http.Response, error) {
	token, err := c.auth.getToken()
	if err != nil {
		return nil, err
	}

	u := c.baseURL + endpoint
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	return c.http.Do(req)
}

// GetCustomerByEmail looks up a customer by email address
func (c *Client) GetCustomerByEmail(ctx context.Context, email string) (*ecommerce.Customer, error) {
	resp, err := c.doRequest(ctx, "/api/v2/customers", url.Values{"email": {email}})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d", resp.StatusCode)
	}

	var customers []magentoCustomer
	if err := json.NewDecoder(resp.Body).Decode(&customers); err != nil {
		return nil, err
	}
	if len(customers) == 0 {
		return nil, ecommerce.ErrNotFound
	}
	return customers[0].toEcommerce(), nil
}

// GetOrdersByEmail returns recent orders for an email address
func (c *Client) GetOrdersByEmail(ctx context.Context, email string, limit int) ([]ecommerce.Order, error) {
	params := url.Values{"email": {email}, "pageSize": {fmt.Sprintf("%d", limit)}}
	resp, err := c.doRequest(ctx, "/api/v2/orders", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d", resp.StatusCode)
	}

	var orders []magentoOrder
	if err := json.NewDecoder(resp.Body).Decode(&orders); err != nil {
		return nil, err
	}

	result := make([]ecommerce.Order, len(orders))
	for i, o := range orders {
		result[i] = o.toEcommerce()
	}
	return result, nil
}

// GetOrderByNumber looks up an order by its display number (increment_id)
func (c *Client) GetOrderByNumber(ctx context.Context, orderNumber string) (*ecommerce.Order, error) {
	resp, err := c.doRequest(ctx, "/api/v2/orders", url.Values{"incrementId": {orderNumber}})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d", resp.StatusCode)
	}

	var orders []magentoOrder
	if err := json.NewDecoder(resp.Body).Decode(&orders); err != nil {
		return nil, err
	}
	if len(orders) == 0 {
		return nil, ecommerce.ErrNotFound
	}
	order := orders[0].toEcommerce()
	return &order, nil
}

// GetOrderByID looks up an order by internal ID
func (c *Client) GetOrderByID(ctx context.Context, orderID string) (*ecommerce.Order, error) {
	resp, err := c.doRequest(ctx, "/api/v2/orders/"+orderID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ecommerce.ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d", resp.StatusCode)
	}

	var order magentoOrder
	if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
		return nil, err
	}
	result := order.toEcommerce()
	return &result, nil
}

// TestConnection verifies the provider configuration is valid
func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.auth.getToken()
	return err
}

// Magento API response types

type magentoCustomer struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Telephone string `json:"telephone"`
	CreatedAt string `json:"createdAt"`
}

func (m *magentoCustomer) toEcommerce() *ecommerce.Customer {
	created, _ := time.Parse("2006-01-02 15:04:05", m.CreatedAt)
	return &ecommerce.Customer{
		ID:        fmt.Sprintf("%d", m.ID),
		Email:     m.Email,
		FirstName: m.Firstname,
		LastName:  m.Lastname,
		Telephone: m.Telephone,
		CreatedAt: created,
	}
}

type magentoOrder struct {
	ID                int                `json:"id"`
	IncrementID       string             `json:"incrementId"`
	CustomerEmail     string             `json:"customerEmail"`
	CustomerFirstname string             `json:"customerFirstname"`
	CustomerLastname  string             `json:"customerLastname"`
	Status            string             `json:"status"`
	State             string             `json:"state"`
	Items             []magentoOrderItem `json:"items"`
	Prices            magentoOrderPrices `json:"prices"`
	ShippingAddress   *magentoAddress    `json:"shippingAddress"`
	BillingAddress    *magentoAddress    `json:"billingAddress"`
	CreatedAt         string             `json:"createdAt"`
}

type magentoOrderItem struct {
	SKU        string  `json:"sku"`
	Name       string  `json:"name"`
	QtyOrdered int     `json:"qtyOrdered"`
	Price      float64 `json:"price"`
	RowTotal   float64 `json:"rowTotal"`
}

type magentoOrderPrices struct {
	Subtotal       float64 `json:"subtotal"`
	GrandTotal     float64 `json:"grandTotal"`
	ShippingAmount float64 `json:"shippingAmount"`
}

type magentoAddress struct {
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Street    string `json:"street"`
	City      string `json:"city"`
	Region    string `json:"region"`
	Postcode  string `json:"postcode"`
	CountryID string `json:"countryId"`
	Telephone string `json:"telephone"`
}

func (m *magentoOrder) toEcommerce() ecommerce.Order {
	created, _ := time.Parse("2006-01-02 15:04:05", m.CreatedAt)
	items := make([]ecommerce.OrderItem, len(m.Items))
	for i, item := range m.Items {
		items[i] = ecommerce.OrderItem{
			SKU:      item.SKU,
			Name:     item.Name,
			Qty:      item.QtyOrdered,
			Price:    item.Price,
			RowTotal: item.RowTotal,
		}
	}
	order := ecommerce.Order{
		ID:             fmt.Sprintf("%d", m.ID),
		IncrementID:    m.IncrementID,
		CustomerEmail:  m.CustomerEmail,
		CustomerName:   m.CustomerFirstname + " " + m.CustomerLastname,
		Status:         m.Status,
		State:          m.State,
		Items:          items,
		Subtotal:       m.Prices.Subtotal,
		GrandTotal:     m.Prices.GrandTotal,
		ShippingAmount: m.Prices.ShippingAmount,
		Currency:       "AUD",
		CreatedAt:      created,
	}
	if m.ShippingAddress != nil {
		order.ShippingAddress = &ecommerce.Address{
			FirstName: m.ShippingAddress.Firstname,
			LastName:  m.ShippingAddress.Lastname,
			Street:    m.ShippingAddress.Street,
			City:      m.ShippingAddress.City,
			Region:    m.ShippingAddress.Region,
			PostCode:  m.ShippingAddress.Postcode,
			Country:   m.ShippingAddress.CountryID,
			Telephone: m.ShippingAddress.Telephone,
		}
	}
	if m.BillingAddress != nil {
		order.BillingAddress = &ecommerce.Address{
			FirstName: m.BillingAddress.Firstname,
			LastName:  m.BillingAddress.Lastname,
			Street:    m.BillingAddress.Street,
			City:      m.BillingAddress.City,
			Region:    m.BillingAddress.Region,
			PostCode:  m.BillingAddress.Postcode,
			Country:   m.BillingAddress.CountryID,
			Telephone: m.BillingAddress.Telephone,
		}
	}
	return order
}
