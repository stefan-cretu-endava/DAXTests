/*
  Copyright 2024 Amazon.com, Inc. or its affiliates. All Rights Reserved.

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
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-dax-go-v2/dax/utils"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/smithy-go/middleware"
)

type RequestOptions struct {
	dynamodb.Options
	LogLevel   utils.LogLevelType
	RetryDelay time.Duration
	Context    context.Context
	//Retryer implements equal jitter backoff stratergy for throttled requests
	Retryer DaxRetryer
}

// rejectCustomMiddleware checks if APIOptions are present and returns an error if they are.
// It's used to explicitly prevent custom middleware usage in the DAX client.
func RejectCustomMiddleware(apiOptions []func(*middleware.Stack) error) error {
	if len(apiOptions) > 0 {
		return fmt.Errorf("custom middleware through APIOptions is not supported in DAX client")
	}
	return nil
}
