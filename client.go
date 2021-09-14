package corpbankclient

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type Client struct {
	keyID       uuid.UUID
	keySec      []byte
	baseURL     *url.URL
	hc          *http.Client
	maxTimeDiff time.Duration
}

type ClientOptions struct {
	APIBaseURL  string
	HTTPClient  *http.Client
	MaxTimeDiff time.Duration
}

const (
	maxReadBytes      = 10 * 1024 * 1024
	maxReadBytesOnErr = 4 * 1024

	defaultServiceURL  = "https://api.birapi.com/corpbank/aispis/v1"
	defaultMaxTimeDiff = 10 * time.Minute
)

func NewClient(apiCreds Credentials, clientOpts *ClientOptions) (*Client, error) {
	apiKeyID, err := uuid.Parse(apiCreds.APIKeyID)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse API key ID: `%s`", apiCreds.APIKeyID)
	}

	apiKeySec, err := base64.StdEncoding.DecodeString(apiCreds.APIKeySecret)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse API secret")
	}

	c := &Client{
		keyID:       apiKeyID,
		keySec:      apiKeySec,
		hc:          http.DefaultClient,
		maxTimeDiff: defaultMaxTimeDiff,
	}

	baseURL := defaultServiceURL
	if clientOpts != nil && clientOpts.APIBaseURL != "" {
		baseURL = clientOpts.APIBaseURL
	}

	c.baseURL, err = url.Parse(baseURL)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse API URL: `%s`", baseURL)
	}

	if clientOpts != nil && clientOpts.HTTPClient != nil {
		c.hc = clientOpts.HTTPClient
	}

	if clientOpts != nil && clientOpts.MaxTimeDiff > 0 {
		c.maxTimeDiff = clientOpts.MaxTimeDiff
	}

	return c, nil
}

func (c *Client) path(p ...string) string {
	u := *c.baseURL

	for _, p := range p {
		u.Path = path.Join(u.Path, url.QueryEscape(p))
	}

	return u.String()
}

func (c *Client) sign(req *http.Request) error {
	token := &BearerToken{
		APIKeyID:  c.keyID,
		Timestamp: time.Now(),
	}

	var reqBuf []byte

	if req.Body != nil && req.Body != http.NoBody {
		var err error

		reqBuf, err = io.ReadAll(req.Body)
		if err != nil {
			return errors.WithStack(err)
		}

		req.Body.Close()

		req.Body = io.NopCloser(bytes.NewBuffer(reqBuf))
	}

	if err := token.Sign(c.keySec, reqBuf); err != nil {
		return errors.WithStack(err)
	}

	packed, err := token.Pack()
	if err != nil {
		return errors.WithStack(err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", packed))

	return nil
}

func (c *Client) do(dst interface{}, req *http.Request, expectedStatusCode int) error {
	if err := c.sign(req); err != nil {
		return errors.WithStack(err)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != expectedStatusCode {
		respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxReadBytesOnErr))
		if err != nil {
			return errors.Wrapf(err, "unable to read HTTP response for status code: %s (expected: %d)", resp.Status, expectedStatusCode)
		}

		return errors.Errorf("remote service returns unexpected response: %s - %s", resp.Status, string(respBody))
	}

	if dst != nil {
		dec := json.NewDecoder(io.TeeReader(io.LimitReader(resp.Body, maxReadBytes), os.Stdout))

		if err := dec.Decode(dst); err != nil {
			return errors.Wrap(err, "unable to parse JSON response of the remote service")
		}
	}

	return nil
}
