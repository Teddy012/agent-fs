package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

var (
	Stdout io.Writer = os.Stdout
	Stderr io.Writer = os.Stderr
)

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Response struct {
	Success bool       `json:"success"`
	Action  string     `json:"action"`
	Data    any        `json:"data"`
	Error   *ErrorInfo `json:"error"`
}

func NewSuccess(action string, data any) Response {
	return Response{
		Success: true,
		Action:  action,
		Data:    data,
		Error:   nil,
	}
}

func NewFailure(action, code, message string) Response {
	return Response{
		Success: false,
		Action:  action,
		Data:    nil,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	}
}

func Write(w io.Writer, resp Response) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(resp)
}

func PrintSuccess(action string, data any) error {
	return Write(Stdout, NewSuccess(action, data))
}

func PrintFailure(action, code, message string) error {
	resp := NewFailure(action, code, message)
	if err := Write(Stdout, resp); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(Stderr, "[%s] %s\n", code, message); err != nil {
		return err
	}
	return nil
}
