package auth_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

func TestPasswordBoundariesAndUpgrade(t *testing.T) {
	passwords, err := auth.NewPasswords()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name  string
		value string
		valid bool
	}{
		{name: "11 bytes", value: strings.Repeat("a", 11)},
		{name: "12 bytes", value: strings.Repeat("a", 12), valid: true},
		{name: "72 bytes", value: strings.Repeat("a", 72), valid: true},
		{name: "73 bytes", value: strings.Repeat("a", 73)},
		{name: "unicode 12 bytes", value: "密码密码密码密码", valid: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			hash, err := passwords.Hash(test.value)
			if (err == nil) != test.valid {
				t.Fatalf("Hash() error = %v, valid %v", err, test.valid)
			}
			if test.valid {
				matched, err := passwords.Compare(hash, test.value)
				if err != nil || !matched {
					t.Fatalf("Compare() = %v, %v", matched, err)
				}
			}
		})
	}
	lowCost, err := bcrypt.GenerateFromPassword([]byte("123456789012"), auth.PasswordCost-1)
	if err != nil {
		t.Fatal(err)
	}
	upgrade, err := auth.NeedsPasswordUpgrade(string(lowCost))
	if err != nil || !upgrade {
		t.Fatalf("NeedsPasswordUpgrade() = %v, %v", upgrade, err)
	}
}

func TestUnknownPasswordUsesDummyComparison(t *testing.T) {
	passwords, err := auth.NewPasswords()
	if err != nil {
		t.Fatal(err)
	}
	matched, err := passwords.Compare("", "123456789012")
	if err != nil || matched {
		t.Fatalf("Compare() = %v, %v", matched, err)
	}
}

func TestRandomIDPropagatesReaderFailure(t *testing.T) {
	if _, err := auth.RandomIDFrom(errorReader{}); err == nil {
		t.Fatal("RandomIDFrom() error = nil")
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("failed") }
