package tool

import (
	"time"

	"github.com/dromara/carbon/v2"
)

// CalculateNextTriggerTime calculates the next trigger time for a failed task based on the failure count
// Rules:
// 1. First 30 failures: Use default timeout time
// 2. Next 30 failures: Use exponential backoff with default time as base
// 3. After 60 failures: Check every 30 minutes
// 4. Maximum detection time: 7 days from first failure (estimated by failure count)
// Returns (nextTriggerTime, isExceededMaxTime)
func CalculateNextTriggerTime(failureCount int64, defaultTimeout time.Duration) (time.Time, bool) {
	currentTime := time.Now()

	// Calculate next trigger time based on failure count
	var nextTriggerTime time.Time

	if failureCount <= 30 {
		// First 30 failures: Use default timeout time
		nextTriggerTime = currentTime.Add(defaultTimeout)
	} else if failureCount <= 60 {
		// Next 30 failures: Exponential backoff (2^n * defaultTimeout)
		// Cap at 30 minutes
		exponent := failureCount - 30
		backoff := defaultTimeout
		for i := int64(0); i < exponent; i++ {
			backoff *= 2
			// Cap at 30 minutes
			if backoff >= 30*time.Minute {
				backoff = 30 * time.Minute
				break
			}
		}
		nextTriggerTime = currentTime.Add(backoff)
	} else {
		// After 60 failures: Check every 30 minutes
		nextTriggerTime = currentTime.Add(30 * time.Minute)
	}

	// Maximum detection time: 7 days from first failure (estimated by failure count)
	// Based on calculation: 500 failures would take approximately 9.5 days total
	// So we use failure count >= 500 as the threshold for exceeding 7 days
	if failureCount >= 500 {
		// If failure count exceeds 500, it's been more than 7 days since first failure
		maxTriggerTime := currentTime.Add(7 * 24 * time.Hour)
		return maxTriggerTime, true
	}

	// Within 7 days, return calculated time and mark as not exceeded
	return nextTriggerTime, false
}

// CalculateNextTriggerTimeString calculates the next trigger time as a string in carbon format
// Returns (nextTriggerTimeString, isExceededMaxTime)
func CalculateNextTriggerTimeString(failureCount int64, defaultTimeout time.Duration) (string, bool) {
	nextTime, isExceeded := CalculateNextTriggerTime(failureCount, defaultTimeout)
	return carbon.CreateFromStdTime(nextTime).ToDateTimeString(), isExceeded
}
