package httpapi

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestLoginLimiterPerIPCapacityAndExpiry(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	limiter := NewLoginLimiter(func() time.Time { return now })
	for index := 0; index < loginIPBuckets; index++ {
		if allowed, _ := limiter.Check("192.0.2.1", fmt.Sprintf("user-%d", index)); !allowed {
			t.Fatalf("key %d rejected", index)
		}
	}
	if allowed, retry := limiter.Check("192.0.2.1", "overflow"); allowed || retry != 1 {
		t.Fatalf("overflow = %v, %d", allowed, retry)
	}
	if allowed, _ := limiter.Check("192.0.2.2", "other"); !allowed {
		t.Fatal("other IP rejected")
	}
	now = now.Add(loginWindow)
	if allowed, _ := limiter.Check("192.0.2.1", "overflow"); !allowed {
		t.Fatal("expired capacity was not released")
	}
}

func TestLoginLimiterProtectsRestrictedBucketsAtGlobalCapacity(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	limiter := NewLoginLimiter(func() time.Time { return now })
	for index := 0; index < loginGlobalBuckets; index++ {
		ip := fmt.Sprintf("198.%d.%d.%d", index/62500+18, (index/250)%250, index%250+1)
		username := fmt.Sprintf("user-%d", index)
		if allowed, _ := limiter.Check(ip, username); !allowed {
			t.Fatalf("key %d rejected", index)
		}
		for failure := 0; failure < loginFailureLimit; failure++ {
			limiter.Failure(ip, username)
		}
	}
	if allowed, _ := limiter.Check("203.0.113.1", "new"); allowed {
		t.Fatal("new key accepted while all buckets protected")
	}
	if len(limiter.buckets) != loginGlobalBuckets {
		t.Fatalf("bucket count = %d", len(limiter.buckets))
	}
}

func TestLoginLimiterConcurrentFailures(t *testing.T) {
	limiter := NewLoginLimiter(nil)
	if allowed, _ := limiter.Check("192.0.2.1", "viewer"); !allowed {
		t.Fatal("initial check rejected")
	}
	var wait sync.WaitGroup
	for index := 0; index < 100; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			limiter.Failure("192.0.2.1", "viewer")
		}()
	}
	wait.Wait()
	if allowed, _ := limiter.Check("192.0.2.1", "viewer"); allowed {
		t.Fatal("concurrent failures did not restrict key")
	}
}

func TestLoginLimiterFailureDoesNotReinsertEvictedBucket(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	limiter := NewLoginLimiter(func() time.Time { return now })
	if allowed, _ := limiter.Check("192.0.2.1", "evicted"); !allowed {
		t.Fatal("initial key rejected")
	}
	delete(limiter.buckets, loginKey{ip: "192.0.2.1", username: "evicted"})
	limiter.Failure("192.0.2.1", "evicted")
	if len(limiter.buckets) != 0 {
		t.Fatalf("failure reinserted bucket, count = %d", len(limiter.buckets))
	}
}

func TestLoginLimiterCancelOnlyRemovesZeroFailureBucket(t *testing.T) {
	limiter := NewLoginLimiter(nil)
	limiter.Check("192.0.2.1", "new")
	limiter.Cancel("192.0.2.1", "new")
	if len(limiter.buckets) != 0 {
		t.Fatal("cancel retained zero-failure bucket")
	}
	limiter.Check("192.0.2.1", "failed")
	limiter.Failure("192.0.2.1", "failed")
	limiter.Cancel("192.0.2.1", "failed")
	if len(limiter.buckets) != 1 {
		t.Fatal("cancel removed prior authentication failure")
	}
}
