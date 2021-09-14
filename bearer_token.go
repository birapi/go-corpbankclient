package corpbankclient

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type tokenJSON struct {
	APIKeyID    string `json:"apiKeyID"`
	Timestamp   string `json:"timestamp"`
	SigningAlgo string `json:"algo"`
	Signature   string `json:"signature"`
}

type BearerToken struct {
	APIKeyID  uuid.UUID
	Timestamp time.Time
	Signature []byte
}

const (
	maxPackedLen = 1024
)

func (t *BearerToken) Sign(apiKeySecret, contentToSign []byte) error {
	t.Signature = t.sign(apiKeySecret, contentToSign)

	return nil
}

func (t *BearerToken) Verify(apiKeySecret, contentToSign []byte, maxClockSkew time.Duration) error {
	now := time.Now()

	calculatedSig := t.sign(apiKeySecret, contentToSign)

	if subtle.ConstantTimeCompare(t.Signature, calculatedSig) != 1 {
		return errors.New("illegal signature")
	}

	min := now.Add(-maxClockSkew)
	max := now.Add(maxClockSkew)

	if ts := t.Timestamp.UTC(); ts.Before(min) || ts.After(max) {
		return errors.Errorf("illegal timestamp")
	}

	return nil
}

func (t *BearerToken) sign(secret, contentToSign []byte) []byte {
	h := hmac.New(sha256.New, secret)

	h.Write([]byte(t.Timestamp.UTC().Format(time.RFC3339)))
	h.Write(contentToSign)

	return h.Sum(nil)
}

func (t *BearerToken) Pack() (string, error) {
	packed, err := json.Marshal(&tokenJSON{
		APIKeyID:    t.APIKeyID.String(),
		Timestamp:   t.Timestamp.Format(time.RFC3339),
		SigningAlgo: "HMAC-SHA256",
		Signature:   hex.EncodeToString(t.Signature),
	})

	if err != nil {
		return "", errors.Wrap(err, "unable to pack the bearer token")
	}

	return base64.URLEncoding.EncodeToString(packed), nil
}

func (t *BearerToken) Unpack(packed string) error {
	if l := len(packed); l > maxPackedLen {
		return errors.Errorf("bearer token string is too long: %d (allowed max: %d)", l, maxPackedLen)
	}

	tokenContent, err := base64.URLEncoding.DecodeString(packed)
	if err != nil {
		return errors.Wrapf(err, "unable to parse the bearer token: `%s`", packed)
	}

	token := &tokenJSON{}
	if err := json.NewDecoder(bytes.NewReader(tokenContent)).Decode(token); err != nil {
		return errors.Wrapf(err, "unable to parse the JSON content of the bearer token: `%s`", tokenContent)
	}

	apiKeyID, err := uuid.Parse(token.APIKeyID)
	if err != nil {
		return errors.Wrapf(err, "unable to parse the API key ID: `%s`", token.APIKeyID)
	}

	timestamp, err := time.Parse(time.RFC3339, token.Timestamp)
	if err != nil {
		return errors.Wrapf(err, "unable to parse the timestamp value: `%s`", token.Timestamp)
	}

	if strings.ToLower(strings.TrimSpace(token.SigningAlgo)) != "hmac-sha256" {
		return errors.Errorf("unsupported signing algorithm: `%s`", token.SigningAlgo)
	}

	sig, err := hex.DecodeString(token.Signature)
	if err != nil {
		return errors.Wrapf(err, "unable to parse the signature value: `%s`", token.Signature)
	}

	t.APIKeyID = apiKeyID
	t.Timestamp = timestamp
	t.Signature = sig

	return nil
}
