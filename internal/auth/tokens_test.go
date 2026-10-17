package auth

import (
	"testing"
	"time"
)

const (
	testSecret = "test-secret-key-32-characters-long!"
	testIssuer = "vaultix-api"
	testUserID = "507f1f77bcf86cd799439011"
	testEmail  = "user@example.com"
	testRole   = "user"
)

func TestIssueAccess_ValidateAccess_RoundTrip(t *testing.T) {
	token, err := IssueAccess(testUserID, testEmail, testRole, testSecret, testIssuer, time.Hour)
	if err != nil {
		t.Fatalf("IssueAccess returned error: %v", err)
	}
	if token == "" {
		t.Fatal("IssueAccess returned empty token")
	}

	claims, err := ValidateAccess(token, testSecret, testIssuer)
	if err != nil {
		t.Fatalf("ValidateAccess returned error: %v", err)
	}

	if claims["user_id"] != testUserID {
		t.Errorf("Expected user_id %q, got %q", testUserID, claims["user_id"])
	}
	if claims["email"] != testEmail {
		t.Errorf("Expected email %q, got %q", testEmail, claims["email"])
	}
	if claims["role"] != testRole {
		t.Errorf("Expected role %q, got %q", testRole, claims["role"])
	}
	if claims["iss"] != testIssuer {
		t.Errorf("Expected issuer %q, got %q", testIssuer, claims["iss"])
	}
}

func TestIssueRefresh_ValidateRefresh_RoundTrip(t *testing.T) {
	token, err := IssueRefresh(testUserID, testSecret, testIssuer, time.Hour)
	if err != nil {
		t.Fatalf("IssueRefresh returned error: %v", err)
	}

	userID, err := ValidateRefresh(token, testSecret, testIssuer)
	if err != nil {
		t.Fatalf("ValidateRefresh returned error: %v", err)
	}

	if userID != testUserID {
		t.Errorf("Expected userID %q, got %q", testUserID, userID)
	}
}

func TestValidateAccess_WrongSecret(t *testing.T) {
	token, _ := IssueAccess(testUserID, testEmail, testRole, testSecret, testIssuer, time.Hour)

	_, err := ValidateAccess(token, "wrong-secret", testIssuer)
	if err == nil {
		t.Error("ValidateAccess with wrong secret should return error")
	}
}

func TestValidateAccess_WrongIssuer(t *testing.T) {
	token, _ := IssueAccess(testUserID, testEmail, testRole, testSecret, testIssuer, time.Hour)

	_, err := ValidateAccess(token, testSecret, "wrong-issuer")
	if err == nil {
		t.Error("ValidateAccess with wrong issuer should return error")
	}
}

func TestValidateAccess_Expired(t *testing.T) {
	// Negative TTL creates an already-expired token
	token, _ := IssueAccess(testUserID, testEmail, testRole, testSecret, testIssuer, -time.Hour)

	_, err := ValidateAccess(token, testSecret, testIssuer)
	if err == nil {
		t.Error("ValidateAccess with expired token should return error")
	}
}

func TestValidateAccess_TamperedToken(t *testing.T) {
	token, _ := IssueAccess(testUserID, testEmail, testRole, testSecret, testIssuer, time.Hour)

	// Corrupt the token
	tampered := token[:len(token)-2] + "xx"
	_, err := ValidateAccess(tampered, testSecret, testIssuer)
	if err == nil {
		t.Error("ValidateAccess with tampered token should return error")
	}
}

func TestValidateAccess_GarbageToken(t *testing.T) {
	_, err := ValidateAccess("not-a-jwt-token", testSecret, testIssuer)
	if err == nil {
		t.Error("ValidateAccess with garbage token should return error")
	}
}

func TestValidateRefresh_GarbageToken(t *testing.T) {
	_, err := ValidateRefresh("not-a-jwt-token", testSecret, testIssuer)
	if err == nil {
		t.Error("ValidateRefresh with garbage token should return error")
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	h1 := HashToken("token-abc")
	h2 := HashToken("token-abc")

	if h1 != h2 {
		t.Error("HashToken should be deterministic")
	}

	if h1 == HashToken("token-def") {
		t.Error("Different tokens should produce different hashes")
	}
}

func TestIssueAccess_ExpiryClaim(t *testing.T) {
	ttl := time.Hour
	token, _ := IssueAccess(testUserID, testEmail, testRole, testSecret, testIssuer, ttl)

	claims, err := ValidateAccess(token, testSecret, testIssuer)
	if err != nil {
		t.Fatalf("ValidateAccess returned error: %v", err)
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		t.Fatal("exp claim should be a number")
	}

	expTime := time.Unix(int64(exp), 0)
	expectedMin := time.Now().Add(ttl - time.Minute)
	expectedMax := time.Now().Add(ttl + time.Minute)

	if expTime.Before(expectedMin) || expTime.After(expectedMax) {
		t.Errorf("exp claim %v not within expected range [%v, %v]", expTime, expectedMin, expectedMax)
	}
}
