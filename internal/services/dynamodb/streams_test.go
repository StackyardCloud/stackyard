package dynamodb

import (
	"errors"
	"testing"
	"time"
)

func TestShardIteratorExpiresAfterFifteenMinutes(t *testing.T) {
	clock := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	s := NewService()
	s.now = func() time.Time { return clock }
	if _, err := s.CreateTable("t", nil, []KeySchemaElement{{AttributeName: "pk", KeyType: "HASH"}}, ""); err != nil {
		t.Fatal(err)
	}
	desc, err := s.UpdateStreamSpecification("t", &StreamSpecification{StreamEnabled: true, StreamViewType: "NEW_IMAGE"}, true)
	if err != nil {
		t.Fatal(err)
	}
	stream, err := s.DescribeStream(desc.LatestStreamArn, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	iterator, err := s.GetShardIterator(desc.LatestStreamArn, stream.Shards[0].ShardID, "TRIM_HORIZON", "")
	if err != nil {
		t.Fatal(err)
	}

	clock = clock.Add(streamIteratorTTL)
	if _, _, err := s.GetRecords(iterator, 0); err != nil {
		t.Fatalf("an iterator exactly 15 minutes old is still valid: %v", err)
	}
	clock = clock.Add(time.Second)
	if _, _, err := s.GetRecords(iterator, 0); !errors.Is(err, ErrExpiredIterator) {
		t.Fatalf("expected ErrExpiredIterator, got %v", err)
	}
}
