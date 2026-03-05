package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fortega2/zink/internal/config"
	"github.com/fortega2/zink/internal/middleware"
)

const (
	testIssuer   = "https://auth.example.com"
	testAudience = "zink-gateway"
	testUserID   = "user-123"
	bearerScheme = "Bearer "
)

func generateTestKeyPair(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	pubDER, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	require.NoError(t, err)

	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "public.pem")
	require.NoError(t, os.WriteFile(keyPath, pubPEM, 0600))

	return privKey, keyPath
}

func signToken(t *testing.T, privKey *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(privKey)
	require.NoError(t, err)
	return signed
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"sub": testUserID,
		"iss": testIssuer,
		"aud": jwt.ClaimStrings{testAudience},
		"exp": time.Now().Add(time.Hour).Unix(),
	}
}

func buildAuthMiddleware(t *testing.T, keyPath string) middleware.Middleware {
	t.Helper()
	cfg := config.AuthMiddleware{
		PublicKeyPath: keyPath,
		Issuer:        testIssuer,
		Audience:      testAudience,
	}
	mw, err := New(cfg)
	require.NoError(t, err)
	return mw
}

func TestAuthInvalidPEMFile(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "bad.pem")
	require.NoError(t, os.WriteFile(keyPath, []byte("not a valid pem"), 0600))

	cfg := config.AuthMiddleware{
		PublicKeyPath: keyPath,
		Issuer:        testIssuer,
		Audience:      testAudience,
	}
	_, err := New(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse public key")
}

func TestAuthMissingAuthorizationHeader(t *testing.T) {
	_, keyPath := generateTestKeyPair(t)
	handler := buildAuthMiddleware(t, keyPath)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, wwwAuthenticateVal, w.Header().Get(wwwAuthenticate))
}

func TestAuthNonBearerScheme(t *testing.T) {
	_, keyPath := generateTestKeyPair(t)
	handler := buildAuthMiddleware(t, keyPath)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMalformedToken(t *testing.T) {
	_, keyPath := generateTestKeyPair(t)
	handler := buildAuthMiddleware(t, keyPath)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer not.a.jwt")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthWrongAlgorithm(t *testing.T) {
	_, keyPath := generateTestKeyPair(t)
	handler := buildAuthMiddleware(t, keyPath)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	hmacToken := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims())
	signed, err := hmacToken.SignedString([]byte("secret"))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", bearerScheme+signed)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthInvalidSignature(t *testing.T) {
	_, keyPath := generateTestKeyPair(t)
	handler := buildAuthMiddleware(t, keyPath)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	signed := signToken(t, otherKey, validClaims())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", bearerScheme+signed)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthExpiredToken(t *testing.T) {
	privKey, keyPath := generateTestKeyPair(t)
	handler := buildAuthMiddleware(t, keyPath)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	claims := validClaims()
	claims["exp"] = time.Now().Add(-time.Hour).Unix()
	signed := signToken(t, privKey, claims)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", bearerScheme+signed)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthWrongIssuer(t *testing.T) {
	privKey, keyPath := generateTestKeyPair(t)
	handler := buildAuthMiddleware(t, keyPath)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	claims := validClaims()
	claims["iss"] = "https://other-issuer.com"
	signed := signToken(t, privKey, claims)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", bearerScheme+signed)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthWrongAudience(t *testing.T) {
	privKey, keyPath := generateTestKeyPair(t)
	handler := buildAuthMiddleware(t, keyPath)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	claims := validClaims()
	claims["aud"] = jwt.ClaimStrings{"other-service"}
	signed := signToken(t, privKey, claims)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", bearerScheme+signed)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthValidTokenWithoutEmail(t *testing.T) {
	privKey, keyPath := generateTestKeyPair(t)

	var capturedReq *http.Request
	handler := buildAuthMiddleware(t, keyPath)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		w.WriteHeader(http.StatusOK)
	}))

	signed := signToken(t, privKey, validClaims())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", bearerScheme+signed)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedReq)
	assert.Equal(t, testUserID, capturedReq.Header.Get(headerUserID))
	assert.Empty(t, capturedReq.Header.Get(headerUserEmail))
}

func TestAuthValidTokenWithEmail(t *testing.T) {
	privKey, keyPath := generateTestKeyPair(t)

	var capturedReq *http.Request
	handler := buildAuthMiddleware(t, keyPath)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		w.WriteHeader(http.StatusOK)
	}))

	claims := validClaims()
	claims["email"] = "user@example.com"
	signed := signToken(t, privKey, claims)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", bearerScheme+signed)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedReq)
	assert.Equal(t, testUserID, capturedReq.Header.Get(headerUserID))
	assert.Equal(t, "user@example.com", capturedReq.Header.Get(headerUserEmail))
}
