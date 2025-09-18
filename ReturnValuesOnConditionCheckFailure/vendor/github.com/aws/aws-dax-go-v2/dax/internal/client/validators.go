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
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	smithy "github.com/aws/smithy-go"
)

//Source: https://github.com/aws/aws-sdk-go-v2/blob/main/service/dynamodb/validators.go

func validateConditionCheck(v *types.ConditionCheck) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "ConditionCheck"}
	if v.Key == nil {
		invalidParams.Add(smithy.NewErrParamRequired("Key"))
	}
	if v.TableName == nil {
		invalidParams.Add(smithy.NewErrParamRequired("TableName"))
	}
	if v.ConditionExpression == nil {
		invalidParams.Add(smithy.NewErrParamRequired("ConditionExpression"))
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func validatePut(v *types.Put) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "Put"}
	if v.Item == nil {
		invalidParams.Add(smithy.NewErrParamRequired("Item"))
	}
	if v.TableName == nil {
		invalidParams.Add(smithy.NewErrParamRequired("TableName"))
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func validateDelete(v *types.Delete) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "Delete"}
	if v.Key == nil {
		invalidParams.Add(smithy.NewErrParamRequired("Key"))
	}
	if v.TableName == nil {
		invalidParams.Add(smithy.NewErrParamRequired("TableName"))
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func validateUpdate(v *types.Update) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "Update"}
	if v.Key == nil {
		invalidParams.Add(smithy.NewErrParamRequired("Key"))
	}
	if v.UpdateExpression == nil {
		invalidParams.Add(smithy.NewErrParamRequired("UpdateExpression"))
	}
	if v.TableName == nil {
		invalidParams.Add(smithy.NewErrParamRequired("TableName"))
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func validateTransactWriteItem(v *types.TransactWriteItem) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "TransactWriteItem"}
	if v.ConditionCheck != nil {
		if err := validateConditionCheck(v.ConditionCheck); err != nil {
			invalidParams.AddNested("ConditionCheck", err.(smithy.InvalidParamsError))
		}
	}
	if v.Put != nil {
		if err := validatePut(v.Put); err != nil {
			invalidParams.AddNested("Put", err.(smithy.InvalidParamsError))
		}
	}
	if v.Delete != nil {
		if err := validateDelete(v.Delete); err != nil {
			invalidParams.AddNested("Delete", err.(smithy.InvalidParamsError))
		}
	}
	if v.Update != nil {
		if err := validateUpdate(v.Update); err != nil {
			invalidParams.AddNested("Update", err.(smithy.InvalidParamsError))
		}
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func validateTransactWriteItemList(v []types.TransactWriteItem) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "TransactWriteItemList"}
	for i := range v {
		if err := validateTransactWriteItem(&v[i]); err != nil {
			invalidParams.AddNested(fmt.Sprintf("[%d]", i), err.(smithy.InvalidParamsError))
		}
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func validateKeysAndAttributes(v *types.KeysAndAttributes) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "KeysAndAttributes"}
	if v.Keys == nil {
		invalidParams.Add(smithy.NewErrParamRequired("Keys"))
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func validateBatchGetRequestMap(v map[string]types.KeysAndAttributes) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "BatchGetRequestMap"}
	for key := range v {
		value := v[key]
		if err := validateKeysAndAttributes(&value); err != nil {
			invalidParams.AddNested(fmt.Sprintf("[%q]", key), err.(smithy.InvalidParamsError))
		}
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func validateGet(v *types.Get) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "Get"}
	if v.Key == nil {
		invalidParams.Add(smithy.NewErrParamRequired("Key"))
	}
	if v.TableName == nil {
		invalidParams.Add(smithy.NewErrParamRequired("TableName"))
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func validateTransactGetItem(v *types.TransactGetItem) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "TransactGetItem"}
	if v.Get == nil {
		invalidParams.Add(smithy.NewErrParamRequired("Get"))
	} else if v.Get != nil {
		if err := validateGet(v.Get); err != nil {
			invalidParams.AddNested("Get", err.(smithy.InvalidParamsError))
		}
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func validateTransactGetItemList(v []types.TransactGetItem) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "TransactGetItemList"}
	for i := range v {
		if err := validateTransactGetItem(&v[i]); err != nil {
			invalidParams.AddNested(fmt.Sprintf("[%d]", i), err.(smithy.InvalidParamsError))
		}
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func validatePutRequest(v *types.PutRequest) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "PutRequest"}
	if v.Item == nil {
		invalidParams.Add(smithy.NewErrParamRequired("Item"))
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func validateDeleteRequest(v *types.DeleteRequest) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "DeleteRequest"}
	if v.Key == nil {
		invalidParams.Add(smithy.NewErrParamRequired("Key"))
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func validateWriteRequest(v *types.WriteRequest) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "WriteRequest"}
	if v.PutRequest != nil {
		if err := validatePutRequest(v.PutRequest); err != nil {
			invalidParams.AddNested("PutRequest", err.(smithy.InvalidParamsError))
		}
	}
	if v.DeleteRequest != nil {
		if err := validateDeleteRequest(v.DeleteRequest); err != nil {
			invalidParams.AddNested("DeleteRequest", err.(smithy.InvalidParamsError))
		}
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func validateWriteRequests(v []types.WriteRequest) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "WriteRequests"}
	for i := range v {
		if err := validateWriteRequest(&v[i]); err != nil {
			invalidParams.AddNested(fmt.Sprintf("[%d]", i), err.(smithy.InvalidParamsError))
		}
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func validateBatchWriteItemRequestMap(v map[string][]types.WriteRequest) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "BatchWriteItemRequestMap"}
	for key := range v {
		if err := validateWriteRequests(v[key]); err != nil {
			invalidParams.AddNested(fmt.Sprintf("[%q]", key), err.(smithy.InvalidParamsError))
		}
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func validateCondition(v *types.Condition) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "Condition"}
	if len(v.ComparisonOperator) == 0 {
		invalidParams.Add(smithy.NewErrParamRequired("ComparisonOperator"))
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func validateKeyConditions(v map[string]types.Condition) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "KeyConditions"}
	for key := range v {
		value := v[key]
		if err := validateCondition(&value); err != nil {
			invalidParams.AddNested(fmt.Sprintf("[%q]", key), err.(smithy.InvalidParamsError))
		}
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func validateFilterConditionMap(v map[string]types.Condition) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "FilterConditionMap"}
	for key := range v {
		value := v[key]
		if err := validateCondition(&value); err != nil {
			invalidParams.AddNested(fmt.Sprintf("[%q]", key), err.(smithy.InvalidParamsError))
		}
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func ValidateOpPutItemInput(v *dynamodb.PutItemInput) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "PutItemInput"}
	if v.TableName == nil {
		invalidParams.Add(smithy.NewErrParamRequired("TableName"))
	}
	if v.Item == nil {
		invalidParams.Add(smithy.NewErrParamRequired("Item"))
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func ValidateOpDeleteItemInput(v *dynamodb.DeleteItemInput) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "DeleteItemInput"}
	if v.TableName == nil {
		invalidParams.Add(smithy.NewErrParamRequired("TableName"))
	}
	if v.Key == nil {
		invalidParams.Add(smithy.NewErrParamRequired("Key"))
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func ValidateOpUpdateItemInput(v *dynamodb.UpdateItemInput) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "UpdateItemInput"}
	if v.TableName == nil {
		invalidParams.Add(smithy.NewErrParamRequired("TableName"))
	}
	if v.Key == nil {
		invalidParams.Add(smithy.NewErrParamRequired("Key"))
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func ValidateOpGetItemInput(v *dynamodb.GetItemInput) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "GetItemInput"}
	if v.TableName == nil {
		invalidParams.Add(smithy.NewErrParamRequired("TableName"))
	}
	if v.Key == nil {
		invalidParams.Add(smithy.NewErrParamRequired("Key"))
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func ValidateOpScanInput(v *dynamodb.ScanInput) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "ScanInput"}
	if v.TableName == nil {
		invalidParams.Add(smithy.NewErrParamRequired("TableName"))
	}
	if v.ScanFilter != nil {
		if err := validateFilterConditionMap(v.ScanFilter); err != nil {
			invalidParams.AddNested("ScanFilter", err.(smithy.InvalidParamsError))
		}
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func ValidateOpQueryInput(v *dynamodb.QueryInput) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "QueryInput"}
	if v.TableName == nil {
		invalidParams.Add(smithy.NewErrParamRequired("TableName"))
	}
	if v.KeyConditions != nil {
		if err := validateKeyConditions(v.KeyConditions); err != nil {
			invalidParams.AddNested("KeyConditions", err.(smithy.InvalidParamsError))
		}
	}
	if v.QueryFilter != nil {
		if err := validateFilterConditionMap(v.QueryFilter); err != nil {
			invalidParams.AddNested("QueryFilter", err.(smithy.InvalidParamsError))
		}
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func ValidateOpBatchWriteItemInput(v *dynamodb.BatchWriteItemInput) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "BatchWriteItemInput"}
	if v.RequestItems == nil {
		invalidParams.Add(smithy.NewErrParamRequired("RequestItems"))
	} else if v.RequestItems != nil {
		if err := validateBatchWriteItemRequestMap(v.RequestItems); err != nil {
			invalidParams.AddNested("RequestItems", err.(smithy.InvalidParamsError))
		}
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func ValidateOpBatchGetItemInput(v *dynamodb.BatchGetItemInput) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "BatchGetItemInput"}
	if v.RequestItems == nil {
		invalidParams.Add(smithy.NewErrParamRequired("RequestItems"))
	} else if v.RequestItems != nil {
		if err := validateBatchGetRequestMap(v.RequestItems); err != nil {
			invalidParams.AddNested("RequestItems", err.(smithy.InvalidParamsError))
		}
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func ValidateOpTransactWriteItemsInput(v *dynamodb.TransactWriteItemsInput) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "TransactWriteItemsInput"}
	if v.TransactItems == nil {
		invalidParams.Add(smithy.NewErrParamRequired("TransactItems"))
	} else if v.TransactItems != nil {
		if err := validateTransactWriteItemList(v.TransactItems); err != nil {
			invalidParams.AddNested("TransactItems", err.(smithy.InvalidParamsError))
		}
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}

func ValidateOpTransactGetItemsInput(v *dynamodb.TransactGetItemsInput) error {
	if v == nil {
		return nil
	}
	invalidParams := smithy.InvalidParamsError{Context: "TransactGetItemsInput"}
	if v.TransactItems == nil {
		invalidParams.Add(smithy.NewErrParamRequired("TransactItems"))
	} else if v.TransactItems != nil {
		if err := validateTransactGetItemList(v.TransactItems); err != nil {
			invalidParams.AddNested("TransactItems", err.(smithy.InvalidParamsError))
		}
	}
	if invalidParams.Len() > 0 {
		return invalidParams
	} else {
		return nil
	}
}
