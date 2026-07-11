package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenIssuer   = "allinme.core-api"
	TokenAudience = "allinme.schema-ui"
	TokenTTL      = 15 * time.Minute
	ClockLeeway   = 30 * time.Second
)

type Clock func() time.Time

type Claims struct {
	Role Role `json:"role"`
	jwt.RegisteredClaims
}

func (claims Claims) Validate() error {
	if claims.Subject == "" || claims.ID == "" || claims.IssuedAt == nil {
		return fmt.Errorf("required claims are missing")
	}
	if !claims.Role.Valid() {
		return fmt.Errorf("invalid role claim")
	}
	return nil
}

type Tokens struct {
	key    []byte
	clock  Clock
	parser *jwt.Parser
}

func NewTokens(key []byte, clock Clock) (*Tokens, error) {
	if len(key) < 32 {
		return nil, fmt.Errorf("JWT signing key must be at least 32 bytes")
	}
	if clock == nil {
		clock = time.Now
	}
	keyCopy := append([]byte(nil), key...)
	return &Tokens{
		key:   keyCopy,
		clock: clock,
		parser: jwt.NewParser(
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			jwt.WithExpirationRequired(),
			jwt.WithIssuer(TokenIssuer),
			jwt.WithAudience(TokenAudience),
			jwt.WithIssuedAt(),
			jwt.WithTimeFunc(clock),
			jwt.WithLeeway(ClockLeeway),
		),
	}, nil
}

func (tokens *Tokens) Issue(userID string, role Role, tokenID string) (string, time.Time, error) {
	if userID == "" || tokenID == "" || !role.Valid() {
		return "", time.Time{}, fmt.Errorf("invalid token identity")
	}
	issuedAt := tokens.clock().UTC()
	expiresAt := issuedAt.Add(TokenTTL)
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: TokenIssuer, Subject: userID, Audience: jwt.ClaimStrings{TokenAudience},
			ExpiresAt: jwt.NewNumericDate(expiresAt), IssuedAt: jwt.NewNumericDate(issuedAt), ID: tokenID,
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(tokens.key)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return token, expiresAt, nil
}

func (tokens *Tokens) Parse(encoded string) (Claims, error) {
	claims := Claims{}
	token, err := tokens.parser.ParseWithClaims(encoded, &claims, func(token *jwt.Token) (any, error) {
		return tokens.key, nil
	})
	if err != nil || token == nil || !token.Valid {
		return Claims{}, fmt.Errorf("invalid access token")
	}
	return claims, nil
}
