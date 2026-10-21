package common

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type PaginationRequest struct {
	Page     int `json:"page,omitempty"`
	PageSize int `json:"pagesize,omitempty"`
}

type Pagination struct {
	Page      int `json:"page,omitempty"`
	PageSize  int `json:"pagesize,omitempty"`
	Total     int `json:"total,omitempty"`
	TotalPage int `json:"total_page,omitempty"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type OrderResponse struct {
	OrderID    string   `json:"order_id"`
	ProductIDs []string `json:"product_ids,omitempty"`
}

type IDResponse struct {
	ID string `json:"id"`
}

type OperationResult struct {
	Success    []string    `json:"success,omitempty"`
	Fail       []string    `json:"fail,omitempty"`
	FailReason FailReasons `json:"fail_reason,omitempty"`
	Message    string      `json:"message,omitempty"`
}

type IDsOperationResponse struct {
	Success []string `json:"success"`
	Fail    []string `json:"fail"`
}

// FailReasons maps an ID to its failure reason. The API sometimes sends an
// empty array instead of an empty object, so both are accepted.
type FailReasons map[string]string

func (f *FailReasons) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "null" || trimmed == "[]" {
		*f = nil
		return nil
	}

	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*f = m
	return nil
}

// Ptr returns a pointer to v, for optional request fields where the zero value
// is meaningful (for example renewal_status=0).
func Ptr[T any](v T) *T {
	return &v
}

// BoolToInt converts an on/off switch to the 1/0 value the API expects.
func BoolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

type Text string

func (t *Text) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*t = ""
		return nil
	}

	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	switch v := value.(type) {
	case string:
		*t = Text(v)
	case float64:
		*t = Text(strconv.FormatFloat(v, 'f', -1, 64))
	case bool:
		*t = Text(strconv.FormatBool(v))
	default:
		return fmt.Errorf("unsupported text value type %T", value)
	}

	return nil
}
