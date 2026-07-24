package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// NewSessionToken gera um token de sessão aleatório (256 bits) codificado
// em base64url — é o valor que vai no cookie, nunca persistido em texto
// puro no banco (só o hash, via HashToken).
func NewSessionToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("gerar token de sessão: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// HashToken produz o SHA-256 hex do token — é o que fica em
// sessions.token_hash e session_lookup.token_hash. Sessão é opaca e
// server-side revogável; SHA-256 simples basta pois o token já tem alta
// entropia (256 bits) e o hash aqui é só para não guardar o segredo em
// texto puro, não para resistir a brute-force de senha humana.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
