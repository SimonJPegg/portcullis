package verdict

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/SimonJPegg/portcullis/backend/internal/domain"
)

// DynamoAPI is the subset of the DynamoDB client needed by the verdict store.
type DynamoAPI interface {
	GetItem(ctx context.Context, input *dynamodb.GetItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, input *dynamodb.PutItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

// DynamoVerdictStore stores and retrieves verdicts from DynamoDB.
type DynamoVerdictStore struct {
	client    DynamoAPI
	tableName string
}

// Compile-time check: DynamoVerdictStore satisfies domain.VerdictStore.
var _ domain.VerdictStore = (*DynamoVerdictStore)(nil)

// NewDynamoVerdictStore constructs a DynamoDB-backed verdict store.
func NewDynamoVerdictStore(client DynamoAPI, tableName string) (*DynamoVerdictStore, error) {
	if client == nil {
		return nil, errors.New("client must not be nil")
	}
	if tableName == "" {
		return nil, errors.New("tableName must not be empty")
	}
	return &DynamoVerdictStore{client: client, tableName: tableName}, nil
}

// Get retrieves a verdict for the given coordinate. Returns Pending on cache miss.
func (s *DynamoVerdictStore) Get(coord domain.PackageCoordinate) (domain.Verdict, error) {
	key := coordinateToKey(coord)

	out, err := s.client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key:       key,
	})
	if err != nil {
		return nil, fmt.Errorf("get verdict: %w", err)
	}

	if out.Item == nil {
		return domain.Pending{}, nil
	}

	return itemToVerdict(out.Item)
}

// Put writes a verdict for the given coordinate, overwriting any existing entry.
func (s *DynamoVerdictStore) Put(coord domain.PackageCoordinate, verdict domain.Verdict) error {
	item, err := verdictToItem(coord, verdict)
	if err != nil {
		return fmt.Errorf("put verdict: %w", err)
	}

	_, err = s.client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put verdict: %w", err)
	}

	return nil
}

func coordinateToKey(coord domain.PackageCoordinate) map[string]types.AttributeValue {
	pk := fmt.Sprintf("%s#%s#%s", coord.EcoSystem(), coord.Name(), coord.Version())
	return map[string]types.AttributeValue{
		"pk": &types.AttributeValueMemberS{Value: pk},
	}
}

func verdictToItem(coord domain.PackageCoordinate, verdict domain.Verdict) (map[string]types.AttributeValue, error) {
	item := coordinateToKey(coord)

	switch v := verdict.(type) {
	case domain.Allowed:
		item["verdict"] = &types.AttributeValueMemberS{Value: "allowed"}
	case domain.Denied:
		item["verdict"] = &types.AttributeValueMemberS{Value: "denied"}
		item["reason"] = &types.AttributeValueMemberS{Value: v.Reason}
		if v.PolicyId != nil {
			item["policy_id"] = &types.AttributeValueMemberS{Value: *v.PolicyId}
		}
	case domain.Pending:
		item["verdict"] = &types.AttributeValueMemberS{Value: "pending"}
	default:
		return nil, fmt.Errorf("unknown verdict type: %T", verdict)
	}

	return item, nil
}

func itemToVerdict(item map[string]types.AttributeValue) (domain.Verdict, error) {
	verdictAttr, ok := item["verdict"]
	if !ok {
		return nil, errors.New("item missing 'verdict' attribute")
	}

	verdictStr, ok := verdictAttr.(*types.AttributeValueMemberS)
	if !ok {
		return nil, errors.New("'verdict' attribute is not a string")
	}

	switch verdictStr.Value {
	case "allowed":
		return domain.Allowed{}, nil
	case "denied":
		reason := extractString(item, "reason")
		policyId := extractStringPtr(item, "policy_id")
		return domain.Denied{Reason: reason, PolicyId: policyId}, nil
	case "pending":
		return domain.Pending{}, nil
	default:
		return nil, fmt.Errorf("unknown verdict value: %s", verdictStr.Value)
	}
}

func extractString(item map[string]types.AttributeValue, key string) string {
	attr, ok := item[key]
	if !ok {
		return ""
	}
	s, ok := attr.(*types.AttributeValueMemberS)
	if !ok {
		return ""
	}
	return s.Value
}

func extractStringPtr(item map[string]types.AttributeValue, key string) *string {
	v := extractString(item, key)
	if v == "" {
		return nil
	}
	return &v
}
