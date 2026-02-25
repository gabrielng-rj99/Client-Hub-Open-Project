/*
 * Client Hub Open Project
 * Copyright (C) 2025 Client Hub Contributors
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published
 * by the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package server

import (
	"errors"
	"fmt"
	"time"

	"Open-Generic-Hub/backend/config"
	"Open-Generic-Hub/backend/domain"
	"Open-Generic-Hub/backend/store"

	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims define o payload do token
// NOTE: Role was intentionally removed from JWT claims for security.
// All authorization checks must be performed via database lookups using roleStore.
// This ensures role changes take effect immediately without requiring token re-issuance.
type JWTClaims struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	TokenType string `json:"token_type,omitempty"` // Should be empty for access tokens
	jwt.RegisteredClaims
}

// getGlobalSecret gets the global secret from configuration
func getGlobalSecret() []byte {
	cfg := config.GetConfig()
	return []byte(cfg.JWT.SecretKey)
}

// Gera a chave de assinatura dinâmica para o usuário
func getUserSigningKey(user *domain.User) ([]byte, error) {
	if user == nil || user.AuthSecret == "" {
		return nil, errors.New("auth_secret do usuário não disponível")
	}
	return append(getGlobalSecret(), []byte(user.AuthSecret)...), nil
}

// Gera um JWT para o usuário autenticado
// NOTE: Role is intentionally NOT included in the JWT.
// All authorization must be performed via database lookups for immediate effect of role changes.
func GenerateJWT(user *domain.User) (string, error) {
	signingKey, err := getUserSigningKey(user)
	if err != nil {
		return "", err
	}

	cfg := config.GetConfig()
	claims := JWTClaims{
		UserID:   user.ID,
		Username: derefString(user.Username),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.JWT.ExpirationTime)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(signingKey)
}

// ========== REFRESH TOKEN ==========

type RefreshTokenClaims struct {
	UserID    string `json:"user_id"`
	TokenType string `json:"token_type"` // Must be "refresh" to distinguish from access tokens
	jwt.RegisteredClaims
}

// Gera um refresh token para o usuário autenticado
func GenerateRefreshToken(user *domain.User) (string, error) {
	signingKey, err := getUserSigningKey(user)
	if err != nil {
		return "", err
	}

	cfg := config.GetConfig()
	claims := RefreshTokenClaims{
		UserID:    user.ID,
		TokenType: "refresh", // Explicitly mark as refresh token
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.JWT.RefreshExpirationTime)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(signingKey)
}

// Valida o refresh token e retorna as claims se válido
func ValidateRefreshToken(tokenString string, userStore *store.UserStore) (*RefreshTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		claims, ok := token.Claims.(*RefreshTokenClaims)
		if !ok {
			return nil, errors.New("formato de claims inválido")
		}
		// Buscar usuário no banco
		user, err := userStore.GetUserByID(claims.UserID)
		if err != nil || user == nil || user.AuthSecret == "" {
			return nil, errors.New("usuário não encontrado ou auth_secret ausente")
		}
		return getUserSigningKey(user)
	})

	if err != nil {
		return nil, fmt.Errorf("refresh token inválido: %w", err)
	}

	claims, ok := token.Claims.(*RefreshTokenClaims)
	if !ok || !token.Valid {
		return nil, errors.New("refresh token inválido ou expirado")
	}

	// SECURITY: Verify this is actually a refresh token, not an access token
	// Access tokens don't have token_type field, so they will fail this check
	if claims.TokenType != "refresh" {
		return nil, errors.New("token fornecido não é um refresh token válido")
	}

	// SECURITY: Check if user is blocked - even when refreshing token
	user, err := userStore.GetUserByID(claims.UserID)
	if err != nil || user == nil {
		return nil, errors.New("usuário não encontrado")
	}

	now := time.Now()
	if user.LockedUntil != nil && now.Before(*user.LockedUntil) {
		return nil, errors.New("usuário bloqueado - acesso recusado")
	}

	return claims, nil
}

// Extrai o JWT sem validar a assinatura para podermos ler o UUID antes de ir no banco
func ExtractJWTClaimsUnverified(tokenString string) (*JWTClaims, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &JWTClaims{})
	if err != nil {
		return nil, fmt.Errorf("formato de token inválido: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, errors.New("formato de claims inválido")
	}

	return claims, nil
}

// Valida o JWT matematicamente contra um secret_secret em memória sem fazer novas consultas no BD
func ValidateJWTSignature(tokenString string, authSecret string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		cfg := config.GetConfig()
		signingKey := append([]byte(cfg.JWT.SecretKey), []byte(authSecret)...)
		return signingKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("assinatura do token inválida: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("token inválido ou expirado")
	}

	// SECURITY: Reject refresh tokens - access tokens should not have token_type claim
	if claims, ok := token.Claims.(*JWTClaims); ok {
		if claims.TokenType != "" {
			return nil, errors.New("token fornecido não é um access token válido")
		}
		return claims, nil
	}

	return nil, errors.New("formato de claims inválido")
}

// Helper para desreferenciar ponteiros de string
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Extrai o token do header Authorization
func extractTokenFromHeader(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}

// Extrai e valida as claims do JWT no request
func getClaimsFromRequest(r *http.Request, userStore *store.UserStore) (*JWTClaims, error) {
	tokenString := extractTokenFromHeader(r)
	if tokenString == "" {
		return nil, errors.New("token não fornecido")
	}

	claims, err := ExtractJWTClaimsUnverified(tokenString)
	if err != nil {
		return nil, errors.New("token inválido ou expirado")
	}

	return claims, nil
}
