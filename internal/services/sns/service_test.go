package sns

import "testing"

func TestServiceTopicAndPublish(t *testing.T) {
	svc := NewService()
	topic, err := svc.CreateTopic("demo", map[string]string{"DisplayName": "demo"})
	if err != nil {
		t.Fatalf("create topic: %v", err)
	}
	if topic.ARN == "" {
		t.Fatalf("expected topic ARN")
	}
	if err := svc.SetTopicAttributes(topic.ARN, map[string]string{"DisplayName": "demo-2"}); err != nil {
		t.Fatalf("set topic attributes: %v", err)
	}
	if _, err := svc.Publish(PublishInput{TopicARN: topic.ARN, Message: "hello"}); err != nil {
		t.Fatalf("publish: %v", err)
	}
}

func TestServiceSubscriptions(t *testing.T) {
	svc := NewService()
	topic, err := svc.CreateTopic("sub-demo", nil)
	if err != nil {
		t.Fatalf("create topic: %v", err)
	}
	sub, err := svc.Subscribe(topic.ARN, "sqs", "arn:aws:sqs:us-east-1:123456789012:queue", nil, false)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if sub.ARN == "" {
		t.Fatalf("expected subscription arn")
	}
	if err := svc.Unsubscribe(sub.ARN); err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}
}

func TestServiceSMSSandbox(t *testing.T) {
	svc := NewService()
	if _, err := svc.CreateSMSSandboxPhoneNumber("+15555550100"); err != nil {
		t.Fatalf("create sandbox phone: %v", err)
	}
	if err := svc.VerifySMSSandboxPhoneNumber("+15555550100"); err != nil {
		t.Fatalf("verify sandbox phone: %v", err)
	}
	if !svc.IsSMSSandboxEnabled() {
		t.Fatalf("expected sandbox enabled")
	}
	if err := svc.DeleteSMSSandboxPhoneNumber("+15555550100"); err != nil {
		t.Fatalf("delete sandbox phone: %v", err)
	}
}
