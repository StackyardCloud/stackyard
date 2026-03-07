package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	chat "cloud.google.com/go/chat/apiv1"
	"cloud.google.com/go/chat/apiv1/chatpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *chat.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	spaceName := getenv("STACKYARD_GCP_CHAT_SPACE", "spaces/team-space")
	messageName := getenv("STACKYARD_GCP_CHAT_MESSAGE", spaceName+"/messages/message-1")
	membershipName := getenv("STACKYARD_GCP_CHAT_MEMBERSHIP", spaceName+"/members/user-1")
	reactionName := getenv("STACKYARD_GCP_CHAT_REACTION", messageName+"/reactions/reaction-1")
	customEmojiName := getenv("STACKYARD_GCP_CHAT_CUSTOM_EMOJI", "customEmojis/emoji-1")
	spaceReadStateName := getenv("STACKYARD_GCP_CHAT_SPACE_READ_STATE", "users/me/"+spaceName+"/spaceReadState")
	threadReadStateName := getenv("STACKYARD_GCP_CHAT_THREAD_READ_STATE", "users/me/"+spaceName+"/threads/thread-1/threadReadState")
	spaceNotificationSettingName := getenv("STACKYARD_GCP_CHAT_SPACE_NOTIFICATION_SETTING", "users/me/"+spaceName+"/spaceNotificationSetting")
	spaceEventName := getenv("STACKYARD_GCP_CHAT_SPACE_EVENT", spaceName+"/spaceEvents/event-1")
	attachmentName := getenv("STACKYARD_GCP_CHAT_ATTACHMENT", messageName+"/attachments/attachment-1")

	fmt.Printf("Stackyard GCP Google Chat apiv1 client using %s\n", apiEndpoint)

	client, err := chat.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create chat client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListSpaces",
			call: func(ctx context.Context, c *chat.Client) error {
				it := c.ListSpaces(ctx, &chatpb.ListSpacesRequest{
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
			name: "GetSpace",
			call: func(ctx context.Context, c *chat.Client) error {
				_, err := c.GetSpace(ctx, &chatpb.GetSpaceRequest{
					Name: spaceName,
				})
				return err
			},
		},
		{
			name: "CreateMessage",
			call: func(ctx context.Context, c *chat.Client) error {
				_, err := c.CreateMessage(ctx, &chatpb.CreateMessageRequest{
					Parent: spaceName,
					Message: &chatpb.Message{
						Text: "hello from stackyard",
					},
				})
				return err
			},
		},
		{
			name: "ListMessages",
			call: func(ctx context.Context, c *chat.Client) error {
				it := c.ListMessages(ctx, &chatpb.ListMessagesRequest{
					Parent:   spaceName,
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
			name: "GetMessage",
			call: func(ctx context.Context, c *chat.Client) error {
				_, err := c.GetMessage(ctx, &chatpb.GetMessageRequest{
					Name: messageName,
				})
				return err
			},
		},
		{
			name: "UpdateMessage",
			call: func(ctx context.Context, c *chat.Client) error {
				_, err := c.UpdateMessage(ctx, &chatpb.UpdateMessageRequest{
					Message: &chatpb.Message{
						Name: messageName,
						Text: "updated by stackyard",
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"text"},
					},
				})
				return err
			},
		},
		{
			name: "ListMemberships",
			call: func(ctx context.Context, c *chat.Client) error {
				it := c.ListMemberships(ctx, &chatpb.ListMembershipsRequest{
					Parent:   spaceName,
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
			name: "GetMembership",
			call: func(ctx context.Context, c *chat.Client) error {
				_, err := c.GetMembership(ctx, &chatpb.GetMembershipRequest{
					Name: membershipName,
				})
				return err
			},
		},
		{
			name: "ListReactions",
			call: func(ctx context.Context, c *chat.Client) error {
				it := c.ListReactions(ctx, &chatpb.ListReactionsRequest{
					Parent:   messageName,
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
			name: "CreateReaction",
			call: func(ctx context.Context, c *chat.Client) error {
				_, err := c.CreateReaction(ctx, &chatpb.CreateReactionRequest{
					Parent: messageName,
					Reaction: &chatpb.Reaction{
						User: &chatpb.User{
							Name: "users/me",
						},
						Emoji: &chatpb.Emoji{
							Content: &chatpb.Emoji_Unicode{
								Unicode: "\U0001f44d",
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "DeleteReaction",
			call: func(ctx context.Context, c *chat.Client) error {
				return c.DeleteReaction(ctx, &chatpb.DeleteReactionRequest{
					Name: reactionName,
				})
			},
		},
		{
			name: "GetSpaceReadState",
			call: func(ctx context.Context, c *chat.Client) error {
				_, err := c.GetSpaceReadState(ctx, &chatpb.GetSpaceReadStateRequest{
					Name: spaceReadStateName,
				})
				return err
			},
		},
		{
			name: "GetThreadReadState",
			call: func(ctx context.Context, c *chat.Client) error {
				_, err := c.GetThreadReadState(ctx, &chatpb.GetThreadReadStateRequest{
					Name: threadReadStateName,
				})
				return err
			},
		},
		{
			name: "GetSpaceNotificationSetting",
			call: func(ctx context.Context, c *chat.Client) error {
				_, err := c.GetSpaceNotificationSetting(ctx, &chatpb.GetSpaceNotificationSettingRequest{
					Name: spaceNotificationSettingName,
				})
				return err
			},
		},
		{
			name: "ListSpaceEvents",
			call: func(ctx context.Context, c *chat.Client) error {
				it := c.ListSpaceEvents(ctx, &chatpb.ListSpaceEventsRequest{
					Parent:   spaceName,
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
			name: "GetSpaceEvent",
			call: func(ctx context.Context, c *chat.Client) error {
				_, err := c.GetSpaceEvent(ctx, &chatpb.GetSpaceEventRequest{
					Name: spaceEventName,
				})
				return err
			},
		},
		{
			name: "ListCustomEmojis",
			call: func(ctx context.Context, c *chat.Client) error {
				it := c.ListCustomEmojis(ctx, &chatpb.ListCustomEmojisRequest{
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
			name: "GetCustomEmoji",
			call: func(ctx context.Context, c *chat.Client) error {
				_, err := c.GetCustomEmoji(ctx, &chatpb.GetCustomEmojiRequest{
					Name: customEmojiName,
				})
				return err
			},
		},
		{
			name: "UploadAttachment",
			call: func(ctx context.Context, c *chat.Client) error {
				_, err := c.UploadAttachment(ctx, &chatpb.UploadAttachmentRequest{
					Parent:   spaceName,
					Filename: "evidence.json",
				})
				return err
			},
		},
		{
			name: "GetAttachment",
			call: func(ctx context.Context, c *chat.Client) error {
				_, err := c.GetAttachment(ctx, &chatpb.GetAttachmentRequest{
					Name: attachmentName,
				})
				return err
			},
		},
		{
			name: "SearchSpaces",
			call: func(ctx context.Context, c *chat.Client) error {
				it := c.SearchSpaces(ctx, &chatpb.SearchSpacesRequest{
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
			name: "FindDirectMessage",
			call: func(ctx context.Context, c *chat.Client) error {
				_, err := c.FindDirectMessage(ctx, &chatpb.FindDirectMessageRequest{
					Name: "users/me",
				})
				return err
			},
		},
		{
			name: "SetUpSpace",
			call: func(ctx context.Context, c *chat.Client) error {
				_, err := c.SetUpSpace(ctx, &chatpb.SetUpSpaceRequest{
					Space: &chatpb.Space{
						DisplayName: "Team Space",
					},
				})
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
		fmt.Fprintf(os.Stderr, "warning: close chat client: %v\n", err)
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
