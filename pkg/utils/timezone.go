package utils

import (
	"os"
	"sync"
	"time"
)

const (
	// SeoulTimezone is the canonical IANA timezone name for Korea.
	SeoulTimezone = "Asia/Seoul"
)

var (
	seoulLocOnce sync.Once
	seoulLoc     *time.Location
)

// SeoulLocation returns Asia/Seoul location with a fixed-offset fallback.
func SeoulLocation() *time.Location {
	seoulLocOnce.Do(func() {
		loc, err := time.LoadLocation(SeoulTimezone)
		if err != nil || loc == nil {
			// Fallback keeps service behavior stable even when tzdata is unavailable.
			seoulLoc = time.FixedZone("KST", 9*60*60)
			return
		}
		seoulLoc = loc
	})

	return seoulLoc
}

// SetProcessTimezoneToSeoul pins process-local timezone to Asia/Seoul.
func SetProcessTimezoneToSeoul() *time.Location {
	loc := SeoulLocation()
	time.Local = loc
	_ = os.Setenv("TZ", SeoulTimezone)
	return loc
}

// NowSeoul returns current time in Asia/Seoul.
func NowSeoul() time.Time {
	return time.Now().In(SeoulLocation())
}

// StartOfDaySeoul normalizes a time to 00:00:00 in Asia/Seoul.
func StartOfDaySeoul(t time.Time) time.Time {
	kst := t.In(SeoulLocation())
	return time.Date(kst.Year(), kst.Month(), kst.Day(), 0, 0, 0, 0, SeoulLocation())
}

// ParseInSeoul parses layout/value in Asia/Seoul.
func ParseInSeoul(layout, value string) (time.Time, error) {
	return time.ParseInLocation(layout, value, SeoulLocation())
}

// DateKeySeoul formats date as YYYY-MM-DD in Asia/Seoul.
func DateKeySeoul(t time.Time) string {
	return t.In(SeoulLocation()).Format("2006-01-02")
}
