package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	procurement "cloud.google.com/go/commerce/consumer/procurement/apiv1"
	"cloud.google.com/go/commerce/consumer/procurement/apiv1/procurementpb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *procurement.ConsumerProcurementClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	billingAccountID := getenv("STACKYARD_GCP_BILLING_ACCOUNT_ID", "0123456789")
	orderID := getenv("STACKYARD_GCP_PROCUREMENT_ORDER_ID", "order-1")
	operationID := getenv("STACKYARD_GCP_PROCUREMENT_OPERATION_ID", "op-1")

	parent := fmt.Sprintf("billingAccounts/%s", billingAccountID)
	orderName := parent + "/orders/" + orderID
	operationName := orderName + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Commerce Consumer Procurement apiv1 client using %s\n", apiEndpoint)

	client, err := procurement.NewConsumerProcurementRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create consumer procurement client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListOrders",
			call: func(ctx context.Context, c *procurement.ConsumerProcurementClient) error {
				it := c.ListOrders(ctx, &procurementpb.ListOrdersRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetOrder",
			call: func(ctx context.Context, c *procurement.ConsumerProcurementClient) error {
				_, err := c.GetOrder(ctx, &procurementpb.GetOrderRequest{Name: orderName})
				return err
			},
		},
		{
			name: "PlaceOrder",
			call: func(ctx context.Context, c *procurement.ConsumerProcurementClient) error {
				_, err := c.PlaceOrder(ctx, &procurementpb.PlaceOrderRequest{
					Parent:      parent,
					DisplayName: "Team commitment order",
				})
				return err
			},
		},
		{
			name: "ModifyOrder",
			call: func(ctx context.Context, c *procurement.ConsumerProcurementClient) error {
				_, err := c.ModifyOrder(ctx, &procurementpb.ModifyOrderRequest{
					Name:        orderName,
					DisplayName: "Team commitment order updated",
					Modifications: []*procurementpb.ModifyOrderRequest_Modification{
						{
							LineItemId: "line-item-1",
							ChangeType: procurementpb.LineItemChangeType_LINE_ITEM_CHANGE_TYPE_CANCEL,
						},
					},
				})
				return err
			},
		},
		{
			name: "CancelOrder",
			call: func(ctx context.Context, c *procurement.ConsumerProcurementClient) error {
				_, err := c.CancelOrder(ctx, &procurementpb.CancelOrderRequest{
					Name:               orderName,
					CancellationPolicy: procurementpb.CancelOrderRequest_CANCELLATION_POLICY_CANCEL_AT_TERM_END,
				})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *procurement.ConsumerProcurementClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, client)
		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedNotImplemented(err):
			logf("%s returned NotImplemented (expected in staged emulation)", call.name)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func isToleratedNotImplemented(err error) bool {
	if err == nil {
		return false
	}

	if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.Unimplemented {
		return true
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close consumer procurement client: %v\n", err)
	}
}

func getenv(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
