//go:build integration

package verdict_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/SimonJPegg/portcullis/backend/internal/domain"
	"github.com/SimonJPegg/portcullis/backend/internal/verdict"
)

const tableName = "verdicts"

func setupDynamoDB(t *testing.T) *dynamodb.Client {
	t.Helper()
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "amazon/dynamodb-local:latest",
			ExposedPorts: []string{"8000/tcp"},
			WaitingFor:   wait.ForListeningPort("8000/tcp"),
		},
		Started: true,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, container.Terminate(ctx))
	})

	endpoint, err := container.Endpoint(ctx, "http")
	require.NoError(t, err)

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("eu-west-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("dummy", "dummy", "dummy")),
	)
	require.NoError(t, err)

	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	_, err = client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("pk"), KeyType: types.KeyTypeHash},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("pk"), AttributeType: types.ScalarAttributeTypeS},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	require.NoError(t, err)

	return client
}

func TestIntegration_GetMiss_ReturnsPending(t *testing.T) {
	client := setupDynamoDB(t)

	store, err := verdict.NewDynamoVerdictStore(client, tableName)
	require.NoError(t, err)

	coord, err := domain.NewPackageCoordinate(domain.PyPI, "requests", "2.31.0")
	require.NoError(t, err)

	v, err := store.Get(coord)
	assert.NoError(t, err)
	assert.IsType(t, domain.Pending{}, v)
}

func TestIntegration_PutThenGet_Allowed(t *testing.T) {
	client := setupDynamoDB(t)

	store, err := verdict.NewDynamoVerdictStore(client, tableName)
	require.NoError(t, err)

	coord, err := domain.NewPackageCoordinate(domain.PyPI, "requests", "2.31.0")
	require.NoError(t, err)

	err = store.Put(coord, domain.Allowed{})
	require.NoError(t, err)

	v, err := store.Get(coord)
	assert.NoError(t, err)
	assert.IsType(t, domain.Allowed{}, v)
}

func TestIntegration_PutThenGet_Denied(t *testing.T) {
	client := setupDynamoDB(t)

	store, err := verdict.NewDynamoVerdictStore(client, tableName)
	require.NoError(t, err)

	coord, err := domain.NewPackageCoordinate(domain.Maven, "com.google.guava:guava", "33.0.0-jre")
	require.NoError(t, err)

	policyId := "vuln-policy-1"
	err = store.Put(coord, domain.Denied{Reason: "CVE-2023-1234", PolicyId: &policyId})
	require.NoError(t, err)

	v, err := store.Get(coord)
	assert.NoError(t, err)

	denied, ok := v.(domain.Denied)
	require.True(t, ok)
	assert.Equal(t, "CVE-2023-1234", denied.Reason)
	assert.Equal(t, "vuln-policy-1", *denied.PolicyId)
}

func TestIntegration_Overwrite_UpdatesVerdict(t *testing.T) {
	client := setupDynamoDB(t)

	store, err := verdict.NewDynamoVerdictStore(client, tableName)
	require.NoError(t, err)

	coord, err := domain.NewPackageCoordinate(domain.PyPI, "flask", "3.0.0")
	require.NoError(t, err)

	// Initially allowed
	err = store.Put(coord, domain.Allowed{})
	require.NoError(t, err)

	v, err := store.Get(coord)
	require.NoError(t, err)
	assert.IsType(t, domain.Allowed{}, v)

	// Vuln discovered — flip to denied
	err = store.Put(coord, domain.Denied{Reason: "new vuln", PolicyId: nil})
	require.NoError(t, err)

	v, err = store.Get(coord)
	require.NoError(t, err)
	assert.IsType(t, domain.Denied{}, v)
	assert.Equal(t, "new vuln", v.(domain.Denied).Reason)
}

func TestIntegration_TwoPackages_Isolated(t *testing.T) {
	client := setupDynamoDB(t)

	store, err := verdict.NewDynamoVerdictStore(client, tableName)
	require.NoError(t, err)

	requests, err := domain.NewPackageCoordinate(domain.PyPI, "requests", "2.31.0")
	require.NoError(t, err)

	flask, err := domain.NewPackageCoordinate(domain.PyPI, "flask", "3.0.0")
	require.NoError(t, err)

	err = store.Put(requests, domain.Allowed{})
	require.NoError(t, err)

	err = store.Put(flask, domain.Denied{Reason: "too new", PolicyId: nil})
	require.NoError(t, err)

	// Each package has its own verdict
	v1, err := store.Get(requests)
	assert.NoError(t, err)
	assert.IsType(t, domain.Allowed{}, v1)

	v2, err := store.Get(flask)
	assert.NoError(t, err)
	assert.IsType(t, domain.Denied{}, v2)
}
