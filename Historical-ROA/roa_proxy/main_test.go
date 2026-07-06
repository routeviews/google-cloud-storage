package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandleROA(t *testing.T) {
	// 1. Create a mock backend server
	expectedBody := `{"roas": [{"asn": "AS1234", "prefix": "1.2.3.0/24", "maxLength": 24, "ta": "test-ta"}]}`
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedBody))
	}))
	defer backend.Close()

	// 2. Temporarily override targetURL flag/variable
	oldTargetURL := *targetURL
	*targetURL = backend.URL
	defer func() { *targetURL = oldTargetURL }()

	// 3. Create a request to our proxy
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// 4. Record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleROA)

	// 5. Serve the request
	handler.ServeHTTP(rr, req)

	// 6. Assertions
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, "application/json")
	}

	if rr.Body.String() != expectedBody {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expectedBody)
	}
}

func TestHandleROA_MethodNotAllowed(t *testing.T) {
	req, err := http.NewRequest("POST", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleROA)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code for POST: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestHandleROA_BackendError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer backend.Close()

	oldTargetURL := *targetURL
	*targetURL = backend.URL
	defer func() { *targetURL = oldTargetURL }()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleROA)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code on backend error: got %v want %v", status, http.StatusInternalServerError)
	}
}

func TestHandleROA_ConnectionFailure(t *testing.T) {
	oldTargetURL := *targetURL
	*targetURL = "http://invalid-url-that-should-fail.local"
	defer func() { *targetURL = oldTargetURL }()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleROA)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code on connection failure: got %v want %v", status, http.StatusInternalServerError)
	}
}

func TestHandleROA_Timeout(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	oldTargetURL := *targetURL
	*targetURL = backend.URL
	defer func() { *targetURL = oldTargetURL }()

	oldTimeout := *timeout
	*timeout = 50 * time.Millisecond
	defer func() { *timeout = oldTimeout }()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleROA)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code on timeout: got %v want %v", status, http.StatusInternalServerError)
	}
}
