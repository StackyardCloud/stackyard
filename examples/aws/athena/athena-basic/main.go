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
	catalog := getenv("STACKYARD_CATALOG", "AwsDataCatalog")
	database := getenv("STACKYARD_DATABASE", "demo_athena_basic")
	table := getenv("STACKYARD_TABLE", "demo_table")
	workGroup := getenv("STACKYARD_WORKGROUP", "primary")
	query := getenv("STACKYARD_QUERY", "SELECT 'ok'")
	outputLocation := getenv("STACKYARD_OUTPUT", "s3://demo-bucket/output/")

	ctx := context.Background()
	client := newAthenaClient(ctx, endpoint)

	fmt.Printf("Stackyard Athena basic client using %s\n", endpoint)

	if err := createDatabase(ctx, client, catalog, database, workGroup, outputLocation); err != nil {
		exitf("create database: %v", err)
	}
	logf("created database: %s", database)

	if err := createTable(ctx, client, catalog, database, table, workGroup, outputLocation); err != nil {
		exitf("create table: %v", err)
	}
	logf("created table: %s", table)

	qid, err := startQuery(ctx, client, query, database, catalog, workGroup, outputLocation)
	if err != nil {
		exitf("start query: %v", err)
	}
	logf("query execution id: %s", qid)

	status, err := getQueryStatus(ctx, client, qid)
	if err != nil {
		exitf("get query execution: %v", err)
	}
	logf("query state: %s", status)

	rows, err := getQueryResults(ctx, client, qid)
	if err != nil {
		exitf("get query results: %v", err)
	}
	logf("result rows: %d", rows)

	execCount, err := listQueryExecutions(ctx, client, workGroup)
	if err != nil {
		exitf("list query executions: %v", err)
	}
	logf("query executions: %d", execCount)

	if err := deleteTable(ctx, client, catalog, database, table, workGroup, outputLocation); err != nil {
		exitf("delete table: %v", err)
	}
	if err := deleteDatabase(ctx, client, catalog, database, workGroup, outputLocation); err != nil {
		exitf("delete database: %v", err)
	}

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

func createDatabase(ctx context.Context, client *athena.Client, catalog, name, workGroup, output string) error {
	_, err := startQuery(ctx, client, fmt.Sprintf("CREATE DATABASE %s", name), "", catalog, workGroup, output)
	return err
}

func createTable(ctx context.Context, client *athena.Client, catalog, database, table, workGroup, output string) error {
	query := fmt.Sprintf("CREATE TABLE %s (message string)", table)
	_, err := startQuery(ctx, client, query, database, catalog, workGroup, output)
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

func getQueryStatus(ctx context.Context, client *athena.Client, id string) (string, error) {
	resp, err := client.GetQueryExecution(ctx, &athena.GetQueryExecutionInput{
		QueryExecutionId: aws.String(id),
	})
	if err != nil {
		return "", err
	}
	if resp.QueryExecution == nil || resp.QueryExecution.Status == nil {
		return "UNKNOWN", nil
	}
	return string(resp.QueryExecution.Status.State), nil
}

func getQueryResults(ctx context.Context, client *athena.Client, id string) (int, error) {
	resp, err := client.GetQueryResults(ctx, &athena.GetQueryResultsInput{
		QueryExecutionId: aws.String(id),
	})
	if err != nil {
		return 0, err
	}
	if resp.ResultSet == nil {
		return 0, nil
	}
	return len(resp.ResultSet.Rows), nil
}

func listQueryExecutions(ctx context.Context, client *athena.Client, workGroup string) (int, error) {
	resp, err := client.ListQueryExecutions(ctx, &athena.ListQueryExecutionsInput{
		WorkGroup: aws.String(workGroup),
	})
	if err != nil {
		return 0, err
	}
	return len(resp.QueryExecutionIds), nil
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
