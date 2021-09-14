package corpbankclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// Me returns the authenticated user details.
func (c *Client) Me(ctx context.Context) (*AuthUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.path("me"), nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	respData := &meResp{}
	if err := c.do(respData, req, http.StatusOK); err != nil {
		return nil, errors.WithStack(err)
	}

	return respData.Acc, nil
}

// AccountBalance returns the balance information for the given account ID.
func (c *Client) AccountBalance(ctx context.Context, accountID uuid.UUID) (*AccountBalance, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.path("accounts", accountID.String(), "balance"), nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	respData := &AccountBalance{}
	if err := c.do(respData, req, http.StatusOK); err != nil {
		return nil, errors.WithStack(err)
	}

	return respData, nil
}

// APIKeys returns the list of API keys.
func (c *Client) APIKeys(ctx context.Context, options ...RequestOption) (*PageInfo, []APIKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.path("api-keys"), nil)

	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	for _, opt := range options {
		if err := opt.apply(req); err != nil {
			return nil, nil, errors.WithStack(err)
		}
	}

	respData := &apiKeysResp{}
	if err := c.do(respData, req, http.StatusOK); err != nil {
		return nil, nil, errors.WithStack(err)
	}

	return &PageInfo{
		CurrentPage:  respData.Pagination.PageNum,
		TotalPages:   respData.Pagination.TotalPages,
		TotalRecords: respData.Pagination.TotalRecords,
	}, respData.APIKeys, nil
}

// NewAPIKey creates a new API key and returns with the secret key.
func (c *Client) NewAPIKey(ctx context.Context) (*APIKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.path("api-keys"), strings.NewReader(`{"permission": {}}`))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	req.Header.Set("Content-Type", "application/json")

	respData := &newAPIKeyResp{}
	if err := c.do(respData, req, http.StatusCreated); err != nil {
		return nil, errors.WithStack(err)
	}

	return respData.APIKey, nil
}

// DelAPIKey deletes the API key by the given ID.
func (c *Client) DelAPIKey(ctx context.Context, apiKeyID uuid.UUID) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.path("api-keys", apiKeyID.String()), nil)
	if err != nil {
		return errors.WithStack(err)
	}

	if err := c.do(nil, req, http.StatusNoContent); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// EnableAPIKey activate the API key by the given ID.
func (c *Client) EnableAPIKey(ctx context.Context, apiKeyID uuid.UUID) error {
	if err := c.setEnableAPIKey(ctx, apiKeyID, true); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// EnableAPIKey de-activate the API key by the given ID.
func (c *Client) DisableAPIKey(ctx context.Context, apiKeyID uuid.UUID) error {
	if err := c.setEnableAPIKey(ctx, apiKeyID, false); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (c *Client) setEnableAPIKey(ctx context.Context, apiKeyID uuid.UUID, enabled bool) error {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPut,
		c.path("api-keys", apiKeyID.String(), "enabled"),
		strings.NewReader(fmt.Sprintf(`{"enabled": %t}`, enabled)),
	)

	if err != nil {
		return errors.WithStack(err)
	}

	req.Header.Set("Content-Type", "application/json")

	if err := c.do(nil, req, http.StatusOK); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// Transactions returns the list of bank transactions. The list can be filtered by the given list of RequestOption.
func (c *Client) Transactions(ctx context.Context, options ...RequestOption) (*PageInfo, []Transaction, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.path("bank-transactions"), nil)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	for _, opt := range options {
		if err := opt.apply(req); err != nil {
			return nil, nil, errors.WithStack(err)
		}
	}

	respData := &transactionsResp{}
	if err := c.do(respData, req, http.StatusOK); err != nil {
		return nil, nil, errors.WithStack(err)
	}

	return &PageInfo{
		CurrentPage:  respData.PageNum,
		TotalPages:   respData.TotalPages,
		TotalRecords: respData.TotalRecords,
	}, respData.Transactions, nil
}

// MakePayment sends payment order to the bank and returns the bank response.
func (c *Client) MakePayment(ctx context.Context, paymentOrder PaymentOrder) (*PaymentResult, error) {
	reqBody, err := json.Marshal(&paymentReq{
		Src: paymentAddr{
			AddrType: "IBAN",
			Addr:     paymentOrder.SenderIBAN,
		},
		Dst: paymentDst{
			Addr: paymentAddr{
				AddrType: "IBAN",
				Addr:     paymentOrder.RecipientIBAN,
			},
			ID: paymentRecipientID{
				IDType: "NATIONAL_ID",
				ID:     paymentOrder.RecipientIdentityNum,
			},
			Name: paymentOrder.RecipientName,
		},
		Date:     "1970-01-01T00:00:00.000Z",
		Amount:   paymentOrder.TransferAmount.StringFixed(2),
		RefCode:  paymentOrder.RefCode,
		Desc:     paymentOrder.Description,
		Callback: "http://example.com",
	})

	if err != nil {
		return nil, errors.WithStack(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.path("payments"), bytes.NewReader(reqBody))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	req.Header.Set("Content-Type", "application/json")

	paymentResult := &PaymentResult{}

	if err := c.do(paymentResult, req, http.StatusAccepted); err != nil {
		return nil, errors.WithStack(err)
	}

	return paymentResult, nil
}
