package auth_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wilgnert/chirpy/internal/auth"
)

func TestDifferentPaswordsHashToDifferentHashes(t *testing.T) {
	pass1 := "pass1"
	pass2 := "pass2"
	hash1, err := auth.HashPassword(pass1)
	if err != nil{
		t.Errorf("Unexpected error: %v", err)
	} 
	hash2, err := auth.HashPassword(pass2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if hash1 == hash2 {
		t.Errorf("Different passwords hashed to the same value")
	}
}

func TestCanDecode(t *testing.T) {
	pass := "supersecretcode"
	hash, err := auth.HashPassword(pass)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := auth.CheckPasswordHash(hash, pass); err != nil {
		t.Errorf("hash did not checkout to password")
	}
}

func TestMakeAndValidateJWT(t *testing.T) {
	userID := uuid.New()
	secret := "supersecretkey"
	expiresIn := time.Minute * 5

	// Create a JWT
	token, err := auth.MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("unexpected error creating JWT: %v", err)
	}

	// Validate the JWT
	parsedID, err := auth.ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("unexpected error validating JWT: %v", err)
	}

	// Check if the parsed ID matches the original user ID
	if parsedID != userID {
		t.Errorf("expected userID %v, got %v", userID, parsedID)
	}
}

func TestExpiredJWT(t *testing.T) {
	userID := uuid.New()
	secret := "supersecretkey"
	expiresIn := -time.Minute // Token already expired

	// Create an expired JWT
	token, err := auth.MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("unexpected error creating JWT: %v", err)
	}

	// Validate the expired JWT
	_, err = auth.ValidateJWT(token, secret)
	if err == nil {
		t.Errorf("expected error validating expired JWT, got none")
	}
}

func TestJWTWithWrongSecret(t *testing.T) {
	userID := uuid.New()
	secret := "supersecretkey"
	wrongSecret := "wrongsecretkey"
	expiresIn := time.Minute * 5

	// Create a JWT
	token, err := auth.MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("unexpected error creating JWT: %v", err)
	}

	// Validate the JWT with the wrong secret
	_, err = auth.ValidateJWT(token, wrongSecret)
	if err == nil {
		t.Errorf("expected error validating JWT with wrong secret, got none")
	}
}