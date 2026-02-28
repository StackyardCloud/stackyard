package sqs

import "testing"

func TestQueueSendReceive(t *testing.T) {
	svc := NewService()

	if _, err := svc.CreateQueue("jobs", nil); err != nil {
		t.Fatalf("create queue: %v", err)
	}

	if _, err := svc.SendMessage("jobs", "hello", nil, nil); err != nil {
		t.Fatalf("send message: %v", err)
	}
	if _, err := svc.SendMessage("jobs", "world", nil, nil); err != nil {
		t.Fatalf("send message: %v", err)
	}

	msgs, err := svc.ReceiveMessages("jobs", 2)
	if err != nil {
		t.Fatalf("receive messages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Body != "hello" || msgs[1].Body != "world" {
		t.Fatalf("messages out of order: %#v", msgs)
	}
}

func TestQueuePurge(t *testing.T) {
	svc := NewService()

	if _, err := svc.CreateQueue("purge", nil); err != nil {
		t.Fatalf("create queue: %v", err)
	}
	if _, err := svc.SendMessage("purge", "one", nil, nil); err != nil {
		t.Fatalf("send message: %v", err)
	}
	if _, err := svc.SendMessage("purge", "two", nil, nil); err != nil {
		t.Fatalf("send message: %v", err)
	}
	if err := svc.PurgeQueue("purge"); err != nil {
		t.Fatalf("purge queue: %v", err)
	}
	msgs, err := svc.ReceiveMessages("purge", 10)
	if err != nil {
		t.Fatalf("receive messages: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages after purge, got %d", len(msgs))
	}
}
