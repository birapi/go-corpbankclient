Example to handle webhook notifications:
```go
package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/birapi/go-corpbankclient"
)

func main() {
	client, err := corpbankclient.NewClient(corpbankclient.Credentials{
		APIKeyID:     "<API_KEY_ID>",
		APIKeySecret: "<API_KEY_SECRET>",
	}, nil)

	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/bank-transfers", client.WebhookHandler(func(c context.Context, t corpbankclient.Transaction) error {
		switch t.Direction {
		case corpbankclient.TrxDirectionIncoming:
			log.Printf("Incoming bank transfer from %s: %s %s", t.Sender.Name, t.Amount.StringFixed(2), t.Currency)

		case corpbankclient.TrxDirectionOutgoing:
			log.Printf("Outgoing bank transfer to %s: %s %s", t.Recipient.Name, t.Amount.StringFixed(2), t.Currency)

		default:
			return errors.New("unknown direction")
		}

		return nil
	}))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

Example to make payment:
```go
package main

import (
	"context"
	"log"
	"errors"

	"github.com/birapi/go-corpbankclient"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func main() {
	client, err := corpbankclient.NewClient(corpbankclient.Credentials{
		APIKeyID:     "<API_KEY_ID>",
		APIKeySecret: "<API_KEY_SECRET>",
	}, nil)

	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	paymentResult, err := client.MakePayment(ctx, corpbankclient.PaymentOrder{
		SenderIBAN:           "<SENDER_BANK_ACCOUNT_IBAN>",
		RecipientIBAN:        "<RECIPIENT_BANK_ACCOUNT_IBAN>",
		RecipientName:        "<RECIPIENT_NAME>",
		RecipientIdentityNum: "<RECIPIENT_NATIONAL_ID_NUMBER>",
		TransferAmount:       decimal.NewFromInt(3), // transfer amount
		RefCode:              uuid.New().String(),   // a unique reference code
		Description:          "test",                // the description of the bank transfer
	})

	if err != nil {
		// the error type can be checked as follows
		if errors.Is(err, corpbankclient.ErrInsufficientBalance) {
			log.Fatal("insufficient balance")
		}

		log.Fatal(err)
	}

	log.Printf("Payment ID: %s", paymentResult.PaymentID)
}
```

Example to list of transactions:
```go
package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/birapi/go-corpbankclient"
)

func main() {
	client, err := corpbankclient.NewClient(corpbankclient.Credentials{
		APIKeyID:     "<API_KEY_ID>",
		APIKeySecret: "<API_KEY_SECRET>",
	}, nil)

	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

    // list the last 10 incoming transfers in the last 24 hours
	_, txList, err := client.Transactions(ctx,
		corpbankclient.WithPageSize(10),
		corpbankclient.WithFilterInDateRange(time.Now().Add(-24*time.Hour), time.Now()),
		corpbankclient.WithFilterIncomingTransactions(),
	)

	if err != nil {
		log.Fatal(err)
	}

	for _, t := range txList {
		switch t.Direction {
		case corpbankclient.TrxDirectionIncoming:
			log.Printf("Incoming bank transfer from %s at %s: %s %s", t.Sender.Name, t.Date.String(), t.Amount.StringFixed(2), t.Currency)

		case corpbankclient.TrxDirectionOutgoing:
			log.Printf("Outgoing bank transfer to %s at %s: %s %s", t.Recipient.Name, t.Date.String(), t.Amount.StringFixed(2), t.Currency)

		default:
			log.Fatal(errors.New("unknown direction"))
		}
	}
}
```

Examples for the rest of the functionality:
```go
package main

import (
	"context"
	"log"

	"github.com/birapi/go-corpbankclient"
	"github.com/google/uuid"
)

func main() {
	client, err := corpbankclient.NewClient(corpbankclient.Credentials{
		APIKeyID:     "<API_KEY_ID>",
		APIKeySecret: "<API_KEY_SECRET>",
	}, nil)

	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	user, err := client.Me(ctx)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Authenticated API user: %s", user.Email)

	balance, err := client.AccountBalance(ctx, uuid.MustParse("<BANK_ACCOUNT_ID>"))
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Account balance: %s", balance.Balance.StringFixed(2))

	_, apiKeys, err := client.APIKeys(ctx, corpbankclient.WithPageSize(100))
	if err != nil {
		log.Fatal(err)
	}

	for i, k := range apiKeys {
		log.Printf("API Key #%d: %s", i, k.ID)
	}

	// besides, the following functions can help to manage API keys
	// client.NewAPIKey()
	// client.EnableAPIKey()
	// client.DisableAPIKey()
	// client.DelAPIKey()
}
```

Example to make payment with a retry mechanism, using idempotency feature:
```go
package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/birapi/go-corpbankclient"
	"github.com/cenkalti/backoff"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func main() {
	client, err := corpbankclient.NewClient(corpbankclient.Credentials{
		APIKeyID:     "<API_KEY_ID>",
		APIKeySecret: "<API_KEY_SECRET>",
	}, nil)

	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	retry := backoff.NewExponentialBackOff()
	retry.InitialInterval = 200 * time.Millisecond
	retry.MaxInterval = 30 * time.Second
	retry.MaxElapsedTime = 3 * time.Minute

	var paymentResult *corpbankclient.PaymentResult

	// following errors can not be recovered
	permanentErrors := []error{
		corpbankclient.ErrInsufficientBalance,
		corpbankclient.ErrCurrencyMismatch,
		corpbankclient.ErrIncorrectRecipientData,
		corpbankclient.ErrInvalidRecipientID,
		corpbankclient.ErrOutOfEFTHours,
	}

	idempotencyKey, err := uuid.NewRandom()
	if err != nil {
		log.Fatal(err)
	}

	err = backoff.RetryNotify(
		func() error {
			paymentResult, err = client.MakePayment(ctx, corpbankclient.PaymentOrder{
				SenderIBAN:           "<SENDER_BANK_ACCOUNT_IBAN>",
				RecipientIBAN:        "<RECIPIENT_BANK_ACCOUNT_IBAN>",
				RecipientName:        "<RECIPIENT_NAME>",
				RecipientIdentityNum: "<RECIPIENT_NATIONAL_ID_NUMBER>",
				TransferAmount:       decimal.NewFromInt(3), // transfer amount
				RefCode:              uuid.New().String(),   // a unique reference code
				Description:          "test",                // the description of the bank transfer
				IdempotencyKey:       idempotencyKey.String(),
			})

			for _, e := range permanentErrors {
				if errors.Is(err, e) {
					return backoff.Permanent(err)
				}
			}

			return err
		},

		backoff.WithContext(retry, ctx),

		func(e error, d time.Duration) {
			log.Printf("payment error (will retry after %s): %+v", d.String(), err)
		})

	if err != nil {
		// the error type can be checked as follows
		if errors.Is(err, corpbankclient.ErrInsufficientBalance) {
			log.Fatal("insufficient balance")
		}

		log.Fatal(err)
	}

	log.Printf("Payment ID: %s", paymentResult.PaymentID)
}
```