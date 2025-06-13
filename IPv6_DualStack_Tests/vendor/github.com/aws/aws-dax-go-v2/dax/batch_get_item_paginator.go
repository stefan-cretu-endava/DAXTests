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
	"reflect"

	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// BatchGetItemPaginator is a paginator for BatchGetItem
type BatchGetItemPaginator struct {
	options      dynamodb.BatchGetItemPaginatorOptions
	client       dynamodb.BatchGetItemAPIClient
	params       *dynamodb.BatchGetItemInput
	firstPage    bool
	requestItems map[string]types.KeysAndAttributes
	isTruncated  bool
}

// NewBatchGetItemPaginator returns a new BatchGetItemPaginator
func NewBatchGetItemPaginator(client dynamodb.BatchGetItemAPIClient, params *dynamodb.BatchGetItemInput, optFns ...func(*dynamodb.BatchGetItemPaginatorOptions)) *BatchGetItemPaginator {
	if params == nil {
		params = &dynamodb.BatchGetItemInput{}
	}

	options := dynamodb.BatchGetItemPaginatorOptions{}

	for _, fn := range optFns {
		fn(&options)
	}

	return &BatchGetItemPaginator{
		options:      options,
		client:       client,
		params:       params,
		firstPage:    true,
		requestItems: params.RequestItems,
	}
}

// HasMorePages returns a boolean indicating whether more pages are available
func (p *BatchGetItemPaginator) HasMorePages() bool {
	return p.firstPage || p.isTruncated
}

// NextPage retrieves the next BatchGetItem page.
func (p *BatchGetItemPaginator) NextPage(ctx context.Context, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error) {
	if !p.HasMorePages() {
		return nil, fmt.Errorf("no more pages available")
	}

	params := *p.params
	params.RequestItems = p.requestItems

	result, err := p.client.BatchGetItem(ctx, &params, optFns...)
	if err != nil {
		return nil, err
	}
	p.firstPage = false

	prevToken := p.requestItems
	p.isTruncated = len(result.UnprocessedKeys) != 0
	p.requestItems = nil
	if p.isTruncated {
		p.requestItems = result.UnprocessedKeys
	}

	if p.options.StopOnDuplicateToken &&
		prevToken != nil &&
		p.requestItems != nil &&
		DeepEqual(prevToken, p.requestItems) {
		p.isTruncated = false
	}

	return result, nil
}

// Source: https://github.com/aws/aws-sdk-go-v2/blob/78fa10aa9eaaa0851b0006145382ec0a0f4304c5/internal/awsutil/equal.go#L13
// DeepEqual returns if the two values are deeply equal like reflect.DeepEqual.
// In addition to this, this method will also dereference the input values if
// possible so the DeepEqual performed will not fail if one parameter is a
// pointer and the other is not.
//
// DeepEqual will not perform indirection of nested values of the input parameters.
func DeepEqual(a, b interface{}) bool {
	ra := reflect.Indirect(reflect.ValueOf(a))
	rb := reflect.Indirect(reflect.ValueOf(b))

	if raValid, rbValid := ra.IsValid(), rb.IsValid(); !raValid && !rbValid {
		// If the elements are both nil, and of the same type the are equal
		// If they are of different types they are not equal
		return reflect.TypeOf(a) == reflect.TypeOf(b)
	} else if raValid != rbValid {
		// Both values must be valid to be equal
		return false
	}

	// Special casing for strings as typed enumerations are string aliases
	// but are not deep equal.
	if ra.Kind() == reflect.String && rb.Kind() == reflect.String {
		return ra.String() == rb.String()
	}

	return reflect.DeepEqual(ra.Interface(), rb.Interface())
}
