package output

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestNewSuccess(t *testing.T) {
	resp := NewSuccess(`cloud_upload`, map[string]any{`size_bytes`: 10})
	if !resp.Success {
		t.Fatalf("expected success=true")
	}
	if resp.Action != `cloud_upload` {
		t.Fatalf("unexpected action: %s", resp.Action)
	}
	if resp.Error != nil {
		t.Fatalf("expected error=nil")
	}
}

func TestNewFailure(t *testing.T) {
	resp := NewFailure(`local_read`, `ERR_PATH_TRAVERSAL`, `denied`)
	if resp.Success {
		t.Fatalf("expected success=false")
	}
	if resp.Data != nil {
		t.Fatalf("expected data=nil")
	}
	if resp.Error == nil {
		t.Fatalf("expected error != nil")
	}
	if resp.Error.Code != `ERR_PATH_TRAVERSAL` {
		t.Fatalf("unexpected error code: %s", resp.Error.Code)
	}
	if resp.Error.Message != `denied` {
		t.Fatalf("unexpected error message: %s", resp.Error.Message)
	}
}

func TestWriteJSONShape(t *testing.T) {
	var buf bytes.Buffer
	resp := NewFailure(`local_read`, `ERR_PATH_TRAVERSAL`, `Access denied`)
	if err := Write(&buf, resp); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if _, ok := parsed[`success`]; !ok {
		t.Fatalf("missing success")
	}
	if _, ok := parsed[`action`]; !ok {
		t.Fatalf("missing action")
	}
	if _, ok := parsed[`data`]; !ok {
		t.Fatalf("missing data")
	}
	if _, ok := parsed[`error`]; !ok {
		t.Fatalf("missing error")
	}
}

func TestPrintFailureWritesStdoutAndStderr(t *testing.T) {
	originalOut := Stdout
	originalErr := Stderr
	defer func() {
		Stdout = originalOut
		Stderr = originalErr
	}()

	var out bytes.Buffer
	var errBuf bytes.Buffer
	Stdout = &out
	Stderr = &errBuf

	if err := PrintFailure(`local_read`, `ERR_PATH_TRAVERSAL`, `Access denied`); err != nil {
		t.Fatalf("PrintFailure failed: %v", err)
	}

	var parsed map[string]any
	if unmarshalErr := json.Unmarshal(out.Bytes(), &parsed); unmarshalErr != nil {
		t.Fatalf("stdout is not valid json: %v", unmarshalErr)
	}
	if success, ok := parsed[`success`].(bool); !ok || success {
		t.Fatalf("expected success=false")
	}
	if errBuf.Len() == 0 {
		t.Fatalf("stderr should contain summary")
	}
}
