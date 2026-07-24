package security

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/magicvr/allinme.core-api/internal/domain"
	"github.com/magicvr/allinme.core-api/internal/port"
)

type jwtClaims struct {
	Username string   `json:"username"`
	Name     string   `json:"name"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// JWTService issues HS256 access tokens.
type JWTService struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewJWTService constructs a TokenService.
func NewJWTService(secret string, ttl time.Duration) *JWTService {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &JWTService{
		secret: []byte(secret),
		ttl:    ttl,
		now:    time.Now,
	}
}

// Issue implements port.TokenService.
func (s *JWTService) Issue(ctx context.Context, user domain.User) (string, int64, error) {
	if err := ctx.Err(); err != nil {
		return "", 0, err
	}
	now := s.now()
	exp := now.Add(s.ttl)
	claims := jwtClaims{
		Username: user.Username,
		Name:     user.Name,
		Roles:    user.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString(s.secret)
	if err != nil {
		return "", 0, fmt.Errorf("jwt sign: %w", err)
	}
	return signed, int64(s.ttl.Seconds()), nil
}

// Parse implements port.TokenService.
func (s *JWTService) Parse(ctx context.Context, tokenStr string) (port.TokenClaims, error) {
	if err := ctx.Err(); err != nil {
		return port.TokenClaims{}, err
	}
	parsed, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil || !parsed.Valid {
		return port.TokenClaims{}, port.ErrInvalidToken
	}
	c, ok := parsed.Claims.(*jwtClaims)
	if !ok || c.Subject == "" {
		return port.TokenClaims{}, port.ErrInvalidToken
	}
	var exp time.Time
	if c.ExpiresAt != nil {
		exp = c.ExpiresAt.Time
	}
	roles := c.Roles
	if roles == nil {
		roles = []string{}
	}
	return port.TokenClaims{
		UserID:   c.Subject,
		Username: c.Username,
		Name:     c.Name,
		Roles:    roles,
		Expires:  exp,
	}, nil
}

var _ port.TokenService = (*JWTService)(nil)
