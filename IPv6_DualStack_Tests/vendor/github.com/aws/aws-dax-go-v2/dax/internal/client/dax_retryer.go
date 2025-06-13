/*
  Copyright 2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.

  Licensed under the Apache License, Version 2.0 (the "License").
  You may not use this file except in compliance with the License.
  A copy of the License is located at

      http://www.apache.org/licenses/LICENSE-2.0

  or in the "license" file accompanying this file. This file is distributed
  on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
  express or implied. See the License for the specific language governing
  permissions and limitations under the License.
*/

package client

import (
	"math/rand"
	"time"
)

// DaxRetryer implements retry strategy with equal jitter backoff for throttled requests
type DaxRetryer struct {
	BaseThrottleDelay time.Duration
	MaxBackoffDelay   time.Duration
}

const (
	// DefaultBaseRetryDelay is base delay for throttled requests
	DefaultBaseRetryDelay = 70 * time.Millisecond
	// DefaultMaxBackoffDelay is max backoff delay for throttled requests
	DefaultMaxBackoffDelay = 20 * time.Second
)

func (r *DaxRetryer) setRetryerDefaults() {
	if r.BaseThrottleDelay == 0 {
		r.BaseThrottleDelay = DefaultBaseRetryDelay
	}
	if r.MaxBackoffDelay == 0 {
		r.MaxBackoffDelay = DefaultMaxBackoffDelay
	}
}

// RetryDelay returns the delay duration before retrying this request again
func (r DaxRetryer) RetryDelay(attempts int, err error) time.Duration {
	if IsThrottleError(err) {
		r.setRetryerDefaults()
		minDelay := time.Duration(1<<uint64(attempts)) * r.BaseThrottleDelay
		if minDelay > r.MaxBackoffDelay {
			minDelay = r.MaxBackoffDelay
		}
		jitter := time.Duration(rand.Intn(int(minDelay)/2 + 1))

		return minDelay/2 + jitter
	}
	return 0
}

// MaxAttempts returns the maximum number of retry attempts
func (r DaxRetryer) MaxAttempts() int {
	return 0 // You can adjust this value based on your requirements
}

// IsErrorRetryable returns if the error is daxError
// if code sequences correct any condition return a value other than unknown.
func (r DaxRetryer) IsErrorRetryable(err error) bool {
	if IsThrottleError(err) {
		return true
	}
	de, ok := err.(daxError)
	if !ok {
		return false
	}
	codes := de.CodeSequence()
	if len(codes) > 0 && (codes[0] == 1 || codes[0] == 2) {
		return true
	}
	// Error code [4.23.31.33] is for AuthenticationRequiredException
	if isAuthCRequiredException(codes) {
		return true
	}
	return false
}

// Error code [4.23.31.33] is for AuthenticationRequiredException
func isAuthCRequiredException(codes []int) bool {
	return len(codes) == 4 && codes[0] == 4 && codes[1] == 23 && codes[2] == 31 && codes[3] == 33
}

func isRetryable(o RequestOptions, err error) bool {
	return o.Retryer.IsErrorRetryable(err)
}
