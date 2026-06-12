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

package dax

import (
	"context"

	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// ScanPaginator is a paginator for Scan
type ScanPaginator struct {
	options   dynamodb.ScanPaginatorOptions
	client    dynamodb.ScanAPIClient
	params    *dynamodb.ScanInput
	nextToken map[string]types.AttributeValue
	firstPage bool
}

// NewScanPaginator returns a new ScanPaginator
func NewScanPaginator(client dynamodb.ScanAPIClient, params *dynamodb.ScanInput, optFns ...func(*dynamodb.ScanPaginatorOptions)) *ScanPaginator {
	if params == nil {
		params = &dynamodb.ScanInput{}
	}

	options := dynamodb.ScanPaginatorOptions{}
	if params.Limit != nil {
		options.Limit = *params.Limit
	}

	for _, fn := range optFns {
		fn(&options)
	}

	return &ScanPaginator{
		options:   options,
		client:    client,
		params:    params,
		firstPage: true,
		nextToken: params.ExclusiveStartKey,
	}
}

// HasMorePages returns a boolean indicating whether more pages are available
func (p *ScanPaginator) HasMorePages() bool {
	return p.firstPage || p.nextToken != nil
}

// NextPage retrieves the next Scan page.
func (p *ScanPaginator) NextPage(ctx context.Context, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
	if !p.HasMorePages() {
		return nil, fmt.Errorf("no more pages available")
	}

	params := *p.params
	params.ExclusiveStartKey = p.nextToken

	var limit *int32
	if p.options.Limit > 0 {
		limit = &p.options.Limit
	}
	params.Limit = limit

	result, err := p.client.Scan(ctx, &params, optFns...)
	if err != nil {
		return nil, err
	}
	p.firstPage = false

	prevToken := p.nextToken
	p.nextToken = result.LastEvaluatedKey

	_ = prevToken

	return result, nil
}
