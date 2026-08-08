package verdict

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SimonJPegg/portcullis/backend/internal/domain"
)

// mockDynamo implements DynamoAPI for testing.
type mockDynamo struct {
	getItemFunc func(ctx context.Context, input *dynamodb.GetItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	putItemFunc func(ctx context.Context, input *dynamodb.PutItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

func (m *mockDynamo) GetItem(ctx context.Context, input *dynamodb.GetItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return m.getItemFunc(ctx, input, opts...)
}

func (m *mockDynamo) PutItem(ctx context.Context, input *dynamodb.PutItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return m.putItemFunc(ctx, input, opts...)
}

func validCoordinate(t *testing.T) domain.PackageCoordinate {
	t.Helper()
	coord, err := domain.NewPackageCoordinate(domain.PyPI, "requests", "2.31.0")
	require.NoError(t, err)
	return coord
}

func TestNewDynamoVerdictStore_NilClient(t *testing.T) {
	_, err := NewDynamoVerdictStore(nil, "verdicts")
	assert.Error(t, err)
}

func TestNewDynamoVerdictStore_EmptyTable(t *testing.T) {
	_, err := NewDynamoVerdictStore(&mockDynamo{}, "")
	assert.Error(t, err)
}

func TestGet_Miss_ReturnsPending(t *testing.T) {
	mock := &mockDynamo{
		getItemFunc: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
	}

	store, err := NewDynamoVerdictStore(mock, "verdicts")
	require.NoError(t, err)

	verdict, err := store.Get(validCoordinate(t))
	assert.NoError(t, err)
	assert.IsType(t, domain.Pending{}, verdict)
}

func TestGet_Allowed(t *testing.T) {
	mock := &mockDynamo{
		getItemFunc: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"pk":      &types.AttributeValueMemberS{Value: "pypi#requests#2.31.0"},
					"verdict": &types.AttributeValueMemberS{Value: "allowed"},
				},
			}, nil
		},
	}

	store, err := NewDynamoVerdictStore(mock, "verdicts")
	require.NoError(t, err)

	verdict, err := store.Get(validCoordinate(t))
	assert.NoError(t, err)
	assert.IsType(t, domain.Allowed{}, verdict)
}

func TestGet_Denied_WithPolicyId(t *testing.T) {
	mock := &mockDynamo{
		getItemFunc: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"pk":        &types.AttributeValueMemberS{Value: "pypi#requests#2.31.0"},
					"verdict":   &types.AttributeValueMemberS{Value: "denied"},
					"reason":    &types.AttributeValueMemberS{Value: "known vulnerability CVE-2023-1234"},
					"policy_id": &types.AttributeValueMemberS{Value: "vuln-policy-1"},
				},
			}, nil
		},
	}

	store, err := NewDynamoVerdictStore(mock, "verdicts")
	require.NoError(t, err)

	verdict, err := store.Get(validCoordinate(t))
	assert.NoError(t, err)

	denied, ok := verdict.(domain.Denied)
	require.True(t, ok)
	assert.Equal(t, "known vulnerability CVE-2023-1234", denied.Reason)
	assert.Equal(t, "vuln-policy-1", *denied.PolicyId)
}

func TestGet_Denied_WithoutPolicyId(t *testing.T) {
	mock := &mockDynamo{
		getItemFunc: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"pk":      &types.AttributeValueMemberS{Value: "pypi#requests#2.31.0"},
					"verdict": &types.AttributeValueMemberS{Value: "denied"},
					"reason":  &types.AttributeValueMemberS{Value: "package too new"},
				},
			}, nil
		},
	}

	store, err := NewDynamoVerdictStore(mock, "verdicts")
	require.NoError(t, err)

	verdict, err := store.Get(validCoordinate(t))
	assert.NoError(t, err)

	denied, ok := verdict.(domain.Denied)
	require.True(t, ok)
	assert.Equal(t, "package too new", denied.Reason)
	assert.Nil(t, denied.PolicyId)
}

func TestGet_NetworkError_PropagatesError(t *testing.T) {
	mock := &mockDynamo{
		getItemFunc: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return nil, errors.New("connection refused")
		},
	}

	store, err := NewDynamoVerdictStore(mock, "verdicts")
	require.NoError(t, err)

	verdict, err := store.Get(validCoordinate(t))
	assert.Error(t, err)
	assert.Nil(t, verdict)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestGet_CorruptedItem_MissingVerdictAttribute(t *testing.T) {
	mock := &mockDynamo{
		getItemFunc: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"pk": &types.AttributeValueMemberS{Value: "pypi#requests#2.31.0"},
				},
			}, nil
		},
	}

	store, err := NewDynamoVerdictStore(mock, "verdicts")
	require.NoError(t, err)

	verdict, err := store.Get(validCoordinate(t))
	assert.Error(t, err)
	assert.Nil(t, verdict)
	assert.Contains(t, err.Error(), "missing 'verdict' attribute")
}

func TestGet_UnknownVerdictValue(t *testing.T) {
	mock := &mockDynamo{
		getItemFunc: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"pk":      &types.AttributeValueMemberS{Value: "pypi#requests#2.31.0"},
					"verdict": &types.AttributeValueMemberS{Value: "explode"},
				},
			}, nil
		},
	}

	store, err := NewDynamoVerdictStore(mock, "verdicts")
	require.NoError(t, err)

	verdict, err := store.Get(validCoordinate(t))
	assert.Error(t, err)
	assert.Nil(t, verdict)
	assert.Contains(t, err.Error(), "unknown verdict value")
}

func TestPut_Allowed(t *testing.T) {
	var capturedInput *dynamodb.PutItemInput

	mock := &mockDynamo{
		putItemFunc: func(_ context.Context, input *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			capturedInput = input
			return &dynamodb.PutItemOutput{}, nil
		},
	}

	store, err := NewDynamoVerdictStore(mock, "verdicts")
	require.NoError(t, err)

	err = store.Put(validCoordinate(t), domain.Allowed{})
	assert.NoError(t, err)

	assert.Equal(t, "verdicts", *capturedInput.TableName)
	assert.Equal(t, "pypi#requests#2.31.0", capturedInput.Item["pk"].(*types.AttributeValueMemberS).Value)
	assert.Equal(t, "allowed", capturedInput.Item["verdict"].(*types.AttributeValueMemberS).Value)
}

func TestPut_Denied(t *testing.T) {
	var capturedInput *dynamodb.PutItemInput
	policyId := "age-policy-1"

	mock := &mockDynamo{
		putItemFunc: func(_ context.Context, input *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			capturedInput = input
			return &dynamodb.PutItemOutput{}, nil
		},
	}

	store, err := NewDynamoVerdictStore(mock, "verdicts")
	require.NoError(t, err)

	err = store.Put(validCoordinate(t), domain.Denied{Reason: "too new", PolicyId: &policyId})
	assert.NoError(t, err)

	assert.Equal(t, "denied", capturedInput.Item["verdict"].(*types.AttributeValueMemberS).Value)
	assert.Equal(t, "too new", capturedInput.Item["reason"].(*types.AttributeValueMemberS).Value)
	assert.Equal(t, "age-policy-1", capturedInput.Item["policy_id"].(*types.AttributeValueMemberS).Value)
}

func TestPut_NetworkError_PropagatesError(t *testing.T) {
	mock := &mockDynamo{
		putItemFunc: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, errors.New("throughput exceeded")
		},
	}

	store, err := NewDynamoVerdictStore(mock, "verdicts")
	require.NoError(t, err)

	err = store.Put(validCoordinate(t), domain.Allowed{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "throughput exceeded")
}

func TestPut_Overwrite(t *testing.T) {
	var calls int

	mock := &mockDynamo{
		putItemFunc: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			calls++
			return &dynamodb.PutItemOutput{}, nil
		},
		getItemFunc: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"pk":      &types.AttributeValueMemberS{Value: "pypi#requests#2.31.0"},
					"verdict": &types.AttributeValueMemberS{Value: "denied"},
					"reason":  &types.AttributeValueMemberS{Value: "vuln found"},
				},
			}, nil
		},
	}

	store, err := NewDynamoVerdictStore(mock, "verdicts")
	require.NoError(t, err)

	coord := validCoordinate(t)

	// First write: allowed
	err = store.Put(coord, domain.Allowed{})
	require.NoError(t, err)

	// Second write: denied (overwrites)
	err = store.Put(coord, domain.Denied{Reason: "vuln found", PolicyId: nil})
	require.NoError(t, err)

	assert.Equal(t, 2, calls)

	// Verify the latest verdict is returned
	verdict, err := store.Get(coord)
	assert.NoError(t, err)
	assert.IsType(t, domain.Denied{}, verdict)
}

func TestCoordinateToKey_Format(t *testing.T) {
	coord := validCoordinate(t)
	key := coordinateToKey(coord)

	pk, ok := key["pk"].(*types.AttributeValueMemberS)
	require.True(t, ok)
	assert.Equal(t, "pypi#requests#2.31.0", pk.Value)
}
