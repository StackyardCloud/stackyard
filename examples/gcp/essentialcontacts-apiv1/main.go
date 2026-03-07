package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	essentialcontacts "cloud.google.com/go/essentialcontacts/apiv1"
	"cloud.google.com/go/essentialcontacts/apiv1/essentialcontactspb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *essentialcontacts.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	contactID := getenv("STACKYARD_GCP_ESSENTIALCONTACTS_CONTACT_ID", "ops-primary")
	contactEmail := getenv("STACKYARD_GCP_ESSENTIALCONTACTS_EMAIL", "platform-oncall@example.com")

	parent := fmt.Sprintf("projects/%s", projectID)
	contactName := fmt.Sprintf("%s/contacts/%s", parent, contactID)
	activeContactName := contactName

	fmt.Printf("Stackyard GCP Essential Contacts apiv1 client using %s\n", apiEndpoint)

	client, err := essentialcontacts.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create essentialcontacts client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListContacts",
			call: func(ctx context.Context, c *essentialcontacts.Client) error {
				it := c.ListContacts(ctx, &essentialcontactspb.ListContactsRequest{
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
			name: "CreateContact",
			call: func(ctx context.Context, c *essentialcontacts.Client) error {
				resp, err := c.CreateContact(ctx, &essentialcontactspb.CreateContactRequest{
					Parent: parent,
					Contact: &essentialcontactspb.Contact{
						Email:       contactEmail,
						LanguageTag: "en-US",
						NotificationCategorySubscriptions: []essentialcontactspb.NotificationCategory{
							essentialcontactspb.NotificationCategory_SECURITY,
							essentialcontactspb.NotificationCategory_TECHNICAL,
						},
					},
				})
				if err == nil && resp.GetName() != "" {
					activeContactName = resp.GetName()
				}
				return err
			},
		},
		{
			name: "GetContact",
			call: func(ctx context.Context, c *essentialcontacts.Client) error {
				_, err := c.GetContact(ctx, &essentialcontactspb.GetContactRequest{
					Name: activeContactName,
				})
				return err
			},
		},
		{
			name: "UpdateContact",
			call: func(ctx context.Context, c *essentialcontacts.Client) error {
				_, err := c.UpdateContact(ctx, &essentialcontactspb.UpdateContactRequest{
					Contact: &essentialcontactspb.Contact{
						Name:        activeContactName,
						Email:       contactEmail,
						LanguageTag: "en-GB",
						NotificationCategorySubscriptions: []essentialcontactspb.NotificationCategory{
							essentialcontactspb.NotificationCategory_SECURITY,
							essentialcontactspb.NotificationCategory_PRODUCT_UPDATES,
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"language_tag", "notification_category_subscriptions"},
					},
				})
				return err
			},
		},
		{
			name: "ComputeContacts",
			call: func(ctx context.Context, c *essentialcontacts.Client) error {
				it := c.ComputeContacts(ctx, &essentialcontactspb.ComputeContactsRequest{
					Parent: parent,
					NotificationCategories: []essentialcontactspb.NotificationCategory{
						essentialcontactspb.NotificationCategory_SECURITY,
					},
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
			name: "SendTestMessage",
			call: func(ctx context.Context, c *essentialcontacts.Client) error {
				return c.SendTestMessage(ctx, &essentialcontactspb.SendTestMessageRequest{
					Resource:             parent,
					Contacts:             []string{activeContactName},
					NotificationCategory: essentialcontactspb.NotificationCategory_SECURITY,
				})
			},
		},
		{
			name: "DeleteContact",
			call: func(ctx context.Context, c *essentialcontacts.Client) error {
				return c.DeleteContact(ctx, &essentialcontactspb.DeleteContactRequest{
					Name: activeContactName,
				})
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
		fmt.Fprintf(os.Stderr, "warning: close essentialcontacts client: %v\n", err)
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
