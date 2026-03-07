package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	ordertracking "cloud.google.com/go/shopping/merchant/ordertracking/apiv1"
	"cloud.google.com/go/shopping/merchant/ordertracking/apiv1/ordertrackingpb"
	"google.golang.org/api/option"
	datetimepb "google.golang.org/genproto/googleapis/type/datetime"
)

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	accountID := getenv("STACKYARD_GCP_MERCHANT_ORDERTRACKING_ACCOUNT_ID", "123456")
	orderID := getenv("STACKYARD_GCP_MERCHANT_ORDERTRACKING_ORDER_ID", "ORDER-1001")
	parent := fmt.Sprintf("accounts/%s", accountID)

	fmt.Printf("Stackyard GCP Shopping Merchant Order Tracking shopping/merchant/ordertracking/apiv1 client using %s\n", apiEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "shopping-merchant-ordertracking-apiv1",
		},
	}

	client, err := ordertracking.NewOrderTrackingSignalsRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create shopping merchant order tracking client: %v", err)
	}
	defer closeClient("shopping merchant order tracking", client.Close)

	resp, err := client.CreateOrderTrackingSignal(ctx, &ordertrackingpb.CreateOrderTrackingSignalRequest{
		Parent: parent,
		OrderTrackingSignal: &ordertrackingpb.OrderTrackingSignal{
			OrderCreatedTime: &datetimepb.DateTime{
				Year:    2026,
				Month:   1,
				Day:     2,
				Hours:   3,
				Minutes: 4,
				Seconds: 5,
			},
			OrderId: orderID,
			ShippingInfo: []*ordertrackingpb.OrderTrackingSignal_ShippingInfo{
				{
					ShipmentId:       "SHIP-1001",
					TrackingId:       "TRACK-1001",
					Carrier:          "UPS",
					ShippingStatus:   ordertrackingpb.OrderTrackingSignal_ShippingInfo_SHIPPED,
					OriginPostalCode: "94043",
					OriginRegionCode: "US",
				},
			},
			LineItems: []*ordertrackingpb.OrderTrackingSignal_LineItemDetails{
				{
					LineItemId: "line-1",
					ProductId:  "online:en:US:offer-1001",
					Quantity:   2,
				},
			},
			ShipmentLineItemMapping: []*ordertrackingpb.OrderTrackingSignal_ShipmentLineItemMapping{
				{
					ShipmentId: "SHIP-1001",
					LineItemId: "line-1",
					Quantity:   2,
				},
			},
			DeliveryPostalCode: "94043",
			DeliveryRegionCode: "US",
		},
	})
	if err != nil {
		exitf("CreateOrderTrackingSignal failed: %v", err)
	}
	if resp.GetOrderTrackingSignalId() <= 0 {
		exitf("CreateOrderTrackingSignal returned invalid orderTrackingSignalId: %d", resp.GetOrderTrackingSignalId())
	}
	fmt.Printf("CreateOrderTrackingSignal succeeded: id=%d orderId=%s\n", resp.GetOrderTrackingSignalId(), resp.GetOrderId())
	fmt.Println("Done.")
}

func waitForStackyardReady(ctx context.Context, endpoint string) error {
	target := strings.TrimRight(endpoint, "/") + "/v1/projects/stackyard/locations/us-central1/shopping_merchant_ordertracking/sample?stackyard_contract_probe=1"
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(30 * time.Second)

	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode < http.StatusInternalServerError {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for stackyard at %s", target)
}

type stackyardHeaderTransport struct {
	base        http.RoundTripper
	serviceName string
}

func (t stackyardHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	if strings.TrimSpace(t.serviceName) != "" {
		clone.Header.Set("X-Stackyard-GCP-Service", t.serviceName)
	}
	return base.RoundTrip(clone)
}

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to close %s client: %v\n", name, err)
	}
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
