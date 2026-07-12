package order

import "time"

type Clock func() time.Time

func UTCNow(clock Clock) time.Time {
	if clock == nil {
		clock = time.Now
	}
	return clock().UTC()
}

func FormatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}
