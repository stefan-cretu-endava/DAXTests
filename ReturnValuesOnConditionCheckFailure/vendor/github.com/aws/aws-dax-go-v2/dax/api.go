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
	"errors"
	"io"
	"log"

	"github.com/aws/aws-dax-go-v2/dax/internal/client"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoDBAPI is compatible to aws-sdk-go-v2/service/dynamodb.Client
type DynamoDBAPI interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	BatchWriteItem(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error)
	BatchGetItem(ctx context.Context, params *dynamodb.BatchGetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error)
	TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
	TransactGetItems(ctx context.Context, params *dynamodb.TransactGetItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactGetItemsOutput, error)

	BatchExecuteStatement(ctx context.Context, params *dynamodb.BatchExecuteStatementInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchExecuteStatementOutput, error)
	CreateBackup(ctx context.Context, params *dynamodb.CreateBackupInput, optFns ...func(*dynamodb.Options)) (*dynamodb.CreateBackupOutput, error)
	CreateGlobalTable(ctx context.Context, params *dynamodb.CreateGlobalTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.CreateGlobalTableOutput, error)
	CreateTable(ctx context.Context, params *dynamodb.CreateTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.CreateTableOutput, error)
	DeleteBackup(ctx context.Context, params *dynamodb.DeleteBackupInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteBackupOutput, error)
	DeleteTable(ctx context.Context, params *dynamodb.DeleteTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteTableOutput, error)
	DescribeBackup(ctx context.Context, params *dynamodb.DescribeBackupInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeBackupOutput, error)
	DescribeContinuousBackups(ctx context.Context, params *dynamodb.DescribeContinuousBackupsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeContinuousBackupsOutput, error)
	DescribeContributorInsights(ctx context.Context, params *dynamodb.DescribeContributorInsightsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeContributorInsightsOutput, error)
	DescribeEndpoints(ctx context.Context, params *dynamodb.DescribeEndpointsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeEndpointsOutput, error)
	DescribeExport(ctx context.Context, params *dynamodb.DescribeExportInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeExportOutput, error)
	DescribeGlobalTable(ctx context.Context, params *dynamodb.DescribeGlobalTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeGlobalTableOutput, error)
	DescribeGlobalTableSettings(ctx context.Context, params *dynamodb.DescribeGlobalTableSettingsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeGlobalTableSettingsOutput, error)
	DescribeImport(ctx context.Context, params *dynamodb.DescribeImportInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeImportOutput, error)
	DescribeKinesisStreamingDestination(ctx context.Context, params *dynamodb.DescribeKinesisStreamingDestinationInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeKinesisStreamingDestinationOutput, error)
	DescribeLimits(ctx context.Context, params *dynamodb.DescribeLimitsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeLimitsOutput, error)
	DescribeTable(ctx context.Context, params *dynamodb.DescribeTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error)
	DescribeTableReplicaAutoScaling(ctx context.Context, params *dynamodb.DescribeTableReplicaAutoScalingInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeTableReplicaAutoScalingOutput, error)
	DescribeTimeToLive(ctx context.Context, params *dynamodb.DescribeTimeToLiveInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeTimeToLiveOutput, error)
	DisableKinesisStreamingDestination(ctx context.Context, params *dynamodb.DisableKinesisStreamingDestinationInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DisableKinesisStreamingDestinationOutput, error)
	EnableKinesisStreamingDestination(ctx context.Context, params *dynamodb.EnableKinesisStreamingDestinationInput, optFns ...func(*dynamodb.Options)) (*dynamodb.EnableKinesisStreamingDestinationOutput, error)
	ExecuteStatement(ctx context.Context, params *dynamodb.ExecuteStatementInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ExecuteStatementOutput, error)
	ExecuteTransaction(ctx context.Context, params *dynamodb.ExecuteTransactionInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ExecuteTransactionOutput, error)
	ExportTableToPointInTime(ctx context.Context, params *dynamodb.ExportTableToPointInTimeInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ExportTableToPointInTimeOutput, error)
	ImportTable(ctx context.Context, params *dynamodb.ImportTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ImportTableOutput, error)
	ListBackups(ctx context.Context, params *dynamodb.ListBackupsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListBackupsOutput, error)
	ListContributorInsights(ctx context.Context, params *dynamodb.ListContributorInsightsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListContributorInsightsOutput, error)
	ListExports(ctx context.Context, params *dynamodb.ListExportsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListExportsOutput, error)
	ListGlobalTables(ctx context.Context, params *dynamodb.ListGlobalTablesInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListGlobalTablesOutput, error)
	ListImports(ctx context.Context, params *dynamodb.ListImportsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListImportsOutput, error)
	ListTables(ctx context.Context, params *dynamodb.ListTablesInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListTablesOutput, error)
	ListTagsOfResource(ctx context.Context, params *dynamodb.ListTagsOfResourceInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListTagsOfResourceOutput, error)
	RestoreTableFromBackup(ctx context.Context, params *dynamodb.RestoreTableFromBackupInput, optFns ...func(*dynamodb.Options)) (*dynamodb.RestoreTableFromBackupOutput, error)
	RestoreTableToPointInTime(ctx context.Context, params *dynamodb.RestoreTableToPointInTimeInput, optFns ...func(*dynamodb.Options)) (*dynamodb.RestoreTableToPointInTimeOutput, error)
	TagResource(ctx context.Context, params *dynamodb.TagResourceInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TagResourceOutput, error)
	UntagResource(ctx context.Context, params *dynamodb.UntagResourceInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UntagResourceOutput, error)
	UpdateContinuousBackups(ctx context.Context, params *dynamodb.UpdateContinuousBackupsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateContinuousBackupsOutput, error)
	UpdateContributorInsights(ctx context.Context, params *dynamodb.UpdateContributorInsightsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateContributorInsightsOutput, error)
	UpdateGlobalTable(ctx context.Context, params *dynamodb.UpdateGlobalTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateGlobalTableOutput, error)
	UpdateGlobalTableSettings(ctx context.Context, params *dynamodb.UpdateGlobalTableSettingsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateGlobalTableSettingsOutput, error)
	UpdateTable(ctx context.Context, params *dynamodb.UpdateTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateTableOutput, error)
	UpdateTableReplicaAutoScaling(ctx context.Context, params *dynamodb.UpdateTableReplicaAutoScalingInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateTableReplicaAutoScalingOutput, error)
	UpdateTimeToLive(ctx context.Context, params *dynamodb.UpdateTimeToLiveInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateTimeToLiveOutput, error)
	DeleteResourcePolicy(ctx context.Context, params *dynamodb.DeleteResourcePolicyInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteResourcePolicyOutput, error)
	GetResourcePolicy(ctx context.Context, params *dynamodb.GetResourcePolicyInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetResourcePolicyOutput, error)
	PutResourcePolicy(ctx context.Context, params *dynamodb.PutResourcePolicyInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutResourcePolicyOutput, error)
	UpdateKinesisStreamingDestination(ctx context.Context, params *dynamodb.UpdateKinesisStreamingDestinationInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateKinesisStreamingDestinationOutput, error)
}

func (d *Dax) PutItem(ctx context.Context, input *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	o, cfn, err := d.config.requestOptions(false, ctx, optFns...)
	if err != nil {
		return nil, err
	}
	if cfn != nil {
		defer cfn()
	}

	out, err := d.client.PutItemWithOptions(ctx, input, &dynamodb.PutItemOutput{}, o)
	if err != nil {
		var ccfe *types.ConditionalCheckFailedException
		if errors.As(err, &ccfe) && input.ReturnValuesOnConditionCheckFailure == types.ReturnValuesOnConditionCheckFailureAllOld {
			/*
			* ReturnValuesOnConditionCheckFailure parameter for single write operations allows to return a copy of the item as it was
			* during a failed write attempt, reducing the need for a read request if one want to investigate the cause of the failure.
			*
			* https://aws.amazon.com/blogs/database/handle-conditional-write-errors-in-high-concurrency-scenarios-with-amazon-dynamodb/
			 */
			getOut, getErr := d.GetItem(ctx, &dynamodb.GetItemInput{
				TableName: input.TableName,
				Key:       input.Item,
			})
			if getErr == nil {
				out.Attributes = getOut.Item
			}
			log.Println("[PutItem]: ", out.Attributes, getOut.Item)

			//The code commented below returns the attributes of the item that was supposed to be written, but failed with ConditionalCheckFailedException
			// out.Attributes = input.Item
			return out, err
		}
		return nil, err
	}
	return out, nil
}

func (d *Dax) DeleteItem(ctx context.Context, input *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	o, cfn, err := d.config.requestOptions(false, ctx, optFns...)
	if err != nil {
		return nil, err
	}
	if cfn != nil {
		defer cfn()
	}

	out, err := d.client.DeleteItemWithOptions(ctx, input, &dynamodb.DeleteItemOutput{}, o)
	if err != nil {
		var ccfe *types.ConditionalCheckFailedException
		if errors.As(err, &ccfe) && input.ReturnValuesOnConditionCheckFailure == types.ReturnValuesOnConditionCheckFailureAllOld {
			/*
			* ReturnValuesOnConditionCheckFailure parameter for single write operations allows to return a copy of the item as it was
			* during a failed write attempt, reducing the need for a read request if one want to investigate the cause of the failure.
			*
			* https://aws.amazon.com/blogs/database/handle-conditional-write-errors-in-high-concurrency-scenarios-with-amazon-dynamodb/
			 */

			getOut, getErr := d.GetItem(ctx, &dynamodb.GetItemInput{
				TableName: input.TableName,
				Key:       input.Key,
			})
			if getErr == nil {
				out.Attributes = getOut.Item
			}
			log.Println("[DeleteItem]: ", out.Attributes, getOut.Item)

			//The code commented below returns the attributes of the item that was supposed to be written, but failed with ConditionalCheckFailedException
			// out.Attributes = input.ExpressionAttributeValues
			return out, err
		}
		return nil, err
	}
	return out, nil
}

func (d *Dax) UpdateItem(ctx context.Context, input *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	o, cfn, err := d.config.requestOptions(false, ctx, optFns...)
	if err != nil {
		return nil, err
	}
	if cfn != nil {
		defer cfn()
	}

	out, err := d.client.UpdateItemWithOptions(ctx, input, &dynamodb.UpdateItemOutput{}, o)
	if err != nil {
		var ccfe *types.ConditionalCheckFailedException
		if errors.As(err, &ccfe) && input.ReturnValuesOnConditionCheckFailure == types.ReturnValuesOnConditionCheckFailureAllOld {
			/*
			* ReturnValuesOnConditionCheckFailure parameter for single write operations allows to return a copy of the item as it was
			* at the moment the condition check failed, reducing the need for a read request if one wants to investigate the cause of the failure.
			* The only way to know the state of the item, when the single write operation failed, is to issue another GetItem operation.
			*
			* https://aws.amazon.com/blogs/database/handle-conditional-write-errors-in-high-concurrency-scenarios-with-amazon-dynamodb/
			 */

			getOut, getErr := d.GetItem(ctx, &dynamodb.GetItemInput{
				TableName: input.TableName,
				Key:       input.Key,
			})
			if getErr == nil {
				out.Attributes = getOut.Item
			}
			log.Println("[UpdateItem] out.Attributes:", out.Attributes, "getOut.Item:", getOut.Item, "out:", out)

			//The code commented below returns the attributes of the item that was supposed to be written, but failed with ConditionalCheckFailedException
			// out.Attributes = input.ExpressionAttributeValues
			return out, err
		}
		return nil, err
	}
	return out, nil
}

func (d *Dax) GetItem(ctx context.Context, input *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	o, cfn, err := d.config.requestOptions(true, ctx, optFns...)
	if err != nil {
		return nil, err
	}
	if cfn != nil {
		defer cfn()
	}
	return d.client.GetItemWithOptions(ctx, input, &dynamodb.GetItemOutput{}, o)
}

func (d *Dax) Scan(ctx context.Context, input *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
	o, cfn, err := d.config.requestOptions(true, ctx, optFns...)
	if err != nil {
		return nil, err
	}
	if cfn != nil {
		defer cfn()
	}
	return d.client.ScanWithOptions(ctx, input, &dynamodb.ScanOutput{}, o)
}

func (d *Dax) Query(ctx context.Context, input *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	o, cfn, err := d.config.requestOptions(true, ctx, optFns...)
	if err != nil {
		return nil, err
	}
	if cfn != nil {
		defer cfn()
	}
	return d.client.QueryWithOptions(ctx, input, &dynamodb.QueryOutput{}, o)
}

func (d *Dax) BatchWriteItem(ctx context.Context, input *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
	o, cfn, err := d.config.requestOptions(false, ctx, optFns...)
	if err != nil {
		return nil, err
	}
	if cfn != nil {
		defer cfn()
	}
	return d.client.BatchWriteItemWithOptions(ctx, input, &dynamodb.BatchWriteItemOutput{}, o)
}

func (d *Dax) BatchGetItem(ctx context.Context, input *dynamodb.BatchGetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error) {
	o, cfn, err := d.config.requestOptions(true, ctx, optFns...)
	if err != nil {
		return nil, err
	}
	if cfn != nil {
		defer cfn()
	}
	return d.client.BatchGetItemWithOptions(ctx, input, &dynamodb.BatchGetItemOutput{}, o)
}

func (d *Dax) TransactWriteItems(ctx context.Context, input *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	o, cfn, err := d.config.requestOptions(false, ctx, optFns...)
	if err != nil {
		return nil, err
	}
	if cfn != nil {
		defer cfn()
	}
	return d.client.TransactWriteItemsWithOptions(ctx, input, &dynamodb.TransactWriteItemsOutput{}, o)
}

func (d *Dax) TransactGetItems(ctx context.Context, input *dynamodb.TransactGetItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactGetItemsOutput, error) {
	o, cfn, err := d.config.requestOptions(true, ctx, optFns...)
	if err != nil {
		return nil, err
	}
	if cfn != nil {
		defer cfn()
	}
	return d.client.TransactGetItemsWithOptions(ctx, input, &dynamodb.TransactGetItemsOutput{}, o)
}

func (d *Dax) BatchExecuteStatement(context.Context, *dynamodb.BatchExecuteStatementInput, ...func(*dynamodb.Options)) (*dynamodb.BatchExecuteStatementOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) CreateBackup(context.Context, *dynamodb.CreateBackupInput, ...func(*dynamodb.Options)) (*dynamodb.CreateBackupOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) CreateGlobalTable(context.Context, *dynamodb.CreateGlobalTableInput, ...func(*dynamodb.Options)) (*dynamodb.CreateGlobalTableOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) CreateTable(context.Context, *dynamodb.CreateTableInput, ...func(*dynamodb.Options)) (*dynamodb.CreateTableOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) DeleteBackup(context.Context, *dynamodb.DeleteBackupInput, ...func(*dynamodb.Options)) (*dynamodb.DeleteBackupOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) DeleteTable(context.Context, *dynamodb.DeleteTableInput, ...func(*dynamodb.Options)) (*dynamodb.DeleteTableOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) DescribeBackup(context.Context, *dynamodb.DescribeBackupInput, ...func(*dynamodb.Options)) (*dynamodb.DescribeBackupOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) DescribeContinuousBackups(context.Context, *dynamodb.DescribeContinuousBackupsInput, ...func(*dynamodb.Options)) (*dynamodb.DescribeContinuousBackupsOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) DescribeContributorInsights(context.Context, *dynamodb.DescribeContributorInsightsInput, ...func(*dynamodb.Options)) (*dynamodb.DescribeContributorInsightsOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) DescribeEndpoints(context.Context, *dynamodb.DescribeEndpointsInput, ...func(*dynamodb.Options)) (*dynamodb.DescribeEndpointsOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) DescribeGlobalTable(context.Context, *dynamodb.DescribeGlobalTableInput, ...func(*dynamodb.Options)) (*dynamodb.DescribeGlobalTableOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) DescribeGlobalTableSettings(context.Context, *dynamodb.DescribeGlobalTableSettingsInput, ...func(*dynamodb.Options)) (*dynamodb.DescribeGlobalTableSettingsOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) DescribeImport(context.Context, *dynamodb.DescribeImportInput, ...func(*dynamodb.Options)) (*dynamodb.DescribeImportOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) DescribeLimits(context.Context, *dynamodb.DescribeLimitsInput, ...func(*dynamodb.Options)) (*dynamodb.DescribeLimitsOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) DescribeTable(context.Context, *dynamodb.DescribeTableInput, ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) DescribeTableReplicaAutoScaling(context.Context, *dynamodb.DescribeTableReplicaAutoScalingInput, ...func(*dynamodb.Options)) (*dynamodb.DescribeTableReplicaAutoScalingOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) DescribeTimeToLive(context.Context, *dynamodb.DescribeTimeToLiveInput, ...func(*dynamodb.Options)) (*dynamodb.DescribeTimeToLiveOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) DescribeExport(context.Context, *dynamodb.DescribeExportInput, ...func(*dynamodb.Options)) (*dynamodb.DescribeExportOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) DescribeKinesisStreamingDestination(context.Context, *dynamodb.DescribeKinesisStreamingDestinationInput, ...func(*dynamodb.Options)) (*dynamodb.DescribeKinesisStreamingDestinationOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) DisableKinesisStreamingDestination(context.Context, *dynamodb.DisableKinesisStreamingDestinationInput, ...func(*dynamodb.Options)) (*dynamodb.DisableKinesisStreamingDestinationOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) EnableKinesisStreamingDestination(context.Context, *dynamodb.EnableKinesisStreamingDestinationInput, ...func(*dynamodb.Options)) (*dynamodb.EnableKinesisStreamingDestinationOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) ExecuteStatement(context.Context, *dynamodb.ExecuteStatementInput, ...func(*dynamodb.Options)) (*dynamodb.ExecuteStatementOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) ExecuteTransaction(context.Context, *dynamodb.ExecuteTransactionInput, ...func(*dynamodb.Options)) (*dynamodb.ExecuteTransactionOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) ExportTableToPointInTime(context.Context, *dynamodb.ExportTableToPointInTimeInput, ...func(*dynamodb.Options)) (*dynamodb.ExportTableToPointInTimeOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) ListBackups(context.Context, *dynamodb.ListBackupsInput, ...func(*dynamodb.Options)) (*dynamodb.ListBackupsOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) ImportTable(context.Context, *dynamodb.ImportTableInput, ...func(*dynamodb.Options)) (*dynamodb.ImportTableOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) ListContributorInsights(context.Context, *dynamodb.ListContributorInsightsInput, ...func(*dynamodb.Options)) (*dynamodb.ListContributorInsightsOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) ListExports(context.Context, *dynamodb.ListExportsInput, ...func(*dynamodb.Options)) (*dynamodb.ListExportsOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) ListGlobalTables(context.Context, *dynamodb.ListGlobalTablesInput, ...func(*dynamodb.Options)) (*dynamodb.ListGlobalTablesOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) ListImports(context.Context, *dynamodb.ListImportsInput, ...func(*dynamodb.Options)) (*dynamodb.ListImportsOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) ListTables(context.Context, *dynamodb.ListTablesInput, ...func(*dynamodb.Options)) (*dynamodb.ListTablesOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) ListTagsOfResource(context.Context, *dynamodb.ListTagsOfResourceInput, ...func(*dynamodb.Options)) (*dynamodb.ListTagsOfResourceOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) RestoreTableFromBackup(context.Context, *dynamodb.RestoreTableFromBackupInput, ...func(*dynamodb.Options)) (*dynamodb.RestoreTableFromBackupOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) RestoreTableToPointInTime(context.Context, *dynamodb.RestoreTableToPointInTimeInput, ...func(*dynamodb.Options)) (*dynamodb.RestoreTableToPointInTimeOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) TagResource(context.Context, *dynamodb.TagResourceInput, ...func(*dynamodb.Options)) (*dynamodb.TagResourceOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) UntagResource(context.Context, *dynamodb.UntagResourceInput, ...func(*dynamodb.Options)) (*dynamodb.UntagResourceOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) UpdateContinuousBackups(context.Context, *dynamodb.UpdateContinuousBackupsInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateContinuousBackupsOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) UpdateContributorInsights(context.Context, *dynamodb.UpdateContributorInsightsInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateContributorInsightsOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) UpdateGlobalTable(context.Context, *dynamodb.UpdateGlobalTableInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateGlobalTableOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) UpdateGlobalTableSettings(context.Context, *dynamodb.UpdateGlobalTableSettingsInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateGlobalTableSettingsOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) UpdateTable(context.Context, *dynamodb.UpdateTableInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateTableOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) UpdateTableReplicaAutoScaling(context.Context, *dynamodb.UpdateTableReplicaAutoScalingInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateTableReplicaAutoScalingOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) UpdateTimeToLive(context.Context, *dynamodb.UpdateTimeToLiveInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateTimeToLiveOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) DeleteResourcePolicy(context.Context, *dynamodb.DeleteResourcePolicyInput, ...func(*dynamodb.Options)) (*dynamodb.DeleteResourcePolicyOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) GetResourcePolicy(context.Context, *dynamodb.GetResourcePolicyInput, ...func(*dynamodb.Options)) (*dynamodb.GetResourcePolicyOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) PutResourcePolicy(context.Context, *dynamodb.PutResourcePolicyInput, ...func(*dynamodb.Options)) (*dynamodb.PutResourcePolicyOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) UpdateKinesisStreamingDestination(context.Context, *dynamodb.UpdateKinesisStreamingDestinationInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateKinesisStreamingDestinationOutput, error) {
	return nil, d.unImpl()
}

func (d *Dax) unImpl() error {
	return errors.New(client.ErrCodeNotImplemented)
}

func (d *Dax) Close() error {
	if c, ok := d.client.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
