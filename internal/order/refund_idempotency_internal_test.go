package order

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestNormalizedRefundDigestIgnoresJSONLayoutAndReasonOuterWhitespace(t *testing.T) {
	raw := []string{
		`{"amount":100,"reason":" customer request ","orderVersion":3}`,
		"{\n  \"orderVersion\": 3, \"reason\": \"customer request\", \"amount\": 100\n}",
	}
	var digests [][32]byte
	for _, input := range raw {
		var command RefundRequestCommand
		if err := json.Unmarshal([]byte(input), &command); err != nil {
			t.Fatal(err)
		}
		normalized, err := NormalizeRefundRequest(command)
		if err != nil {
			t.Fatal(err)
		}
		digest, err := normalizedRefundDigest("ord_00000000000000000000000000000001", normalized)
		if err != nil {
			t.Fatal(err)
		}
		digests = append(digests, digest)
	}
	if !bytes.Equal(digests[0][:], digests[1][:]) {
		t.Fatalf("equivalent digests differ: %x / %x", digests[0], digests[1])
	}
	base, _ := NormalizeRefundRequest(RefundRequestCommand{Amount: 100, Reason: "customer request", OrderVersion: 3})
	baseDigest, _ := normalizedRefundDigest("ord_00000000000000000000000000000001", base)
	differences := []RefundRequestCommand{
		{Amount: 101, Reason: "customer request", OrderVersion: 3},
		{Amount: 100, Reason: "Customer request", OrderVersion: 3},
		{Amount: 100, Reason: "customer\nrequest", OrderVersion: 3},
		{Amount: 100, Reason: "customer request", OrderVersion: 4},
	}
	for _, command := range differences {
		normalized, err := NormalizeRefundRequest(command)
		if err != nil {
			t.Fatal(err)
		}
		digest, err := normalizedRefundDigest("ord_00000000000000000000000000000001", normalized)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(digest[:], baseDigest[:]) {
			t.Errorf("fact difference has base digest: %+v", command)
		}
	}
	otherOrderDigest, err := normalizedRefundDigest("ord_00000000000000000000000000000002", base)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(otherOrderDigest[:], baseDigest[:]) {
		t.Fatal("different order ID has base digest")
	}
}
