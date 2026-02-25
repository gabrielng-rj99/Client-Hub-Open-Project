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
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// IPRateLimiter holds rate limiters for each IP
type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  sync.RWMutex
	r   rate.Limit
	b   int
}

// NewIPRateLimiter creates a new rate limiter
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	i := &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		r:   r,
		b:   b,
	}

	// Clean up old entries periodically to prevent memory leaks
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			i.cleanup()
		}
	}()

	return i
}

// AddIP adds an IP to the map
func (i *IPRateLimiter) AddIP(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter := rate.NewLimiter(i.r, i.b)
	i.ips[ip] = limiter
	return limiter
}

// GetLimiter returns the rate limiter for the provided IP address
// If it doesn't exist, it creates a new one
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.RLock()
	limiter, exists := i.ips[ip]
	i.mu.RUnlock()

	if !exists {
		return i.AddIP(ip)
	}

	return limiter
}

func (i *IPRateLimiter) cleanup() {
	i.mu.Lock()
	// In a real implementation, we would track last access time and remove old ones
	// For now, we just reset the map if it gets too big to avoid memory exhaustion
	if len(i.ips) > 10000 {
		i.ips = make(map[string]*rate.Limiter)
	}
	i.mu.Unlock()
}

// RateLimitMiddleware enforces rate limiting per IP
func (s *Server) rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ipPtr := getIPAddress(r)
		ip := "unknown"
		if ipPtr != nil {
			ip = *ipPtr
		}

		limiter := s.rateLimiter.GetLimiter(ip)
		if !limiter.Allow() {
			respondError(w, http.StatusTooManyRequests, "Rate limit exceeded")
			return
		}
		next(w, r)
	}
}

// SecurityHeadersMiddleware adds security headers to responses
func (s *Server) securityHeadersMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// XSS Protection (legacy but useful)
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		// Content Security Policy (Basic)
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		// HSTS (Strict Transport Security) - 1 year
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		next(w, r)
	}
}

// requirePermissionAction restricts access to users with a specific permission
// Must be used after authMiddleware
func (s *Server) requirePermissionAction(resource, action string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := extractTokenFromHeader(r)

		// 1. Extração levíssima do UUID sem checar banco
		claims, err := ExtractJWTClaimsUnverified(tokenString)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "Token inválido ou corrompido")
			return
		}

		// 2. A Super Query: Busca Auth Secret, Status de Bloqueio e Permissão RBAC (1 ida ao Postgres)
		access, err := s.roleStore.CheckUserAccess(claims.UserID, resource, action)
		if err != nil {
			// Se o Check falhar, o auth_secret estará vazio e o JWT não vai validar de qualquer forma.
			// Falha limpa para evitar brute force.
			respondError(w, http.StatusUnauthorized, "Não autorizado")
			return
		}

		// 3. Validação Matemática da Assinatura (100% CPU, 0 banco)
		_, err = ValidateJWTSignature(tokenString, access.AuthSecret)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "Assinatura do token inválida")
			return
		}

		// 4. Checagem de Bloqueio
		if access.IsBlocked {
			respondError(w, http.StatusLocked, "Sua conta está bloqueada - acesso negado")
			return
		}

		// 5. O Veridito Final da Autorização (A Catraca RBAC)
		if !access.HasAccess {
			respondError(w, http.StatusForbidden, "Permissão negada")
			return
		}

		next(w, r)
	}
}

// requirePermissionByMethod restricts access based on HTTP method mapping to actions
// methodMap maps HTTP methods to action strings (e.g. map[string]string{http.MethodGet: "read", http.MethodPost: "create"})
// Must be used after authMiddleware
func (s *Server) requirePermissionByMethod(resource string, methodMap map[string]string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		action, exists := methodMap[r.Method]
		if !exists {
			respondError(w, http.StatusMethodNotAllowed, "Method requires permission mapping")
			return
		}

		tokenString := extractTokenFromHeader(r)

		claims, err := ExtractJWTClaimsUnverified(tokenString)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "Token inválido ou corrompido")
			return
		}

		access, err := s.roleStore.CheckUserAccess(claims.UserID, resource, action)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "Não autorizado")
			return
		}

		_, err = ValidateJWTSignature(tokenString, access.AuthSecret)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "Assinatura do token inválida")
			return
		}

		if access.IsBlocked {
			respondError(w, http.StatusLocked, "Sua conta está bloqueada - acesso negado")
			return
		}

		if !access.HasAccess {
			respondError(w, http.StatusForbidden, "Permissão negada")
			return
		}

		next(w, r)
	}
}
