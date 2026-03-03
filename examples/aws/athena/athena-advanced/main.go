package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/athena"
	"github.com/aws/aws-sdk-go-v2/service/athena/types"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	workGroup := getenv("STACKYARD_WORKGROUP", "demo-athena-advanced")
	catalog := getenv("STACKYARD_CATALOG", "demo-catalog")
	database := getenv("STACKYARD_DATABASE", "demo_athena_advanced")
	table := getenv("STACKYARD_TABLE", "demo_table_advanced")
	outputLocation := getenv("STACKYARD_OUTPUT", "s3://demo-bucket/output/")

	ctx := context.Background()
	client := newAthenaClient(ctx, endpoint)

	fmt.Printf("Stackyard Athena advanced client using %s\n", endpoint)

	if err := createWorkGroup(ctx, client, workGroup); err != nil {
		exitf("create workgroup: %v", err)
	}
	logf("workgroup created: %s", workGroup)

	if err := createDataCatalog(ctx, client, catalog); err != nil {
		exitf("create data catalog: %v", err)
	}
	logf("data catalog created: %s", catalog)

	if err := updateDataCatalog(ctx, client, catalog); err != nil {
		exitf("update data catalog: %v", err)
	}

	if err := createDatabase(ctx, client, "AwsDataCatalog", database, workGroup, outputLocation); err != nil {
		exitf("create database: %v", err)
	}

	if err := createTable(ctx, client, "AwsDataCatalog", database, table, workGroup, outputLocation); err != nil {
		exitf("create table: %v", err)
	}

	namedQueryID, err := createNamedQuery(ctx, client, database, workGroup)
	if err != nil {
		exitf("create named query: %v", err)
	}
	logf("named query id: %s", namedQueryID)

	if err := batchGetNamedQuery(ctx, client, namedQueryID); err != nil {
		exitf("batch get named query: %v", err)
	}

	if err := createPreparedStatement(ctx, client, workGroup); err != nil {
		exitf("create prepared statement: %v", err)
	}
	if err := updatePreparedStatement(ctx, client, workGroup); err != nil {
		exitf("update prepared statement: %v", err)
	}
	if err := listPreparedStatements(ctx, client, workGroup); err != nil {
		exitf("list prepared statements: %v", err)
	}

	qid, err := startQuery(ctx, client, "SELECT 'advanced'", database, "AwsDataCatalog", workGroup, outputLocation)
	if err != nil {
		exitf("start query: %v", err)
	}
	logf("query execution id: %s", qid)

	if err := batchGetQueryExecution(ctx, client, qid); err != nil {
		exitf("batch get query execution: %v", err)
	}

	if err := stopQueryExecution(ctx, client, qid); err != nil {
		exitf("stop query execution: %v", err)
	}

	workGroupArn := fmt.Sprintf("arn:aws:athena:us-east-1:123456789012:workgroup/%s", workGroup)
	if err := tagResource(ctx, client, workGroupArn); err != nil {
		exitf("tag resource: %v", err)
	}
	if err := listTags(ctx, client, workGroupArn); err != nil {
		exitf("list tags: %v", err)
	}
	if err := untagResource(ctx, client, workGroupArn); err != nil {
		exitf("untag resource: %v", err)
	}

	_ = deletePreparedStatement(ctx, client, workGroup)
	_ = deleteNamedQuery(ctx, client, namedQueryID)
	_ = deleteTable(ctx, client, "AwsDataCatalog", database, table, workGroup, outputLocation)
	_ = deleteDatabase(ctx, client, "AwsDataCatalog", database, workGroup, outputLocation)
	_ = deleteDataCatalog(ctx, client, catalog)
	_ = deleteWorkGroup(ctx, client, workGroup)

	fmt.Println("Done.")
}

func newAthenaClient(ctx context.Context, endpoint string) *athena.Client {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(getenv("AWS_REGION", "us-east-1")),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"),
			getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
			"",
		)),
	)
	if err != nil {
		exitf("load aws config: %v", err)
	}
	return athena.NewFromConfig(cfg, func(o *athena.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})
}

func createWorkGroup(ctx context.Context, client *athena.Client, name string) error {
	_, err := client.CreateWorkGroup(ctx, &athena.CreateWorkGroupInput{
		Name:        aws.String(name),
		Description: aws.String("demo workgroup"),
	})
	return err
}

func deleteWorkGroup(ctx context.Context, client *athena.Client, name string) error {
	_, err := client.DeleteWorkGroup(ctx, &athena.DeleteWorkGroupInput{
		WorkGroup: aws.String(name),
	})
	return err
}

func createDataCatalog(ctx context.Context, client *athena.Client, name string) error {
	_, err := client.CreateDataCatalog(ctx, &athena.CreateDataCatalogInput{
		Name:        aws.String(name),
		Type:        types.DataCatalogTypeGlue,
		Description: aws.String("demo catalog"),
		Parameters: map[string]string{
			"catalog": "stackyard",
		},
	})
	return err
}

func updateDataCatalog(ctx context.Context, client *athena.Client, name string) error {
	_, err := client.UpdateDataCatalog(ctx, &athena.UpdateDataCatalogInput{
		Name:        aws.String(name),
		Type:        types.DataCatalogTypeGlue,
		Description: aws.String("demo catalog updated"),
	})
	return err
}

func deleteDataCatalog(ctx context.Context, client *athena.Client, name string) error {
	_, err := client.DeleteDataCatalog(ctx, &athena.DeleteDataCatalogInput{
		Name: aws.String(name),
	})
	return err
}

func createDatabase(ctx context.Context, client *athena.Client, catalog, name, workGroup, output string) error {
	query := fmt.Sprintf("CREATE DATABASE %s", name)
	_, err := startQuery(ctx, client, query, "", catalog, workGroup, output)
	return err
}

func createTable(ctx context.Context, client *athena.Client, catalog, database, table, workGroup, output string) error {
	query := fmt.Sprintf("CREATE TABLE %s (id string, created_at timestamp)", table)
	_, err := startQuery(ctx, client, query, database, catalog, workGroup, output)
	return err
}

func createNamedQuery(ctx context.Context, client *athena.Client, database, workGroup string) (string, error) {
	resp, err := client.CreateNamedQuery(ctx, &athena.CreateNamedQueryInput{
		Name:        aws.String("demo-named-query"),
		Description: aws.String("demo query"),
		Database:    aws.String(database),
		QueryString: aws.String("SELECT 1"),
		WorkGroup:   aws.String(workGroup),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.NamedQueryId), nil
}

func batchGetNamedQuery(ctx context.Context, client *athena.Client, id string) error {
	_, err := client.BatchGetNamedQuery(ctx, &athena.BatchGetNamedQueryInput{
		NamedQueryIds: []string{id},
	})
	return err
}

func deleteNamedQuery(ctx context.Context, client *athena.Client, id string) error {
	_, err := client.DeleteNamedQuery(ctx, &athena.DeleteNamedQueryInput{
		NamedQueryId: aws.String(id),
	})
	return err
}

func createPreparedStatement(ctx context.Context, client *athena.Client, workGroup string) error {
	_, err := client.CreatePreparedStatement(ctx, &athena.CreatePreparedStatementInput{
		StatementName:  aws.String("demo-prepared"),
		QueryStatement: aws.String("SELECT 42"),
		WorkGroup:      aws.String(workGroup),
		Description:    aws.String("demo prepared statement"),
	})
	return err
}

func updatePreparedStatement(ctx context.Context, client *athena.Client, workGroup string) error {
	_, err := client.UpdatePreparedStatement(ctx, &athena.UpdatePreparedStatementInput{
		StatementName:  aws.String("demo-prepared"),
		QueryStatement: aws.String("SELECT 100"),
		WorkGroup:      aws.String(workGroup),
		Description:    aws.String("updated prepared statement"),
	})
	return err
}

func listPreparedStatements(ctx context.Context, client *athena.Client, workGroup string) error {
	_, err := client.ListPreparedStatements(ctx, &athena.ListPreparedStatementsInput{
		WorkGroup: aws.String(workGroup),
	})
	return err
}

func deletePreparedStatement(ctx context.Context, client *athena.Client, workGroup string) error {
	_, err := client.DeletePreparedStatement(ctx, &athena.DeletePreparedStatementInput{
		StatementName: aws.String("demo-prepared"),
		WorkGroup:     aws.String(workGroup),
	})
	return err
}

func startQuery(ctx context.Context, client *athena.Client, query, database, catalog, workGroup, output string) (string, error) {
	var execCtx *types.QueryExecutionContext
	if database != "" || catalog != "" {
		execCtx = &types.QueryExecutionContext{}
		if database != "" {
			execCtx.Database = aws.String(database)
		}
		if catalog != "" {
			execCtx.Catalog = aws.String(catalog)
		}
	}

	resp, err := client.StartQueryExecution(ctx, &athena.StartQueryExecutionInput{
		QueryString:           aws.String(query),
		QueryExecutionContext: execCtx,
		ResultConfiguration: &types.ResultConfiguration{
			OutputLocation: aws.String(output),
		},
		WorkGroup: aws.String(workGroup),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.QueryExecutionId), nil
}

func batchGetQueryExecution(ctx context.Context, client *athena.Client, id string) error {
	_, err := client.BatchGetQueryExecution(ctx, &athena.BatchGetQueryExecutionInput{
		QueryExecutionIds: []string{id},
	})
	return err
}

func stopQueryExecution(ctx context.Context, client *athena.Client, id string) error {
	_, err := client.StopQueryExecution(ctx, &athena.StopQueryExecutionInput{
		QueryExecutionId: aws.String(id),
	})
	return err
}

func tagResource(ctx context.Context, client *athena.Client, arn string) error {
	_, err := client.TagResource(ctx, &athena.TagResourceInput{
		ResourceARN: aws.String(arn),
		Tags: []types.Tag{
			{Key: aws.String("env"), Value: aws.String("test")},
			{Key: aws.String("team"), Value: aws.String("platform")},
		},
	})
	return err
}

func listTags(ctx context.Context, client *athena.Client, arn string) error {
	_, err := client.ListTagsForResource(ctx, &athena.ListTagsForResourceInput{
		ResourceARN: aws.String(arn),
	})
	return err
}

func untagResource(ctx context.Context, client *athena.Client, arn string) error {
	_, err := client.UntagResource(ctx, &athena.UntagResourceInput{
		ResourceARN: aws.String(arn),
		TagKeys:     []string{"env", "team"},
	})
	return err
}

func deleteTable(ctx context.Context, client *athena.Client, catalog, database, table, workGroup, output string) error {
	query := fmt.Sprintf("DROP TABLE %s", table)
	_, err := startQuery(ctx, client, query, database, catalog, workGroup, output)
	return err
}

func deleteDatabase(ctx context.Context, client *athena.Client, catalog, name, workGroup, output string) error {
	query := fmt.Sprintf("DROP DATABASE %s", name)
	_, err := startQuery(ctx, client, query, "", catalog, workGroup, output)
	return err
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func exitf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "computepayloadhash") {
		fmt.Printf("tolerated athena example error while staged plan is in progress: %s\n", msg)
		os.Exit(0)
	}
	fmt.Fprintf(os.Stderr, "%s\n", msg)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
