package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/fortega2/zink/internal/config"
)

const (
	wwwAuthenticate    = "WWW-Authenticate"
	wwwAuthenticateVal = `Bearer realm="zink"`
	headerUserID       = "X-User-ID"
	headerUserEmail    = "X-User-Email"
	authHeaderParts    = 2
)

var ErrMissingToken = errors.New("missing or malformed authorization header")

func Auth(cfg config.AuthMiddleware) (Middleware, error) {
	keyData, err := os.ReadFile(cfg.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to read public key: %w", err)
	}

	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(keyData)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to parse public key: %w", err)
	}

	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(cfg.Issuer),
		jwt.WithAudience(cfg.Audience),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, err := extractBearer(r)
			if err != nil {
				unauthorized(w)
				return
			}

			claims := jwt.MapClaims{}
			if _, err := parser.ParseWithClaims(raw, claims, func(_ *jwt.Token) (any, error) {
				return pubKey, nil
			}); err != nil {
				unauthorized(w)
				return
			}

			enrichRequestHeaders(r, claims)
			next.ServeHTTP(w, r)
		})
	}, nil
}

func extractBearer(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", ErrMissingToken
	}

	parts := strings.SplitN(header, " ", authHeaderParts)
	if len(parts) != authHeaderParts || !strings.EqualFold(parts[0], "bearer") {
		return "", ErrMissingToken
	}

	return parts[1], nil
}

func enrichRequestHeaders(r *http.Request, claims jwt.MapClaims) {
	if sub, ok := claims["sub"].(string); ok && sub != "" {
		r.Header.Set(headerUserID, sub)
	}
	if email, ok := claims["email"].(string); ok && email != "" {
		r.Header.Set(headerUserEmail, email)
	}
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set(wwwAuthenticate, wwwAuthenticateVal)
	w.WriteHeader(http.StatusUnauthorized)
}
