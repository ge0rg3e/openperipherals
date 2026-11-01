package wallpaper

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestVideoStreamRangesAndIsolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "video with spaces & unicode Ω.mp4")
	if err := os.WriteFile(path, []byte("0123456789"), 0600); err != nil {
		t.Fatal(err)
	}
	handler := videoHandler(path)
	request := httptest.NewRequest(http.MethodGet, "/video", nil)
	request.Header.Set("Range", "bytes=3-5")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusPartialContent || response.Body.String() != "345" || response.Header().Get("Content-Range") != "bytes 3-5/10" {
		t.Fatalf("range response: %d %v %q", response.Code, response.Header(), response.Body.String())
	}
	for _, test := range []struct {
		method, path string
		status       int
	}{
		{http.MethodGet, "/", http.StatusOK},
		{http.MethodHead, "/video", http.StatusOK},
		{http.MethodPost, "/video", http.StatusMethodNotAllowed},
		{http.MethodGet, "/other.mp4", http.StatusNotFound},
		{http.MethodGet, "/../secret", http.StatusNotFound},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(test.method, test.path, nil))
		if response.Code != test.status {
			t.Errorf("%s %s: got %d, want %d", test.method, test.path, response.Code, test.status)
		}
	}
}
