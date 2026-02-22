package oauth

import (
	"testing"
	"time"
)

func TestManager_FullFlow(t *testing.T) {
	mgr := NewManager()
	
	// Test 1: Start OAuth flow
	session, err := mgr.Start("github-copilot", "http://localhost/callback")
	if err != nil {
		t.Fatalf("Start OAuth: %v", err)
	}
	if session.Status != SessionStatusPending {
		t.Errorf("Expected pending status, got %s", session.Status)
	}
	
	// Test 2: Complete OAuth flow
	session, err = mgr.Complete("github-copilot", session.State, "auth_code_123")
	if err != nil {
		t.Fatalf("Complete OAuth: %v", err)
	}
	if session.Status != SessionStatusConnected {
		t.Errorf("Expected connected status, got %s", session.Status)
	}
	if session.Token == nil {
		t.Fatal("Expected token, got nil")
	}
	
	// Test 3: Token expiration check
	if session.Token.IsExpired() {
		t.Error("Token should not be expired immediately")
	}
	if session.Token.TimeToExpiry() <= 0 {
		t.Error("Token should have positive time to expiry")
	}
	
	// Test 4: Validate token
	valid, metadata, err := mgr.ValidateWithMetadata("github-copilot", session.Token.AccessToken)
	if err != nil {
		t.Fatalf("Validate token: %v", err)
	}
	if !valid {
		t.Error("Token should be valid")
	}
	if metadata == nil {
		t.Error("Expected metadata")
	}
	
	// Test 5: Refresh token
	oldToken := session.Token.AccessToken
	newToken, err := mgr.Refresh("github-copilot", session.Token.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh token: %v", err)
	}
	if newToken.AccessToken == oldToken {
		t.Error("Access token should be different after refresh")
	}
	
	// Test 6: Get session
	retrieved, found := mgr.GetSession(session.ID)
	if !found {
		t.Fatal("Session not found")
	}
	if retrieved.ID != session.ID {
		t.Error("Retrieved session ID mismatch")
	}
	
	// Test 7: Revoke session
	err = mgr.RevokeSession(session.ID)
	if err != nil {
		t.Fatalf("Revoke session: %v", err)
	}
	
	// Test 8: Verify revoked
	_, found = mgr.GetSession(session.ID)
	if found {
		t.Error("Revoked session should not be accessible")
	}
}

func TestToken_IsExpired(t *testing.T) {
	// Test expired token
	expired := &Token{
		AccessToken: "test",
		ExpiresAt:   time.Now().UTC().Add(-1 * time.Hour),
	}
	if !expired.IsExpired() {
		t.Error("Expired token should return true")
	}
	
	// Test valid token
	valid := &Token{
		AccessToken: "test",
		ExpiresAt:   time.Now().UTC().Add(1 * time.Hour),
	}
	if valid.IsExpired() {
		t.Error("Valid token should return false")
	}
	
	// Test token with no expiry
	noExpiry := &Token{
		AccessToken: "test",
	}
	if noExpiry.IsExpired() {
		t.Error("Token with no expiry should not be expired")
	}
}

func TestSession_IsExpired(t *testing.T) {
	// Test pending session expiry (>10 minutes)
	oldPending := &Session{
		Status:    SessionStatusPending,
		CreatedAt: time.Now().UTC().Add(-11 * time.Minute),
	}
	if !oldPending.IsExpired() {
		t.Error("Old pending session should be expired")
	}
	
	// Test fresh pending session
	freshPending := &Session{
		Status:    SessionStatusPending,
		CreatedAt: time.Now().UTC(),
	}
	if freshPending.IsExpired() {
		t.Error("Fresh pending session should not be expired")
	}
	
	// Test connected session with expired token
	connectedExpired := &Session{
		Status: SessionStatusConnected,
		Token: &Token{
			ExpiresAt: time.Now().UTC().Add(-1 * time.Hour),
		},
	}
	if !connectedExpired.IsExpired() {
		t.Error("Connected session with expired token should be expired")
	}
}

func TestManager_CleanupExpiredSessions(t *testing.T) {
	mgr := NewManager()
	
	// Create and expire a session
	session, _ := mgr.Start("github-copilot", "")
	mgr.mu.Lock()
	session.CreatedAt = time.Now().UTC().Add(-11 * time.Minute)
	mgr.mu.Unlock()
	
	// Cleanup
	count := mgr.CleanupExpiredSessions()
	if count != 1 {
		t.Errorf("Expected 1 expired session cleaned, got %d", count)
	}
	
	// Verify session removed
	_, found := mgr.GetSession(session.ID)
	if found {
		t.Error("Expired session should be cleaned up")
	}
}
