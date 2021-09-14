package corpbankclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type WebhookHandler func(context.Context, Transaction) error

func (c *Client) WebhookHandler(handler WebhookHandler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		token := &BearerToken{}

		hdrs := r.Header.Values("Authorization")
		if l := len(hdrs); l == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Missing `Authorization` header."))
			return

		} else if l > 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Multiple `Authorization` header."))
			return

		} else if hdr := strings.TrimSpace(hdrs[0]); len(hdr) < 7 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Incomplete `Authorization` header."))
			return

		} else if v := strings.ToLower(hdr[:7]); v != "bearer " {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid `Authorization` token type."))
			return

		} else if v := strings.TrimSpace(hdr[7:]); len(v) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Missing bearer token."))
			return

		} else if err := token.Unpack(v); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Invalid bearer token: %s", err.Error())))
			return
		}

		payload, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Unable to read request body: %s", err.Error())))
			return
		}

		if err := token.Verify(c.keySec, payload, c.maxTimeDiff); err != nil {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(fmt.Sprintf("Unable to verify the request signature: %s", err.Error())))
			return
		}

		if !bytes.Equal(token.APIKeyID[:], c.keyID[:]) {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(fmt.Sprintf("Illegal signer: %s", err.Error())))
			return
		}

		trx := &Transaction{}
		if err := json.Unmarshal(payload, trx); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Invalid request payload: %s", err.Error())))
			return
		}

		if err := handler(r.Context(), *trx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("An error occurred while processing the webhook notification: %s", err.Error())))
			return
		}

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("The webhook notification has been processed successfully."))
	}
}
