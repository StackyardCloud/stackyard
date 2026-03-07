package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	topicName := getenv("STACKYARD_TOPIC", "demo-sns")
	smsNumber := getenv("STACKYARD_SMS_NUMBER", "+15555550123")
	smsOTP := getenv("STACKYARD_SMS_OTP", "000000")
	subEndpoint := getenv("STACKYARD_SUB_ENDPOINT", "demo@example.com")

	ctx := context.Background()
	client := newSNSClient(ctx, endpoint)

	fmt.Printf("Stackyard SNS advanced client using %s\n", endpoint)

	topicArn, err := createTopic(ctx, client, topicName)
	if err != nil {
		exitf("create topic: %v", err)
	}
	logf("created topic: %s", topicArn)

	if err := setTopicAttributes(ctx, client, topicArn, map[string]string{
		"DisplayName": "Stackyard Demo",
	}); err != nil {
		exitf("set topic attributes: %v", err)
	}

	attrs, err := getTopicAttributes(ctx, client, topicArn)
	if err != nil {
		exitf("get topic attributes: %v", err)
	}
	logf("topic attributes: %d", len(attrs))

	policy := `{"Version":"2012-10-17","Statement":[]}`
	if err := putDataProtectionPolicy(ctx, client, topicArn, policy); err != nil {
		exitf("put data protection policy: %v", err)
	}

	policyOut, err := getDataProtectionPolicy(ctx, client, topicArn)
	if err != nil {
		exitf("get data protection policy: %v", err)
	}
	logf("data protection policy length: %d", len(policyOut))

	if err := tagResource(ctx, client, topicArn, map[string]string{"env": "dev", "team": "stackyard"}); err != nil {
		exitf("tag resource: %v", err)
	}

	tags, err := listTags(ctx, client, topicArn)
	if err != nil {
		exitf("list tags: %v", err)
	}
	logf("tags: %d", len(tags))

	subArn, err := subscribe(ctx, client, topicArn, "email", subEndpoint, true)
	if err != nil {
		exitf("subscribe: %v", err)
	}
	logf("subscription arn: %s", subArn)

	if _, err := getSubscriptionAttributes(ctx, client, subArn); err != nil {
		exitf("get subscription attributes: %v", err)
	}

	if err := setSubscriptionAttribute(ctx, client, subArn, "RawMessageDelivery", "true"); err != nil {
		exitf("set subscription attribute: %v", err)
	}

	if _, err := listSubscriptions(ctx, client); err != nil {
		exitf("list subscriptions: %v", err)
	}

	if _, err := listSubscriptionsByTopic(ctx, client, topicArn); err != nil {
		exitf("list subscriptions by topic: %v", err)
	}

	if err := publishMessage(ctx, client, topicArn, "hello from stackyard sns advanced example"); err != nil {
		exitf("publish message: %v", err)
	}

	if err := publishBatch(ctx, client, topicArn); err != nil {
		exitf("publish batch: %v", err)
	}

	appArn, err := createPlatformApplication(ctx, client, "demo-app", "APNS")
	if err != nil {
		exitf("create platform app: %v", err)
	}
	logf("platform app arn: %s", appArn)

	if err := setPlatformAttributes(ctx, client, appArn, map[string]string{"PlatformCredential": "demo"}); err != nil {
		exitf("set platform attributes: %v", err)
	}

	if _, err := getPlatformAttributes(ctx, client, appArn); err != nil {
		exitf("get platform attributes: %v", err)
	}

	if _, err := listPlatformApplications(ctx, client); err != nil {
		exitf("list platform applications: %v", err)
	}

	endpointArn, err := createPlatformEndpoint(ctx, client, appArn, "token-1", "demo-user")
	if err != nil {
		exitf("create platform endpoint: %v", err)
	}
	logf("platform endpoint arn: %s", endpointArn)

	if err := setEndpointAttributes(ctx, client, endpointArn, map[string]string{"Enabled": "true"}); err != nil {
		exitf("set endpoint attributes: %v", err)
	}

	if _, err := getEndpointAttributes(ctx, client, endpointArn); err != nil {
		exitf("get endpoint attributes: %v", err)
	}

	if _, err := listEndpointsByPlatform(ctx, client, appArn); err != nil {
		exitf("list endpoints by platform: %v", err)
	}

	if err := createSMSSandboxPhoneNumber(ctx, client, smsNumber); err != nil {
		exitf("create sms sandbox phone: %v", err)
	}

	if err := verifySMSSandboxPhoneNumber(ctx, client, smsNumber, smsOTP); err != nil {
		exitf("verify sms sandbox phone: %v", err)
	}

	if _, err := listSMSSandboxPhoneNumbers(ctx, client); err != nil {
		exitf("list sms sandbox phone numbers: %v", err)
	}

	if _, err := getSMSSandboxStatus(ctx, client); err != nil {
		exitf("get sms sandbox status: %v", err)
	}

	if err := setSMSAttributes(ctx, client, map[string]string{"DefaultSenderID": "Stackyard"}); err != nil {
		exitf("set sms attributes: %v", err)
	}

	if _, err := getSMSAttributes(ctx, client); err != nil {
		exitf("get sms attributes: %v", err)
	}

	if _, err := checkIfPhoneNumberIsOptedOut(ctx, client, smsNumber); err != nil {
		exitf("check opted out: %v", err)
	}

	if _, err := listPhoneNumbersOptedOut(ctx, client); err != nil {
		exitf("list opted out numbers: %v", err)
	}

	if err := deleteSMSSandboxPhoneNumber(ctx, client, smsNumber); err != nil {
		exitf("delete sms sandbox phone: %v", err)
	}

	if err := deleteEndpoint(ctx, client, endpointArn); err != nil {
		exitf("delete endpoint: %v", err)
	}

	if err := deletePlatformApplication(ctx, client, appArn); err != nil {
		exitf("delete platform application: %v", err)
	}

	if err := unsubscribe(ctx, client, subArn); err != nil {
		exitf("unsubscribe: %v", err)
	}

	if err := deleteTopic(ctx, client, topicArn); err != nil {
		exitf("delete topic: %v", err)
	}

	fmt.Println("Done.")
}

func newSNSClient(ctx context.Context, endpoint string) *sns.Client {
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, _ ...any) (aws.Endpoint, error) {
		if service == sns.ServiceID {
			return aws.Endpoint{
				URL:               endpoint,
				SigningRegion:     region,
				HostnameImmutable: true,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(getenv("AWS_REGION", "us-east-1")),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"),
			getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
			"",
		)),
		config.WithEndpointResolverWithOptions(resolver),
	)
	if err != nil {
		exitf("load aws config: %v", err)
	}

	return sns.NewFromConfig(cfg)
}

func createTopic(ctx context.Context, client *sns.Client, name string) (string, error) {
	resp, err := client.CreateTopic(ctx, &sns.CreateTopicInput{Name: aws.String(name)})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.TopicArn), nil
}

func deleteTopic(ctx context.Context, client *sns.Client, arn string) error {
	_, err := client.DeleteTopic(ctx, &sns.DeleteTopicInput{TopicArn: aws.String(arn)})
	return err
}

func setTopicAttributes(ctx context.Context, client *sns.Client, arn string, attrs map[string]string) error {
	for key, val := range attrs {
		_, err := client.SetTopicAttributes(ctx, &sns.SetTopicAttributesInput{
			TopicArn:       aws.String(arn),
			AttributeName:  aws.String(key),
			AttributeValue: aws.String(val),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func getTopicAttributes(ctx context.Context, client *sns.Client, arn string) (map[string]string, error) {
	resp, err := client.GetTopicAttributes(ctx, &sns.GetTopicAttributesInput{TopicArn: aws.String(arn)})
	if err != nil {
		return nil, err
	}
	return resp.Attributes, nil
}

func putDataProtectionPolicy(ctx context.Context, client *sns.Client, arn, policy string) error {
	_, err := client.PutDataProtectionPolicy(ctx, &sns.PutDataProtectionPolicyInput{
		ResourceArn:          aws.String(arn),
		DataProtectionPolicy: aws.String(policy),
	})
	return err
}

func getDataProtectionPolicy(ctx context.Context, client *sns.Client, arn string) (string, error) {
	resp, err := client.GetDataProtectionPolicy(ctx, &sns.GetDataProtectionPolicyInput{ResourceArn: aws.String(arn)})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.DataProtectionPolicy), nil
}

func tagResource(ctx context.Context, client *sns.Client, arn string, tags map[string]string) error {
	items := make([]types.Tag, 0, len(tags))
	for key, val := range tags {
		items = append(items, types.Tag{Key: aws.String(key), Value: aws.String(val)})
	}
	_, err := client.TagResource(ctx, &sns.TagResourceInput{
		ResourceArn: aws.String(arn),
		Tags:        items,
	})
	return err
}

func listTags(ctx context.Context, client *sns.Client, arn string) ([]types.Tag, error) {
	resp, err := client.ListTagsForResource(ctx, &sns.ListTagsForResourceInput{ResourceArn: aws.String(arn)})
	if err != nil {
		return nil, err
	}
	return resp.Tags, nil
}

func subscribe(ctx context.Context, client *sns.Client, topicArn, protocol, endpoint string, returnArn bool) (string, error) {
	resp, err := client.Subscribe(ctx, &sns.SubscribeInput{
		TopicArn:              aws.String(topicArn),
		Protocol:              aws.String(protocol),
		Endpoint:              aws.String(endpoint),
		ReturnSubscriptionArn: returnArn,
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.SubscriptionArn), nil
}

func getSubscriptionAttributes(ctx context.Context, client *sns.Client, arn string) (map[string]string, error) {
	resp, err := client.GetSubscriptionAttributes(ctx, &sns.GetSubscriptionAttributesInput{SubscriptionArn: aws.String(arn)})
	if err != nil {
		return nil, err
	}
	return resp.Attributes, nil
}

func setSubscriptionAttribute(ctx context.Context, client *sns.Client, arn, name, value string) error {
	_, err := client.SetSubscriptionAttributes(ctx, &sns.SetSubscriptionAttributesInput{
		SubscriptionArn: aws.String(arn),
		AttributeName:   aws.String(name),
		AttributeValue:  aws.String(value),
	})
	return err
}

func listSubscriptions(ctx context.Context, client *sns.Client) ([]types.Subscription, error) {
	resp, err := client.ListSubscriptions(ctx, &sns.ListSubscriptionsInput{})
	if err != nil {
		return nil, err
	}
	return resp.Subscriptions, nil
}

func listSubscriptionsByTopic(ctx context.Context, client *sns.Client, arn string) ([]types.Subscription, error) {
	resp, err := client.ListSubscriptionsByTopic(ctx, &sns.ListSubscriptionsByTopicInput{TopicArn: aws.String(arn)})
	if err != nil {
		return nil, err
	}
	return resp.Subscriptions, nil
}

func unsubscribe(ctx context.Context, client *sns.Client, arn string) error {
	_, err := client.Unsubscribe(ctx, &sns.UnsubscribeInput{SubscriptionArn: aws.String(arn)})
	return err
}

func publishMessage(ctx context.Context, client *sns.Client, topicArn, message string) error {
	_, err := client.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(topicArn),
		Message:  aws.String(message),
		Subject:  aws.String("demo"),
	})
	return err
}

func publishBatch(ctx context.Context, client *sns.Client, topicArn string) error {
	_, err := client.PublishBatch(ctx, &sns.PublishBatchInput{
		TopicArn: aws.String(topicArn),
		PublishBatchRequestEntries: []types.PublishBatchRequestEntry{
			{Id: aws.String("msg-1"), Message: aws.String("batch-one")},
			{Id: aws.String("msg-2"), Message: aws.String("batch-two")},
		},
	})
	return err
}

func createPlatformApplication(ctx context.Context, client *sns.Client, name, platform string) (string, error) {
	resp, err := client.CreatePlatformApplication(ctx, &sns.CreatePlatformApplicationInput{
		Name:     aws.String(name),
		Platform: aws.String(platform),
		Attributes: map[string]string{
			"PlatformCredential": "demo",
		},
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.PlatformApplicationArn), nil
}

func deletePlatformApplication(ctx context.Context, client *sns.Client, arn string) error {
	_, err := client.DeletePlatformApplication(ctx, &sns.DeletePlatformApplicationInput{PlatformApplicationArn: aws.String(arn)})
	return err
}

func listPlatformApplications(ctx context.Context, client *sns.Client) ([]types.PlatformApplication, error) {
	resp, err := client.ListPlatformApplications(ctx, &sns.ListPlatformApplicationsInput{})
	if err != nil {
		return nil, err
	}
	return resp.PlatformApplications, nil
}

func getPlatformAttributes(ctx context.Context, client *sns.Client, arn string) (map[string]string, error) {
	resp, err := client.GetPlatformApplicationAttributes(ctx, &sns.GetPlatformApplicationAttributesInput{
		PlatformApplicationArn: aws.String(arn),
	})
	if err != nil {
		return nil, err
	}
	return resp.Attributes, nil
}

func setPlatformAttributes(ctx context.Context, client *sns.Client, arn string, attrs map[string]string) error {
	_, err := client.SetPlatformApplicationAttributes(ctx, &sns.SetPlatformApplicationAttributesInput{
		PlatformApplicationArn: aws.String(arn),
		Attributes:             attrs,
	})
	return err
}

func createPlatformEndpoint(ctx context.Context, client *sns.Client, appArn, token, customUserData string) (string, error) {
	resp, err := client.CreatePlatformEndpoint(ctx, &sns.CreatePlatformEndpointInput{
		PlatformApplicationArn: aws.String(appArn),
		Token:                  aws.String(token),
		CustomUserData:         aws.String(customUserData),
		Attributes: map[string]string{
			"Enabled": "true",
		},
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.EndpointArn), nil
}

func deleteEndpoint(ctx context.Context, client *sns.Client, arn string) error {
	_, err := client.DeleteEndpoint(ctx, &sns.DeleteEndpointInput{EndpointArn: aws.String(arn)})
	return err
}

func listEndpointsByPlatform(ctx context.Context, client *sns.Client, appArn string) ([]types.Endpoint, error) {
	resp, err := client.ListEndpointsByPlatformApplication(ctx, &sns.ListEndpointsByPlatformApplicationInput{
		PlatformApplicationArn: aws.String(appArn),
	})
	if err != nil {
		return nil, err
	}
	return resp.Endpoints, nil
}

func getEndpointAttributes(ctx context.Context, client *sns.Client, arn string) (map[string]string, error) {
	resp, err := client.GetEndpointAttributes(ctx, &sns.GetEndpointAttributesInput{EndpointArn: aws.String(arn)})
	if err != nil {
		return nil, err
	}
	return resp.Attributes, nil
}

func setEndpointAttributes(ctx context.Context, client *sns.Client, arn string, attrs map[string]string) error {
	_, err := client.SetEndpointAttributes(ctx, &sns.SetEndpointAttributesInput{
		EndpointArn: aws.String(arn),
		Attributes:  attrs,
	})
	return err
}

func createSMSSandboxPhoneNumber(ctx context.Context, client *sns.Client, number string) error {
	_, err := client.CreateSMSSandboxPhoneNumber(ctx, &sns.CreateSMSSandboxPhoneNumberInput{PhoneNumber: aws.String(number)})
	return err
}

func verifySMSSandboxPhoneNumber(ctx context.Context, client *sns.Client, number, otp string) error {
	_, err := client.VerifySMSSandboxPhoneNumber(ctx, &sns.VerifySMSSandboxPhoneNumberInput{
		PhoneNumber:     aws.String(number),
		OneTimePassword: aws.String(otp),
	})
	return err
}

func deleteSMSSandboxPhoneNumber(ctx context.Context, client *sns.Client, number string) error {
	_, err := client.DeleteSMSSandboxPhoneNumber(ctx, &sns.DeleteSMSSandboxPhoneNumberInput{PhoneNumber: aws.String(number)})
	return err
}

func listSMSSandboxPhoneNumbers(ctx context.Context, client *sns.Client) ([]types.SMSSandboxPhoneNumber, error) {
	resp, err := client.ListSMSSandboxPhoneNumbers(ctx, &sns.ListSMSSandboxPhoneNumbersInput{})
	if err != nil {
		return nil, err
	}
	return resp.PhoneNumbers, nil
}

func getSMSSandboxStatus(ctx context.Context, client *sns.Client) (bool, error) {
	resp, err := client.GetSMSSandboxAccountStatus(ctx, &sns.GetSMSSandboxAccountStatusInput{})
	if err != nil {
		return false, err
	}
	return resp.IsInSandbox, nil
}

func getSMSAttributes(ctx context.Context, client *sns.Client) (map[string]string, error) {
	resp, err := client.GetSMSAttributes(ctx, &sns.GetSMSAttributesInput{})
	if err != nil {
		return nil, err
	}
	return resp.Attributes, nil
}

func setSMSAttributes(ctx context.Context, client *sns.Client, attrs map[string]string) error {
	_, err := client.SetSMSAttributes(ctx, &sns.SetSMSAttributesInput{Attributes: attrs})
	return err
}

func checkIfPhoneNumberIsOptedOut(ctx context.Context, client *sns.Client, number string) (bool, error) {
	resp, err := client.CheckIfPhoneNumberIsOptedOut(ctx, &sns.CheckIfPhoneNumberIsOptedOutInput{PhoneNumber: aws.String(number)})
	if err != nil {
		return false, err
	}
	return resp.IsOptedOut, nil
}

func listPhoneNumbersOptedOut(ctx context.Context, client *sns.Client) ([]string, error) {
	resp, err := client.ListPhoneNumbersOptedOut(ctx, &sns.ListPhoneNumbersOptedOutInput{})
	if err != nil {
		return nil, err
	}
	return resp.PhoneNumbers, nil
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func init() {
	_ = time.Now()
}
