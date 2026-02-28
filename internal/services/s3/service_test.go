package s3

import "testing"

func TestBucketAndObjectFlow(t *testing.T) {
	svc := NewService()

	if _, err := svc.CreateBucket("docs"); err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	_, err := svc.PutObject("docs", "hello.txt", "text/plain", []byte("hello"), map[string]string{"owner": "stackyard"}, "", "")
	if err != nil {
		t.Fatalf("put object: %v", err)
	}

	obj, err := svc.GetObject("docs", "hello.txt")
	if err != nil {
		t.Fatalf("get object: %v", err)
	}
	if string(obj.Body) != "hello" {
		t.Fatalf("unexpected body: %q", string(obj.Body))
	}

	objects, err := svc.ListObjects("docs", "hello")
	if err != nil {
		t.Fatalf("list objects: %v", err)
	}
	if len(objects) != 1 || objects[0].Key != "hello.txt" {
		t.Fatalf("unexpected objects: %#v", objects)
	}
}

func TestBucketLoggingConfig(t *testing.T) {
	svc := NewService()

	if _, err := svc.CreateBucket("logs"); err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	cfg, err := svc.GetBucketLogging("logs")
	if err != nil {
		t.Fatalf("get logging: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil logging config, got %#v", cfg)
	}

	err = svc.SetBucketLogging("logs", &LoggingConfiguration{
		TargetBucket: "logs",
		TargetPrefix: "access/",
	})
	if err != nil {
		t.Fatalf("set logging: %v", err)
	}

	cfg, err = svc.GetBucketLogging("logs")
	if err != nil {
		t.Fatalf("get logging after set: %v", err)
	}
	if cfg == nil || cfg.TargetBucket != "logs" || cfg.TargetPrefix != "access/" {
		t.Fatalf("unexpected logging config: %#v", cfg)
	}

	if err := svc.DeleteBucketLogging("logs"); err != nil {
		t.Fatalf("delete logging: %v", err)
	}
	cfg, err = svc.GetBucketLogging("logs")
	if err != nil {
		t.Fatalf("get logging after delete: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil logging config after delete, got %#v", cfg)
	}
}
