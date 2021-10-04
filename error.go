package corpbankclient

import (
	"encoding/json"
	"errors"
	"fmt"
)

type errUnexpectedStatus struct {
	StatusCode int
	RespBody   []byte
}

type APIErr struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

var ErrCurrencyMismatch = errors.New("payment error: currency mismatch")
var ErrIncorrectRecipientData = errors.New("payment error: incorrect recipient data")
var ErrInsufficientBalance = errors.New("payment error: insufficient balance")
var ErrInvalidRecipientID = errors.New("payment error: recipient id")
var ErrOutOfEFTHours = errors.New("payment error: out of eft hours")

func wrapErr(err error) error {
	e := &errUnexpectedStatus{}

	if !errors.As(err, &e) {
		return err
	}

	aErr := &APIErr{}
	if err := json.Unmarshal(e.RespBody, &aErr); err != nil {
		return err
	}

	switch aErr.Code {
	case "CURRENCY_MISMATCH":
		return ErrCurrencyMismatch

	case "INCORRECT_RECIPIENT_DATA":
		return ErrIncorrectRecipientData

	case "INSUFFICIENT_BALANCE":
		return ErrInsufficientBalance

	case "INVALID_RECIPIENT_ID":
		return ErrInvalidRecipientID

	case "OUT_OF_EFT_HOURS":
		return ErrOutOfEFTHours
	}

	return aErr
}

func (e *errUnexpectedStatus) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, string(e.RespBody))
}

func (e *APIErr) Error() string {
	return fmt.Sprintf("APIErr %s: %s", e.Code, e.Message)
}
