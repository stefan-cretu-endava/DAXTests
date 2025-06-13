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

package dax

import (
	"context"

	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// QueryPaginator is a paginator for Query
type QueryPaginator struct {
	options   dynamodb.QueryPaginatorOptions
	client    dynamodb.QueryAPIClient
	params    *dynamodb.QueryInput
	nextToken map[string]types.AttributeValue
	firstPage bool
}

// NewQueryPaginator returns a new QueryPaginator
func NewQueryPaginator(client dynamodb.QueryAPIClient, params *dynamodb.QueryInput, optFns ...func(*dynamodb.QueryPaginatorOptions)) *QueryPaginator {
	if params == nil {
		params = &dynamodb.QueryInput{}
	}

	options := dynamodb.QueryPaginatorOptions{}
	if params.Limit != nil {
		options.Limit = *params.Limit
	}

	for _, fn := range optFns {
		fn(&options)
	}

	return &QueryPaginator{
		options:   options,
		client:    client,
		params:    params,
		firstPage: true,
		nextToken: params.ExclusiveStartKey,
	}
}

// HasMorePages returns a boolean indicating whether more pages are available
func (p *QueryPaginator) HasMorePages() bool {
	return p.firstPage || p.nextToken != nil
}

// NextPage retrieves the next Query page.
func (p *QueryPaginator) NextPage(ctx context.Context, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
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
	result, err := p.client.Query(ctx, &params, optFns...)
	if err != nil {
		return nil, err
	}
	p.firstPage = false

	prevToken := p.nextToken
	p.nextToken = result.LastEvaluatedKey

	_ = prevToken

	return result, nil
}