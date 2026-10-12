package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/cznUSER/B-HIS/internal/domain"
)

var (
	ErrExpired = errors.New("token expired")
	ErrInvalid = errors.New("token invalid")
)

// Claims are the Phase1 access-token claims (H03 §3.0.1): sub, username, roles, iat, exp, jti.
type Claims struct {
	jwt.RegisteredClaims
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

// Issuer signs HS256 access tokens. There is no refresh token in Phase1.
type Issuer struct {
	secret []byte
	ttl    time.Duration
}

func NewIssuer(secret string, ttl time.Duration) (*Issuer, error) {
	if len(secret) < 16 {
		return nil, fmt.Errorf("jwt secret must be at least 16 characters")
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("access token ttl must be positive")
	}
	return &Issuer{secret: []byte(secret), ttl: ttl}, nil
}

func (i *Issuer) TTL() time.Duration { return i.ttl }

// Issue returns a signed token, its expiry, and jti.
func (i *Issuer) Issue(user domain.User, now time.Time) (string, time.Time, string, error) {
	jti, err := randomID()
	if err != nil {
		return "", time.Time{}, "", err
	}
	now = now.UTC()
	exp := now.Add(i.ttl)
	roles := user.RoleCodes()
	if roles == nil {
		roles = []string{}
	}
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			ID:        jti,
		},
		Username: user.Username,
		Roles:    roles,
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(i.secret)
	if err != nil {
		return "", time.Time{}, "", err
	}
	return signed, exp, jti, nil
}

func (i *Issuer) Parse(token string, now time.Time) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		return i.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithTimeFunc(func() time.Time { return now }))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpired
		}
		return nil, ErrInvalid
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid || claims.Subject == "" || claims.ID == "" {
		return nil, ErrInvalid
	}
	return claims, nil
}

func randomID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
