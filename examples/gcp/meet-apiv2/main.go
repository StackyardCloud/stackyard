package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	meet "cloud.google.com/go/apps/meet/apiv2"
	"cloud.google.com/go/apps/meet/apiv2/meetpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *meet.SpacesClient, *meet.ConferenceRecordsClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	spaceName := getenv("STACKYARD_GCP_MEET_SPACE", "spaces/team-space")
	conferenceRecordName := getenv("STACKYARD_GCP_MEET_CONFERENCE_RECORD", "conferenceRecords/team-conference")
	participantName := getenv("STACKYARD_GCP_MEET_PARTICIPANT", conferenceRecordName+"/participants/team-participant")
	participantSessionName := getenv("STACKYARD_GCP_MEET_PARTICIPANT_SESSION", participantName+"/participantSessions/session-1")
	recordingName := getenv("STACKYARD_GCP_MEET_RECORDING", conferenceRecordName+"/recordings/recording-1")
	transcriptName := getenv("STACKYARD_GCP_MEET_TRANSCRIPT", conferenceRecordName+"/transcripts/transcript-1")
	transcriptEntryName := getenv("STACKYARD_GCP_MEET_TRANSCRIPT_ENTRY", transcriptName+"/entries/entry-1")

	fmt.Printf("Stackyard GCP Google Meet apiv2 clients using %s\n", apiEndpoint)

	spacesClient, err := meet.NewSpacesRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create meet spaces client: %v", err)
	}
	defer closeClient("meet spaces", spacesClient.Close)

	conferenceRecordsClient, err := meet.NewConferenceRecordsRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create meet conference records client: %v", err)
	}
	defer closeClient("meet conference records", conferenceRecordsClient.Close)

	calls := []callSpec{
		{
			name: "CreateSpace",
			call: func(ctx context.Context, s *meet.SpacesClient, _ *meet.ConferenceRecordsClient) error {
				_, err := s.CreateSpace(ctx, &meetpb.CreateSpaceRequest{
					Space: &meetpb.Space{},
				})
				return err
			},
		},
		{
			name: "GetSpace",
			call: func(ctx context.Context, s *meet.SpacesClient, _ *meet.ConferenceRecordsClient) error {
				_, err := s.GetSpace(ctx, &meetpb.GetSpaceRequest{
					Name: spaceName,
				})
				return err
			},
		},
		{
			name: "UpdateSpace",
			call: func(ctx context.Context, s *meet.SpacesClient, _ *meet.ConferenceRecordsClient) error {
				_, err := s.UpdateSpace(ctx, &meetpb.UpdateSpaceRequest{
					Space: &meetpb.Space{
						Name: spaceName,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"config"},
					},
				})
				return err
			},
		},
		{
			name: "EndActiveConference",
			call: func(ctx context.Context, s *meet.SpacesClient, _ *meet.ConferenceRecordsClient) error {
				return s.EndActiveConference(ctx, &meetpb.EndActiveConferenceRequest{
					Name: spaceName,
				})
			},
		},
		{
			name: "ListConferenceRecords",
			call: func(ctx context.Context, _ *meet.SpacesClient, c *meet.ConferenceRecordsClient) error {
				it := c.ListConferenceRecords(ctx, &meetpb.ListConferenceRecordsRequest{
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
			name: "GetConferenceRecord",
			call: func(ctx context.Context, _ *meet.SpacesClient, c *meet.ConferenceRecordsClient) error {
				_, err := c.GetConferenceRecord(ctx, &meetpb.GetConferenceRecordRequest{
					Name: conferenceRecordName,
				})
				return err
			},
		},
		{
			name: "ListParticipants",
			call: func(ctx context.Context, _ *meet.SpacesClient, c *meet.ConferenceRecordsClient) error {
				it := c.ListParticipants(ctx, &meetpb.ListParticipantsRequest{
					Parent:   conferenceRecordName,
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
			name: "GetParticipant",
			call: func(ctx context.Context, _ *meet.SpacesClient, c *meet.ConferenceRecordsClient) error {
				_, err := c.GetParticipant(ctx, &meetpb.GetParticipantRequest{
					Name: participantName,
				})
				return err
			},
		},
		{
			name: "ListParticipantSessions",
			call: func(ctx context.Context, _ *meet.SpacesClient, c *meet.ConferenceRecordsClient) error {
				it := c.ListParticipantSessions(ctx, &meetpb.ListParticipantSessionsRequest{
					Parent:   participantName,
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
			name: "GetParticipantSession",
			call: func(ctx context.Context, _ *meet.SpacesClient, c *meet.ConferenceRecordsClient) error {
				_, err := c.GetParticipantSession(ctx, &meetpb.GetParticipantSessionRequest{
					Name: participantSessionName,
				})
				return err
			},
		},
		{
			name: "ListRecordings",
			call: func(ctx context.Context, _ *meet.SpacesClient, c *meet.ConferenceRecordsClient) error {
				it := c.ListRecordings(ctx, &meetpb.ListRecordingsRequest{
					Parent:   conferenceRecordName,
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
			name: "GetRecording",
			call: func(ctx context.Context, _ *meet.SpacesClient, c *meet.ConferenceRecordsClient) error {
				_, err := c.GetRecording(ctx, &meetpb.GetRecordingRequest{
					Name: recordingName,
				})
				return err
			},
		},
		{
			name: "ListTranscripts",
			call: func(ctx context.Context, _ *meet.SpacesClient, c *meet.ConferenceRecordsClient) error {
				it := c.ListTranscripts(ctx, &meetpb.ListTranscriptsRequest{
					Parent:   conferenceRecordName,
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
			name: "GetTranscript",
			call: func(ctx context.Context, _ *meet.SpacesClient, c *meet.ConferenceRecordsClient) error {
				_, err := c.GetTranscript(ctx, &meetpb.GetTranscriptRequest{
					Name: transcriptName,
				})
				return err
			},
		},
		{
			name: "ListTranscriptEntries",
			call: func(ctx context.Context, _ *meet.SpacesClient, c *meet.ConferenceRecordsClient) error {
				it := c.ListTranscriptEntries(ctx, &meetpb.ListTranscriptEntriesRequest{
					Parent:   transcriptName,
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
			name: "GetTranscriptEntry",
			call: func(ctx context.Context, _ *meet.SpacesClient, c *meet.ConferenceRecordsClient) error {
				_, err := c.GetTranscriptEntry(ctx, &meetpb.GetTranscriptEntryRequest{
					Name: transcriptEntryName,
				})
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, spacesClient, conferenceRecordsClient)
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

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", name, err)
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
