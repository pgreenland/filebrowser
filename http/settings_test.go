package fbhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/filebrowser/filebrowser/v2/settings"
	"github.com/filebrowser/filebrowser/v2/users"
)

func TestSettingsRotateSigningKeyInvalidatesExistingToken(t *testing.T) {
	oldKey := []byte("test-signing-key")
	perm := users.Permissions{Admin: true}
	st := scopedUserStorage(t, t.TempDir(), perm, oldKey)
	token := signToken(t, perm, oldKey)

	rotateReq := httptest.NewRequest(http.MethodPost, "/settings/key", http.NoBody)
	rotateReq.Header.Set("X-Auth", token)
	rotateResp := httptest.NewRecorder()
	handle(settingsRotateSigningKeyHandler, "", st, &settings.Server{}).ServeHTTP(rotateResp, rotateReq)
	if rotateResp.Code != http.StatusOK {
		t.Fatalf("expected rotate to succeed, got %d body=%q", rotateResp.Code, rotateResp.Body.String())
	}

	saved, err := st.Settings.Get()
	if err != nil {
		t.Fatalf("failed to read settings after rotate: %v", err)
	}
	if len(saved.Key) == 0 {
		t.Fatal("expected non-empty signing key after rotation")
	}
	if string(saved.Key) == string(oldKey) {
		t.Fatal("expected signing key to change")
	}

	protected := withUser(func(w http.ResponseWriter, _ *http.Request, _ *data) (int, error) {
		_, writeErr := w.Write([]byte("ok"))
		return 0, writeErr
	})
	protectedReq := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	protectedReq.Header.Set("X-Auth", token)
	protectedResp := httptest.NewRecorder()
	handle(protected, "", st, &settings.Server{}).ServeHTTP(protectedResp, protectedReq)
	if protectedResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected old token to be rejected, got %d body=%q", protectedResp.Code, protectedResp.Body.String())
	}
}
