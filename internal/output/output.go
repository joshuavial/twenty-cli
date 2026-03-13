package output

import (
	"encoding/json"
	"fmt"
	"io"
)

type ExitCode int

const (
	ExitOK       ExitCode = 0
	ExitUsage    ExitCode = 2
	ExitAuth     ExitCode = 3
	ExitAPI      ExitCode = 4
	ExitInternal ExitCode = 10
)

type ErrorKind string

const (
	ErrorKindUsage    ErrorKind = "usage"
	ErrorKindAuth     ErrorKind = "auth"
	ErrorKindAPI      ErrorKind = "api"
	ErrorKindInternal ErrorKind = "internal"
)

type Envelope struct {
	OK      bool   `json:"ok"`
	Command string `json:"command"`
	Data    any    `json:"data,omitempty"`
	Error   *Error `json:"error,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
}

type Error struct {
	Kind      ErrorKind `json:"kind"`
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Retryable bool      `json:"retryable,omitempty"`
	Details   any       `json:"details,omitempty"`
}

type Meta struct {
	PageInfo *PageInfo `json:"page_info,omitempty"`
	Warnings []Warning `json:"warnings,omitempty"`
}

type PageInfo struct {
	Limit      int    `json:"limit,omitempty"`
	Returned   int    `json:"returned,omitempty"`
	Total      int    `json:"total,omitempty"`
	NextCursor string `json:"next_cursor,omitempty"`
	PrevCursor string `json:"prev_cursor,omitempty"`
}

type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type APIErrorDetails struct {
	StatusCode int    `json:"status_code"`
	Body       string `json:"body,omitempty"`
}

type Result struct {
	Command string
	Data    any
	Meta    *Meta
	Text    string
}

type Failure struct {
	Command   string
	Kind      ErrorKind
	Code      string
	Message   string
	Details   any
	Meta      *Meta
	Retryable bool
	Text      string
}

func (f Failure) ExitCode() ExitCode {
	switch f.Kind {
	case ErrorKindUsage:
		return ExitUsage
	case ErrorKindAuth:
		return ExitAuth
	case ErrorKindAPI:
		return ExitAPI
	default:
		return ExitInternal
	}
}

func (f Failure) Envelope() Envelope {
	return Envelope{
		OK:      false,
		Command: f.Command,
		Meta:    f.Meta,
		Error: &Error{
			Kind:      f.Kind,
			Code:      f.Code,
			Message:   f.Message,
			Retryable: f.Retryable,
			Details:   f.Details,
		},
	}
}

func (r Result) Envelope() Envelope {
	return Envelope{
		OK:      true,
		Command: r.Command,
		Data:    r.Data,
		Meta:    r.Meta,
	}
}

func WriteJSON(w io.Writer, envelope Envelope) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(envelope)
}

func WriteText(w io.Writer, msg string) error {
	_, err := fmt.Fprintln(w, msg)
	return err
}

func WriteSuccessJSON(w io.Writer, result Result) error {
	return WriteJSON(w, result.Envelope())
}

func WriteFailureJSON(w io.Writer, failure Failure) error {
	return WriteJSON(w, failure.Envelope())
}

func WriteSuccessText(w io.Writer, result Result) error {
	msg := result.Text
	if msg == "" {
		msg = "ok"
	}

	return WriteText(w, msg)
}

func WriteFailureText(w io.Writer, failure Failure) error {
	msg := failure.Text
	if msg == "" {
		msg = failure.Message
	}

	return WriteText(w, msg)
}
