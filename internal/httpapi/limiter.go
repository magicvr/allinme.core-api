package httpapi

import (
	"math"
	"net"
	"net/netip"
	"sync"
	"time"
)

const (
	loginWindow        = time.Minute
	loginFailureLimit  = 5
	loginGlobalBuckets = 4096
	loginIPBuckets     = 32
)

type LimiterClock func() time.Time

type LoginLimiter struct {
	mutex   sync.Mutex
	clock   LimiterClock
	buckets map[loginKey]*loginBucket
}

type loginKey struct{ ip, username string }

type loginBucket struct {
	failures   int
	windowEnds time.Time
	lastAccess time.Time
}

func NewLoginLimiter(clock LimiterClock) *LoginLimiter {
	if clock == nil {
		clock = time.Now
	}
	return &LoginLimiter{clock: clock, buckets: make(map[loginKey]*loginBucket)}
}

func ClientIP(remoteAddress string) string {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		return "unknown"
	}
	address, err := netip.ParseAddr(host)
	if err != nil {
		return "unknown"
	}
	return address.Unmap().String()
}

func (limiter *LoginLimiter) Check(ip, username string) (bool, int) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()
	now := limiter.clock()
	limiter.cleanup(now)
	key := loginKey{ip: ip, username: username}
	if bucket, ok := limiter.buckets[key]; ok {
		bucket.lastAccess = now
		if bucket.failures >= loginFailureLimit {
			return false, retryAfter(now, bucket.windowEnds)
		}
		return true, 0
	}
	ipCount := 0
	for existing := range limiter.buckets {
		if existing.ip == ip {
			ipCount++
		}
	}
	if ipCount >= loginIPBuckets {
		return false, 1
	}
	if len(limiter.buckets) >= loginGlobalBuckets && !limiter.evictOldestUnrestricted() {
		return false, 1
	}
	limiter.buckets[key] = &loginBucket{windowEnds: now.Add(loginWindow), lastAccess: now}
	return true, 0
}

func (limiter *LoginLimiter) Failure(ip, username string) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()
	now := limiter.clock()
	key := loginKey{ip: ip, username: username}
	bucket, ok := limiter.buckets[key]
	if !ok || !now.Before(bucket.windowEnds) {
		return
	}
	bucket.failures++
	bucket.lastAccess = now
}

func (limiter *LoginLimiter) Success(ip, username string) {
	limiter.mutex.Lock()
	delete(limiter.buckets, loginKey{ip: ip, username: username})
	limiter.mutex.Unlock()
}

func (limiter *LoginLimiter) Cancel(ip, username string) {
	limiter.mutex.Lock()
	key := loginKey{ip: ip, username: username}
	if bucket, ok := limiter.buckets[key]; ok && bucket.failures == 0 {
		delete(limiter.buckets, key)
	}
	limiter.mutex.Unlock()
}

func (limiter *LoginLimiter) cleanup(now time.Time) {
	for key, bucket := range limiter.buckets {
		if !now.Before(bucket.windowEnds) {
			delete(limiter.buckets, key)
		}
	}
}

func (limiter *LoginLimiter) evictOldestUnrestricted() bool {
	var oldest loginKey
	var oldestTime time.Time
	found := false
	for key, bucket := range limiter.buckets {
		if bucket.failures >= loginFailureLimit {
			continue
		}
		if !found || bucket.lastAccess.Before(oldestTime) {
			oldest, oldestTime, found = key, bucket.lastAccess, true
		}
	}
	if found {
		delete(limiter.buckets, oldest)
	}
	return found
}

func retryAfter(now, windowEnds time.Time) int {
	seconds := int(math.Ceil(windowEnds.Sub(now).Seconds()))
	if seconds < 1 {
		return 1
	}
	return seconds
}
