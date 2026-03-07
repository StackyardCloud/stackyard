package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	notifications "cloud.google.com/go/shopping/merchant/notifications/apiv1"
	"cloud.google.com/go/shopping/merchant/notifications/apiv1/notificationspb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	accountID := getenv("STACKYARD_GCP_MERCHANT_NOTIFICATIONS_ACCOUNT_ID", "123456")
	createCallbackURI := getenv("STACKYARD_GCP_MERCHANT_NOTIFICATIONS_CREATE_CALLBACK_URI", "https://example.com/hooks/merchant-notifications")
	updateCallbackURI := getenv("STACKYARD_GCP_MERCHANT_NOTIFICATIONS_UPDATE_CALLBACK_URI", "https://example.com/hooks/merchant-notifications-updated")
	parent := fmt.Sprintf("accounts/%s", accountID)

	fmt.Printf("Stackyard GCP Shopping Merchant Notifications shopping/merchant/notifications/apiv1 client using %s\n", apiEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint, parent); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "shopping-merchant-notifications-apiv1",
		},
	}

	client, err := notifications.NewNotificationsApiRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create shopping merchant notifications client: %v", err)
	}
	defer closeClient("shopping merchant notifications", client.Close)

	subscriptionName := ""

	calls := []callSpec{
		{
			name: "CreateNotificationSubscription",
			call: func(ctx context.Context) error {
				resp, err := client.CreateNotificationSubscription(ctx, &notificationspb.CreateNotificationSubscriptionRequest{
					Parent: parent,
					NotificationSubscription: &notificationspb.NotificationSubscription{
						RegisteredEvent: notificationspb.NotificationSubscription_PRODUCT_STATUS_CHANGE,
						CallBackUri:     createCallbackURI,
						InterestedIn: &notificationspb.NotificationSubscription_AllManagedAccounts{
							AllManagedAccounts: true,
						},
					},
				})
				if err != nil {
					return err
				}
				subscriptionName = strings.TrimSpace(resp.GetName())
				if subscriptionName == "" {
					return fmt.Errorf("create response returned empty notification subscription name")
				}
				return nil
			},
		},
		{
			name: "GetNotificationSubscription",
			call: func(ctx context.Context) error {
				_, err := client.GetNotificationSubscription(ctx, &notificationspb.GetNotificationSubscriptionRequest{
					Name: subscriptionName,
				})
				return err
			},
		},
		{
			name: "ListNotificationSubscriptions",
			call: func(ctx context.Context) error {
				it := client.ListNotificationSubscriptions(ctx, &notificationspb.ListNotificationSubscriptionsRequest{
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
			name: "UpdateNotificationSubscription",
			call: func(ctx context.Context) error {
				_, err := client.UpdateNotificationSubscription(ctx, &notificationspb.UpdateNotificationSubscriptionRequest{
					NotificationSubscription: &notificationspb.NotificationSubscription{
						Name:            subscriptionName,
						RegisteredEvent: notificationspb.NotificationSubscription_PRODUCT_STATUS_CHANGE,
						CallBackUri:     updateCallbackURI,
						InterestedIn: &notificationspb.NotificationSubscription_AllManagedAccounts{
							AllManagedAccounts: true,
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"callBackUri"},
					},
				})
				return err
			},
		},
		{
			name: "GetNotificationSubscriptionHealthMetrics",
			call: func(ctx context.Context) error {
				_, err := client.GetNotificationSubscriptionHealthMetrics(ctx, &notificationspb.GetNotificationSubscriptionHealthMetricsRequest{
					Name: subscriptionName,
				})
				return err
			},
		},
		{
			name: "DeleteNotificationSubscription",
			call: func(ctx context.Context) error {
				return client.DeleteNotificationSubscription(ctx, &notificationspb.DeleteNotificationSubscriptionRequest{
					Name: subscriptionName,
				})
			},
		},
	}

	for _, spec := range calls {
		if err := spec.call(ctx); err != nil {
			exitf("%s failed: %v", spec.name, err)
		}
		fmt.Printf("%s succeeded\n", spec.name)
	}

	fmt.Println("Done.")
}

func waitForStackyardReady(ctx context.Context, endpoint, parent string) error {
	target := strings.TrimRight(endpoint, "/") + "/notifications/v1/" + parent + "/notificationsubscriptions?pageSize=1"
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
