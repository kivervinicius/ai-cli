package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteErrorPreservesMessageAndAddsStableCode(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeError(recorder, http.StatusNotFound, "project not found")

	var body APIError
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error != "project not found" || body.Code != "PROJECT_NOT_FOUND" {
		t.Fatalf("unexpected error response: %#v", body)
	}
}

func TestRuntimeDetailResponsePreservesContractFields(t *testing.T) {
	response := RuntimeDetailResponse{}
	if response.Session.RuntimeID != "" || response.Capabilities != nil {
		t.Fatalf("zero value unexpectedly populated: %#v", response)
	}
}

func TestStableErrorCodeDoesNotTreatUnknownAsZero(t *testing.T) {
	if got := stableErrorCode(http.StatusServiceUnavailable, "quota unavailable"); got != "QUOTA_UNKNOWN" {
		t.Fatalf("code = %q, want QUOTA_UNKNOWN", got)
	}
	if got := stableErrorCode(http.StatusInternalServerError, "something failed"); got != "INTERNAL_ERROR" {
		t.Fatalf("code = %q, want INTERNAL_ERROR", got)
	}
}
