package corpbankclient

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AuthUserStatus string
type TrxDirection string
type TrxTransferMethod string

const (
	AuthUserStatusActive       AuthUserStatus = "ACTIVE"
	AuthUserStatusSuspended    AuthUserStatus = "SUSPENDED"
	AuthUserStatusPaused       AuthUserStatus = "PAUSED"
	AuthUserStatusNotActivated AuthUserStatus = "WAITING_FOR_ACTIVATION"

	TrxDirectionIncoming TrxDirection = "INCOMING"
	TrxDirectionOutgoing TrxDirection = "OUTGOING"

	TrxTransferMethodHavale TrxTransferMethod = "HAVALE"
	TrxTransferMethodEFT    TrxTransferMethod = "EFT"
	TrxTransferMethodFAST   TrxTransferMethod = "FAST"
)

type Credentials struct {
	APIKeyID     string
	APIKeySecret string
}

type AuthUser struct {
	Email     string         `json:"accountIdentifier"`
	FirstName string         `json:"firstName"`
	LastName  string         `json:"lastName"`
	Status    AuthUserStatus `json:"accountStatus"`
}

type meResp struct {
	Acc *AuthUser `json:"userAccount"`
}

type AccountBalance struct {
	Balance       decimal.Decimal `json:"balance"`
	LastUpdatedAt time.Time       `json:"lastUpdatedAt"`
}

type PageInfo struct {
	CurrentPage  int
	TotalPages   int
	TotalRecords int
}

type TransactionAccount struct {
	BankCode string `json:"bank_code"`
	IBAN     string `json:"iban"`
}

type TransactionParticipant struct {
	BankCode        string `json:"bank_code"`
	IBAN            string `json:"iban"`
	IdentitiyNumber string `json:"identity_number"`
	Name            string `json:"name"`
}

type Transaction struct {
	ID             uuid.UUID               `json:"id"`
	Date           time.Time               `json:"date"`
	Account        TransactionAccount      `json:"account"`
	Amount         decimal.Decimal         `json:"amount"`
	Currency       string                  `json:"currency"`
	Direction      TrxDirection            `json:"direction"`
	Description    string                  `json:"description"`
	ReceivedAt     time.Time               `json:"received_at"`
	RefCode        string                  `json:"reference_code"`
	TransferMethod TrxTransferMethod       `json:"transfer_type"`
	Sender         *TransactionParticipant `json:"sender"`
	Recipient      *TransactionParticipant `json:"recipient"`
}

type transactionsResp struct {
	PageNum      int           `json:"page_num"`
	TotalPages   int           `json:"total_pages"`
	TotalRecords int           `json:"total_records"`
	Transactions []Transaction `json:"transactions"`
}

type APIKey struct {
	ID         uuid.UUID  `json:"apiKeyID"`
	CreatedAt  time.Time  `json:"createdAt"`
	ModifiedAt *time.Time `json:"modifiedAt,omitempty"`
	Enabled    bool       `json:"enabled"`
	Secret     *string    `json:"apiKeySecret,omitempty"`
}

type apiKeysResp struct {
	Pagination struct {
		PageNum      int `json:"pageNum"`
		PageSize     int `json:"pageSize"`
		TotalPages   int `json:"totalPages"`
		TotalRecords int `json:"totalRecords"`
	} `json:"pagination"`
	APIKeys []APIKey `json:"apiKeys"`
}

type newAPIKeyResp struct {
	APIKey *APIKey `json:"apiKey"`
}

type PaymentOrder struct {
	IdempotencyKey       string
	SenderIBAN           string
	RecipientIBAN        string
	RecipientName        string
	RecipientIdentityNum string
	TransferAmount       decimal.Decimal
	RefCode              string
	Description          string
}

type PaymentResult struct {
	PaymentID uuid.UUID `json:"payment_id"`
}

type paymentAddr struct {
	AddrType string `json:"addressType"`
	Addr     string `json:"address"`
}

type paymentRecipientID struct {
	IDType string `json:"identifierType"`
	ID     string `json:"identifier"`
}

type paymentDst struct {
	Addr paymentAddr        `json:"address"`
	ID   paymentRecipientID `json:"identifier"`
	Name string             `json:"name"`
}

type paymentReq struct {
	Src      paymentAddr `json:"source"`
	Dst      paymentDst  `json:"destination"`
	Date     string      `json:"date"`
	Amount   string      `json:"amount"`
	RefCode  string      `json:"refNum"`
	Desc     string      `json:"description"`
	Callback string      `json:"callbackURL"`
}
