package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbiface"
	"github.com/pkg/errors"

	commonpb "github.com/kinecosystem/kin-api/genproto/common/v3"

	"github.com/kinecosystem/agora-transaction-services-internal/pkg/data"
)

type db struct {
	db dynamodbiface.ClientAPI
}

// New returns a dynamo backed data.Store
func New(client dynamodbiface.ClientAPI) data.Store {
	return &db{
		db: client,
	}
}

// Add implements data.Store.Add.
func (d *db) Add(ctx context.Context, ad *commonpb.AgoraData) error {
	item, err := toItem(ad)
	if err != nil {
		return err
	}

	_, err = d.db.PutItemRequest(&dynamodb.PutItemInput{
		TableName:           tableNameStr,
		Item:                item,
		ConditionExpression: putConditionStr,
	}).Send(ctx)
	if err != nil {
		if aErr, ok := err.(awserr.Error); ok {
			switch aErr.Code() {
			case dynamodb.ErrCodeConditionalCheckFailedException:
				return data.ErrCollision
			}
		}

		return errors.Wrapf(err, "failed to persist message")
	}

	return nil
}

// Get implements data.Store.Get.
func (d *db) Get(ctx context.Context, prefixOrKey []byte) (*commonpb.AgoraData, error) {
	queryExpression := "prefix = :prefix"
	queryValues := make(map[string]dynamodb.AttributeValue)

	switch len(prefixOrKey) {
	case 32: // full key
		queryExpression += " and suffix = :suffix"
		queryValues[":suffix"] = dynamodb.AttributeValue{B: prefixOrKey[29:]}
		fallthrough
	case 29: // prefix
		queryValues[":prefix"] = dynamodb.AttributeValue{B: prefixOrKey[:29]}
	default:
		return nil, errors.Errorf("invalid key len: %d", len(prefixOrKey))
	}

	// Note: we don't need to page here because we're limiting the results to 2.
	//
	// todo: expand the set of results and validate against the entire memo input.
	//       this should be currently protected against via the Add(), though.
	resp, err := d.db.QueryRequest(&dynamodb.QueryInput{
		TableName:                 tableNameStr,
		Limit:                     aws.Int64(2),
		KeyConditionExpression:    aws.String(queryExpression),
		ExpressionAttributeValues: queryValues,
	}).Send(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to dynamo")
	}

	if len(resp.Items) == 0 {
		return nil, data.ErrNotFound
	}
	if len(resp.Items) > 1 {
		// todo: address this issue
		return nil, data.ErrCollision
	}

	return fromItem(resp.Items[0])
}
