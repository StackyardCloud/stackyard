package server

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
)

func TestSQSStage8InvalidReceiptHandle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := []byte("Action=CreateQueue&QueueName=stage8-receipt")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", create, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	del := []byte("Action=DeleteMessage&QueueUrl=" + ts.URL + "/123456789012/stage8-receipt&ReceiptHandle=bad-handle")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", del, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "<Code>ReceiptHandleIsInvalid</Code>") {
		t.Fatalf("expected ReceiptHandleIsInvalid, got %s", body)
	}
}

func TestSQSStage8ReceiveMessageMaxNumberInvalid(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := []byte("Action=CreateQueue&QueueName=stage8-max")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", create, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	recv := []byte("Action=ReceiveMessage&QueueUrl=" + ts.URL + "/123456789012/stage8-max&MaxNumberOfMessages=11")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", recv, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestSQSStage8SendMessageDelayInvalid(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := []byte("Action=CreateQueue&QueueName=stage8-delay")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", create, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	send := []byte("Action=SendMessage&QueueUrl=" + ts.URL + "/123456789012/stage8-delay&MessageBody=hi&DelaySeconds=901")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", send, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestSQSStage8BatchDuplicateIds(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := []byte("Action=CreateQueue&QueueName=stage8-batch")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", create, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	body := "Action=SendMessageBatch&QueueUrl=" + ts.URL + "/123456789012/stage8-batch" +
		"&SendMessageBatchRequestEntry.1.Id=dup" +
		"&SendMessageBatchRequestEntry.1.MessageBody=one" +
		"&SendMessageBatchRequestEntry.2.Id=dup" +
		"&SendMessageBatchRequestEntry.2.MessageBody=two"
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusBadRequest)
	data := string(mustBody(t, resp))
	if !strings.Contains(data, "<Code>BatchEntryIdsNotDistinct</Code>") {
		t.Fatalf("expected BatchEntryIdsNotDistinct, got %s", data)
	}
}

func TestSQSStage8InvalidNextToken(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := []byte("Action=CreateQueue&QueueName=stage8-move")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", create, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	sourceArn := "arn:aws:sqs:us-east-1:123456789012:stage8-move"
	list := []byte("Action=ListMessageMoveTasks&SourceArn=" + sourceArn + "&NextToken=abc")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", list, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestSQSStage8InvalidRedrivePolicy(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := "Action=CreateQueue&QueueName=stage8-redrive" +
		"&Attribute.1.Name=RedrivePolicy" +
		"&Attribute.1.Value=%7B"
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestSQSStage8MessageAttributesAndSystemAttributes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := []byte("Action=CreateQueue&QueueName=stage8-attrs")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", create, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	sendBody := "Action=SendMessage&QueueUrl=" + ts.URL + "/123456789012/stage8-attrs&MessageBody=hello" +
		"&MessageAttribute.1.Name=attr1&MessageAttribute.1.Value.DataType=String&MessageAttribute.1.Value.StringValue=hi" +
		"&MessageAttribute.2.Name=num&MessageAttribute.2.Value.DataType=Number&MessageAttribute.2.Value.StringValue=42" +
		"&MessageSystemAttribute.1.Name=AWSTraceHeader&MessageSystemAttribute.1.Value.DataType=String&MessageSystemAttribute.1.Value.StringValue=Root%3D1-abcdef"
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(sendBody), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	recv := []byte("Action=ReceiveMessage&QueueUrl=" + ts.URL + "/123456789012/stage8-attrs&AttributeName=All&MessageAttributeName=All")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", recv, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "<Name>attr1</Name>") || !strings.Contains(body, "<DataType>String</DataType>") {
		t.Fatalf("expected attr1 in message attributes, got %s", body)
	}
	if !strings.Contains(body, "<Name>AWSTraceHeader</Name>") {
		t.Fatalf("expected AWSTraceHeader in attributes, got %s", body)
	}
	if !strings.Contains(body, "<Name>SenderId</Name>") {
		t.Fatalf("expected SenderId in attributes, got %s", body)
	}
	if !strings.Contains(body, "<Name>ApproximateFirstReceiveTimestamp</Name>") {
		t.Fatalf("expected ApproximateFirstReceiveTimestamp in attributes, got %s", body)
	}

	expectedMsgMD5 := md5OfAttrs(map[string]attrValue{
		"attr1": {dataType: "String", stringValue: "hi"},
		"num":   {dataType: "Number", stringValue: "42"},
	})
	expectedSysMD5 := md5OfAttrs(map[string]attrValue{
		"AWSTraceHeader": {dataType: "String", stringValue: "Root=1-abcdef"},
	})
	if !strings.Contains(body, "<MD5OfMessageAttributes>"+expectedMsgMD5+"</MD5OfMessageAttributes>") {
		t.Fatalf("expected MD5OfMessageAttributes %s, got %s", expectedMsgMD5, body)
	}
	if !strings.Contains(body, "<MD5OfMessageSystemAttributes>"+expectedSysMD5+"</MD5OfMessageSystemAttributes>") {
		t.Fatalf("expected MD5OfMessageSystemAttributes %s, got %s", expectedSysMD5, body)
	}
}

func TestSQSStage8FifoSequenceAndGroup(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := []byte("Action=CreateQueue&QueueName=fifo-attrs.fifo&Attribute.1.Name=FifoQueue&Attribute.1.Value=true")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", create, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)

	send := []byte("Action=SendMessage&QueueUrl=" + ts.URL + "/123456789012/fifo-attrs.fifo&MessageBody=hello&MessageGroupId=group-1&MessageDeduplicationId=dedup-1")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", send, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "<SequenceNumber>") {
		t.Fatalf("expected SequenceNumber in SendMessage response, got %s", body)
	}

	recv := []byte("Action=ReceiveMessage&QueueUrl=" + ts.URL + "/123456789012/fifo-attrs.fifo&AttributeName=All")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", recv, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sqs")
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if !strings.Contains(body, "<Name>SequenceNumber</Name>") {
		t.Fatalf("expected SequenceNumber attribute, got %s", body)
	}
	if !strings.Contains(body, "<Name>MessageGroupId</Name>") {
		t.Fatalf("expected MessageGroupId attribute, got %s", body)
	}
	if !strings.Contains(body, "<Name>MessageDeduplicationId</Name>") {
		t.Fatalf("expected MessageDeduplicationId attribute, got %s", body)
	}
}

type attrValue struct {
	dataType    string
	stringValue string
	binaryValue string
}

func md5OfAttrs(attrs map[string]attrValue) string {
	names := make([]string, 0, len(attrs))
	for name := range attrs {
		names = append(names, name)
	}
	sort.Strings(names)
	buf := &bytes.Buffer{}
	for _, name := range names {
		attr := attrs[name]
		writeMD5StringTest(buf, name)
		writeMD5StringTest(buf, attr.dataType)
		baseType := attr.dataType
		if strings.Contains(baseType, ".") {
			baseType = strings.Split(baseType, ".")[0]
		}
		switch baseType {
		case "String", "Number":
			buf.WriteByte(1)
			writeMD5StringTest(buf, attr.stringValue)
		case "Binary":
			buf.WriteByte(2)
			data := []byte(attr.binaryValue)
			decoded, err := base64.StdEncoding.DecodeString(attr.binaryValue)
			if err == nil {
				data = decoded
			}
			writeMD5BytesTest(buf, data)
		default:
			buf.WriteByte(1)
			writeMD5StringTest(buf, attr.stringValue)
		}
	}
	sum := md5.Sum(buf.Bytes())
	return hex.EncodeToString(sum[:])
}

func writeMD5StringTest(buf *bytes.Buffer, value string) {
	_ = binary.Write(buf, binary.BigEndian, uint32(len(value)))
	_, _ = buf.WriteString(value)
}

func writeMD5BytesTest(buf *bytes.Buffer, value []byte) {
	_ = binary.Write(buf, binary.BigEndian, uint32(len(value)))
	_, _ = buf.Write(value)
}
