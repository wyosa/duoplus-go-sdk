package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

type PaginationRequest struct {
	// One-based; defaults to 1.
	Page int `json:"page,omitempty"`
	// Defaults to 10; maximum depends on the endpoint.
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

// OperationResult may contain Fail entries even when the method returns no error.
type OperationResult struct {
	// Accepted IDs; asynchronous operations still need status polling.
	Success    []string    `json:"success,omitempty"`
	Fail       []string    `json:"fail,omitempty"`
	FailReason FailReasons `json:"fail_reason,omitempty"`
	Message    string      `json:"message,omitempty"`
}

type IDsOperationResponse struct {
	Success []string `json:"success"`
	Fail    []string `json:"fail"`
}

// FailReasons accepts an ID-to-message object, an empty array, or null.
type FailReasons map[string]string

func (f *FailReasons) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var items []json.RawMessage
		if err := json.Unmarshal(trimmed, &items); err != nil {
			return err
		}
		if len(items) != 0 {
			return fmt.Errorf("fail reasons: expected an object or an empty array")
		}
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

// Ptr supplies an explicit value for an optional field; nil omits it.
func Ptr[T any](v T) *T {
	return &v
}

func BoolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// Text decodes strings, booleans, numbers, or null without rounding numeric values.
type Text string

func (t *Text) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err != nil {
			return err
		}
		return fmt.Errorf("text: expected a single JSON value")
	}

	switch v := value.(type) {
	case string:
		*t = Text(v)
	case json.Number:
		*t = Text(v.String())
	case bool:
		*t = Text(strconv.FormatBool(v))
	case nil:
		*t = ""
	default:
		return fmt.Errorf("unsupported text value type %T", value)
	}

	return nil
}
