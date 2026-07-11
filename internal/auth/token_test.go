package auth_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/magicvr/allinme.core-api/internal/auth"
)

func TestTokensIssueAndParseStrictClaims(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	key := []byte("12345678901234567890123456789012")
	tokens, err := auth.NewTokens(key, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	encoded, expiresAt, err := tokens.Issue("user-1", auth.RoleViewer, "token-1")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := tokens.Parse(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "user-1" || claims.ID != "token-1" || claims.Role != auth.RoleViewer || !expiresAt.Equal(now.Add(auth.TokenTTL)) {
		t.Fatalf("claims = %+v, expiresAt = %v", claims, expiresAt)
	}

	for _, mutate := range []func(*auth.Claims){
		func(claims *auth.Claims) { claims.Subject = "" },
		func(claims *auth.Claims) { claims.ID = "" },
		func(claims *auth.Claims) { claims.IssuedAt = nil },
		func(claims *auth.Claims) { claims.Role = "owner" },
	} {
		claims := auth.Claims{Role: auth.RoleViewer, RegisteredClaims: jwt.RegisteredClaims{
			Issuer: auth.TokenIssuer, Subject: "user-1", Audience: jwt.ClaimStrings{auth.TokenAudience},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)), IssuedAt: jwt.NewNumericDate(now), ID: "token-1",
		}}
		mutate(&claims)
		bad, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tokens.Parse(bad); err == nil {
			t.Fatal("Parse() accepted missing or invalid claim")
		}
	}
}

func TestTokensRejectAlgorithmAndTimeBoundaries(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	key := []byte("12345678901234567890123456789012")
	clock := now
	tokens, err := auth.NewTokens(key, func() time.Time { return clock })
	if err != nil {
		t.Fatal(err)
	}
	encoded, _, err := tokens.Issue("user-1", auth.RoleAdmin, "token-1")
	if err != nil {
		t.Fatal(err)
	}
	clock = now.Add(auth.TokenTTL + auth.ClockLeeway - time.Nanosecond)
	if _, err := tokens.Parse(encoded); err != nil {
		t.Fatalf("Parse() before expiry boundary error = %v", err)
	}
	clock = now.Add(auth.TokenTTL + auth.ClockLeeway)
	if _, err := tokens.Parse(encoded); err == nil {
		t.Fatal("Parse() accepted token at expiry boundary")
	}

	claims := auth.Claims{Role: auth.RoleAdmin, RegisteredClaims: jwt.RegisteredClaims{
		Issuer: auth.TokenIssuer, Subject: "user-1", Audience: jwt.ClaimStrings{auth.TokenAudience},
		ExpiresAt: jwt.NewNumericDate(clock.Add(time.Minute)), IssuedAt: jwt.NewNumericDate(clock), ID: "token-2",
	}}
	wrongAlgorithm, err := jwt.NewWithClaims(jwt.SigningMethodHS384, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tokens.Parse(wrongAlgorithm); err == nil {
		t.Fatal("Parse() accepted HS384")
	}
}

func TestTokensRejectIssuerAudienceTamperingAndFutureIssuedAt(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	key := []byte("12345678901234567890123456789012")
	tokens, err := auth.NewTokens(key, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	base := func() auth.Claims {
		return auth.Claims{Role: auth.RoleOperator, RegisteredClaims: jwt.RegisteredClaims{
			Issuer: auth.TokenIssuer, Subject: "user-1", Audience: jwt.ClaimStrings{auth.TokenAudience},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)), IssuedAt: jwt.NewNumericDate(now), ID: "token-1",
		}}
	}
	for name, mutate := range map[string]func(*auth.Claims){
		"issuer":   func(claims *auth.Claims) { claims.Issuer = "other" },
		"audience": func(claims *auth.Claims) { claims.Audience = jwt.ClaimStrings{"other"} },
		"future iat": func(claims *auth.Claims) {
			claims.IssuedAt = jwt.NewNumericDate(now.Add(auth.ClockLeeway + time.Second))
		},
	} {
		t.Run(name, func(t *testing.T) {
			claims := base()
			mutate(&claims)
			encoded, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(key)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := tokens.Parse(encoded); err == nil {
				t.Fatal("Parse() accepted invalid token")
			}
		})
	}
	encoded, _, err := tokens.Issue("user-1", auth.RoleOperator, "token-1")
	if err != nil {
		t.Fatal(err)
	}
	last := encoded[len(encoded)-1]
	replacement := byte('a')
	if last == replacement {
		replacement = 'b'
	}
	if _, err := tokens.Parse(encoded[:len(encoded)-1] + string(replacement)); err == nil {
		t.Fatal("Parse() accepted tampered signature")
	}
}
