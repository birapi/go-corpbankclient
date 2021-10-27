package corpbankclient

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type RequestOption interface {
	apply(req *http.Request) error
}

type reqOpt func(req *http.Request) error

func (o reqOpt) apply(req *http.Request) error {
	return o(req)
}

// WithPageNum allows customizing the requested page number.
func WithPageNum(pageNum int) RequestOption {
	return reqOpt(func(req *http.Request) error {
		q := req.URL.Query()
		q.Set("pageNum", strconv.FormatInt(int64(pageNum), 10))
		req.URL.RawQuery = q.Encode()
		return nil
	})
}

// WithPageSize allows customizing the requested page size.
func WithPageSize(pageSize int) RequestOption {
	return reqOpt(func(req *http.Request) error {
		q := req.URL.Query()
		q.Set("pageSize", strconv.FormatInt(int64(pageSize), 10))
		req.URL.RawQuery = q.Encode()
		return nil
	})
}

// WithPageSize filters the list of bank transactions by the given date range
func WithFilterInDateRange(startDate, endDate time.Time) RequestOption {
	return reqOpt(func(req *http.Request) error {
		q := req.URL.Query()
		q.Set("startDate", startDate.Format(time.RFC3339))
		q.Set("endDate", endDate.Format(time.RFC3339))
		req.URL.RawQuery = q.Encode()
		return nil
	})
}

// WithFilterIncomingTransactions filters the list of bank transactions for only incoming transfers.
func WithFilterIncomingTransactions() RequestOption {
	return reqOpt(func(req *http.Request) error {
		q := req.URL.Query()
		q.Set("direction", "INCOMING")
		req.URL.RawQuery = q.Encode()
		return nil
	})
}

// WithFilterOutgoingTransactions filters the list of bank transactions for only outgoing transfers.
func WithFilterOutgoingTransactions() RequestOption {
	return reqOpt(func(req *http.Request) error {
		q := req.URL.Query()
		q.Set("direction", "OUTGOING")
		req.URL.RawQuery = q.Encode()
		return nil
	})
}

// WithFilterAccountIDs filters the list of bank transactions for the given list of account ID.
func WithFilterAccountIDs(accountID ...uuid.UUID) RequestOption {
	return reqOpt(func(req *http.Request) error {
		q := req.URL.Query()
		for _, aid := range accountID {
			q.Add("account", aid.String())
		}
		req.URL.RawQuery = q.Encode()
		return nil
	})
}
