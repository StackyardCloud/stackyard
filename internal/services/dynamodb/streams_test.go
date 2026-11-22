package dynamodb

import (
	"errors"
	"testing"
	"time"
)

func TestDynamoDBCreateTableRejectsBadStreamSpecificationFirst(t *testing.T) {
	s := NewService()
	_, err := s.CreateTable("t", nil, []KeySchemaElement{{AttributeName: "pk", KeyType: "HASH"}}, "", &StreamSpecification{StreamEnabled: true, StreamViewType: "ALL"})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
	if _, err := s.DescribeTable("t"); !errors.Is(err, ErrResourceNotFound) {
		t.Fatalf("a rejected stream specification must not create the table, got %v", err)
	}
}

func TestShardIteratorExpiresAfterFifteenMinutes(t *testing.T) {
	clock := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	s := NewService()
	s.now = func() time.Time { return clock }
	desc, err := s.CreateTable("t", nil, []KeySchemaElement{{AttributeName: "pk", KeyType: "HASH"}}, "", &StreamSpecification{StreamEnabled: true, StreamViewType: "NEW_IMAGE"})
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
