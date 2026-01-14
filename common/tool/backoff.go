package tool

import (
	"time"

	"github.com/dromara/carbon/v2"
)

func CalculateNextTriggerTime(failureCount int64, expiry time.Duration) (time.Time, bool) {
	currentTime := time.Now()
	var nextTriggerTime time.Time
	if failureCount <= 30 {
		nextTriggerTime = currentTime.Add(expiry)
	} else if failureCount <= 60 {
		exponent := failureCount - 30
		backoff := expiry
		for i := int64(0); i < exponent; i++ {
			backoff *= 2
			if backoff >= 30*time.Minute {
				backoff = 30 * time.Minute
				break
			}
		}
		nextTriggerTime = currentTime.Add(backoff)
	} else {
		nextTriggerTime = currentTime.Add(30 * time.Minute)
	}

	if failureCount >= 500 {
		maxTriggerTime := currentTime.Add(7 * 24 * time.Hour)
		return maxTriggerTime, true
	}

	return nextTriggerTime, false
}

func CalculateNextTriggerTimeString(failureCount int64, defaultTimeout time.Duration) (string, bool) {
	nextTime, isExceeded := CalculateNextTriggerTime(failureCount, defaultTimeout)
	return carbon.CreateFromStdTime(nextTime).ToDateTimeString(), isExceeded
}
