package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	source := getenv("STACKYARD_SOURCE_EMAIL", "sender@example.com")
	destination := getenv("STACKYARD_DEST_EMAIL", "recipient@example.com")
	domain := getenv("STACKYARD_DOMAIN", "example.com")
	templateName := getenv("STACKYARD_TEMPLATE", "welcome-template")

	ctx := context.Background()
	client := newSESClient(ctx, endpoint)

	fmt.Printf("Stackyard SES advanced client using %s\n", endpoint)

	if err := verifyEmailIdentity(ctx, client, source); err != nil {
		exitf("verify source email: %v", err)
	}

	token, err := verifyDomainIdentity(ctx, client, domain)
	if err != nil {
		exitf("verify domain identity: %v", err)
	}
	logf("domain verification token: %s", token)

	dkimTokens, err := verifyDomainDkim(ctx, client, domain)
	if err != nil {
		exitf("verify domain dkim: %v", err)
	}
	logf("dkim tokens: %d", len(dkimTokens))

	identities, err := listIdentities(ctx, client)
	if err != nil {
		exitf("list identities: %v", err)
	}
	logf("identities: %d", len(identities))

	verifiedEmails, err := listVerifiedEmailAddresses(ctx, client)
	if err != nil {
		exitf("list verified email addresses: %v", err)
	}
	logf("verified emails: %d", len(verifiedEmails))

	attrs, err := getIdentityVerificationAttributes(ctx, client, []string{source, domain})
	if err != nil {
		exitf("get identity verification attributes: %v", err)
	}
	logf("verification attributes: %d", len(attrs))

	if err := setIdentityNotificationTopic(ctx, client, domain, types.NotificationTypeBounce, "arn:aws:sns:us-east-1:123456789012:bounces"); err != nil {
		exitf("set identity notification topic: %v", err)
	}
	if err := setIdentityHeadersInNotificationsEnabled(ctx, client, domain, types.NotificationTypeBounce, true); err != nil {
		exitf("set identity headers notifications: %v", err)
	}
	if err := setIdentityFeedbackForwardingEnabled(ctx, client, domain, false); err != nil {
		exitf("set identity feedback forwarding: %v", err)
	}
	if err := setIdentityMailFromDomain(ctx, client, domain, "mail."+domain); err != nil {
		exitf("set identity mail from domain: %v", err)
	}

	dkimAttrs, err := getIdentityDkimAttributes(ctx, client, []string{domain})
	if err != nil {
		exitf("get identity dkim attributes: %v", err)
	}
	logf("dkim attributes: %d", len(dkimAttrs))

	notifAttrs, err := getIdentityNotificationAttributes(ctx, client, []string{domain})
	if err != nil {
		exitf("get identity notification attributes: %v", err)
	}
	logf("notification attributes: %d", len(notifAttrs))

	mailFromAttrs, err := getIdentityMailFromDomainAttributes(ctx, client, []string{domain})
	if err != nil {
		exitf("get identity mail from attributes: %v", err)
	}
	logf("mail-from attributes: %d", len(mailFromAttrs))

	accountSendingEnabled, err := getAccountSendingEnabled(ctx, client)
	if err != nil {
		exitf("get account sending enabled: %v", err)
	}
	logf("account sending enabled: %t", accountSendingEnabled)

	if err := createTemplate(ctx, client, templateName); err != nil {
		exitf("create template: %v", err)
	}

	tpl, err := getTemplate(ctx, client, templateName)
	if err != nil {
		exitf("get template: %v", err)
	}
	logf("template subject: %s", aws.ToString(tpl.SubjectPart))

	templates, err := listTemplates(ctx, client)
	if err != nil {
		exitf("list templates: %v", err)
	}
	logf("templates: %d", len(templates))

	if err := updateTemplate(ctx, client, templateName); err != nil {
		exitf("update template: %v", err)
	}

	msgID, err := sendTemplatedEmail(ctx, client, source, destination, templateName)
	if err != nil {
		exitf("send templated email: %v", err)
	}
	logf("templated message id: %s", msgID)

	rawMsgID, err := sendRawEmail(ctx, client, source, destination)
	if err != nil {
		exitf("send raw email: %v", err)
	}
	logf("raw message id: %s", rawMsgID)

	quota, err := getSendQuota(ctx, client)
	if err != nil {
		exitf("get send quota: %v", err)
	}
	logf("send quota used last 24h: %.0f", quota.SentLast24Hours)

	stats, err := getSendStatistics(ctx, client)
	if err != nil {
		exitf("get send statistics: %v", err)
	}
	logf("send datapoints: %d", len(stats))

	if err := deleteTemplate(ctx, client, templateName); err != nil {
		exitf("delete template: %v", err)
	}
	if err := deleteIdentity(ctx, client, source); err != nil {
		exitf("delete source identity: %v", err)
	}
	if err := deleteIdentity(ctx, client, domain); err != nil {
		exitf("delete domain identity: %v", err)
	}

	fmt.Println("Done.")
}

func newSESClient(ctx context.Context, endpoint string) *ses.Client {
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, _ ...any) (aws.Endpoint, error) {
		if service == ses.ServiceID {
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

	return ses.NewFromConfig(cfg)
}

func verifyEmailIdentity(ctx context.Context, client *ses.Client, email string) error {
	_, err := client.VerifyEmailIdentity(ctx, &ses.VerifyEmailIdentityInput{EmailAddress: aws.String(email)})
	return err
}

func verifyDomainIdentity(ctx context.Context, client *ses.Client, domain string) (string, error) {
	resp, err := client.VerifyDomainIdentity(ctx, &ses.VerifyDomainIdentityInput{Domain: aws.String(domain)})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.VerificationToken), nil
}

func verifyDomainDkim(ctx context.Context, client *ses.Client, domain string) ([]string, error) {
	resp, err := client.VerifyDomainDkim(ctx, &ses.VerifyDomainDkimInput{Domain: aws.String(domain)})
	if err != nil {
		return nil, err
	}
	return resp.DkimTokens, nil
}

func listIdentities(ctx context.Context, client *ses.Client) ([]string, error) {
	resp, err := client.ListIdentities(ctx, &ses.ListIdentitiesInput{MaxItems: aws.Int32(100)})
	if err != nil {
		return nil, err
	}
	return resp.Identities, nil
}

func getIdentityVerificationAttributes(ctx context.Context, client *ses.Client, identities []string) (map[string]types.IdentityVerificationAttributes, error) {
	resp, err := client.GetIdentityVerificationAttributes(ctx, &ses.GetIdentityVerificationAttributesInput{Identities: identities})
	if err != nil {
		return nil, err
	}
	return resp.VerificationAttributes, nil
}

func getIdentityDkimAttributes(ctx context.Context, client *ses.Client, identities []string) (map[string]types.IdentityDkimAttributes, error) {
	resp, err := client.GetIdentityDkimAttributes(ctx, &ses.GetIdentityDkimAttributesInput{Identities: identities})
	if err != nil {
		return nil, err
	}
	return resp.DkimAttributes, nil
}

func getIdentityNotificationAttributes(ctx context.Context, client *ses.Client, identities []string) (map[string]types.IdentityNotificationAttributes, error) {
	resp, err := client.GetIdentityNotificationAttributes(ctx, &ses.GetIdentityNotificationAttributesInput{Identities: identities})
	if err != nil {
		return nil, err
	}
	return resp.NotificationAttributes, nil
}

func getIdentityMailFromDomainAttributes(ctx context.Context, client *ses.Client, identities []string) (map[string]types.IdentityMailFromDomainAttributes, error) {
	resp, err := client.GetIdentityMailFromDomainAttributes(ctx, &ses.GetIdentityMailFromDomainAttributesInput{Identities: identities})
	if err != nil {
		return nil, err
	}
	return resp.MailFromDomainAttributes, nil
}

func listVerifiedEmailAddresses(ctx context.Context, client *ses.Client) ([]string, error) {
	resp, err := client.ListVerifiedEmailAddresses(ctx, &ses.ListVerifiedEmailAddressesInput{})
	if err != nil {
		return nil, err
	}
	return resp.VerifiedEmailAddresses, nil
}

func setIdentityNotificationTopic(ctx context.Context, client *ses.Client, identity string, notificationType types.NotificationType, topicARN string) error {
	_, err := client.SetIdentityNotificationTopic(ctx, &ses.SetIdentityNotificationTopicInput{
		Identity:         aws.String(identity),
		NotificationType: notificationType,
		SnsTopic:         aws.String(topicARN),
	})
	return err
}

func setIdentityHeadersInNotificationsEnabled(ctx context.Context, client *ses.Client, identity string, notificationType types.NotificationType, enabled bool) error {
	_, err := client.SetIdentityHeadersInNotificationsEnabled(ctx, &ses.SetIdentityHeadersInNotificationsEnabledInput{
		Identity:         aws.String(identity),
		NotificationType: notificationType,
		Enabled:          enabled,
	})
	return err
}

func setIdentityFeedbackForwardingEnabled(ctx context.Context, client *ses.Client, identity string, enabled bool) error {
	_, err := client.SetIdentityFeedbackForwardingEnabled(ctx, &ses.SetIdentityFeedbackForwardingEnabledInput{
		Identity:          aws.String(identity),
		ForwardingEnabled: enabled,
	})
	return err
}

func setIdentityMailFromDomain(ctx context.Context, client *ses.Client, identity, mailFromDomain string) error {
	_, err := client.SetIdentityMailFromDomain(ctx, &ses.SetIdentityMailFromDomainInput{
		Identity:            aws.String(identity),
		MailFromDomain:      aws.String(mailFromDomain),
		BehaviorOnMXFailure: types.BehaviorOnMXFailureRejectMessage,
	})
	return err
}

func getAccountSendingEnabled(ctx context.Context, client *ses.Client) (bool, error) {
	resp, err := client.GetAccountSendingEnabled(ctx, &ses.GetAccountSendingEnabledInput{})
	if err != nil {
		return false, err
	}
	return resp.Enabled, nil
}

func createTemplate(ctx context.Context, client *ses.Client, templateName string) error {
	_, err := client.CreateTemplate(ctx, &ses.CreateTemplateInput{Template: &types.Template{
		TemplateName: aws.String(templateName),
		SubjectPart:  aws.String("Welcome {{name}}"),
		TextPart:     aws.String("Hello {{name}} from Stackyard SES advanced example."),
		HtmlPart:     aws.String("<h1>Hello {{name}}</h1><p>from Stackyard SES advanced example.</p>"),
	}})
	return err
}

func getTemplate(ctx context.Context, client *ses.Client, templateName string) (*types.Template, error) {
	resp, err := client.GetTemplate(ctx, &ses.GetTemplateInput{TemplateName: aws.String(templateName)})
	if err != nil {
		return nil, err
	}
	return resp.Template, nil
}

func listTemplates(ctx context.Context, client *ses.Client) ([]types.TemplateMetadata, error) {
	resp, err := client.ListTemplates(ctx, &ses.ListTemplatesInput{MaxItems: aws.Int32(100)})
	if err != nil {
		return nil, err
	}
	return resp.TemplatesMetadata, nil
}

func updateTemplate(ctx context.Context, client *ses.Client, templateName string) error {
	_, err := client.UpdateTemplate(ctx, &ses.UpdateTemplateInput{Template: &types.Template{
		TemplateName: aws.String(templateName),
		SubjectPart:  aws.String("Updated welcome {{name}}"),
		TextPart:     aws.String("Updated text for {{name}}."),
		HtmlPart:     aws.String("<strong>Updated html for {{name}}</strong>"),
	}})
	return err
}

func sendTemplatedEmail(ctx context.Context, client *ses.Client, source, destination, templateName string) (string, error) {
	resp, err := client.SendTemplatedEmail(ctx, &ses.SendTemplatedEmailInput{
		Source: aws.String(source),
		Destination: &types.Destination{
			ToAddresses: []string{destination},
		},
		Template:     aws.String(templateName),
		TemplateData: aws.String(`{"name":"Stackyard"}`),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.MessageId), nil
}

func sendRawEmail(ctx context.Context, client *ses.Client, source, destination string) (string, error) {
	raw := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: Stackyard SES Raw\r\n\r\nhello from raw email", source, destination)
	resp, err := client.SendRawEmail(ctx, &ses.SendRawEmailInput{
		Source:       aws.String(source),
		Destinations: []string{destination},
		RawMessage:   &types.RawMessage{Data: []byte(raw)},
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.MessageId), nil
}

func getSendQuota(ctx context.Context, client *ses.Client) (*ses.GetSendQuotaOutput, error) {
	return client.GetSendQuota(ctx, &ses.GetSendQuotaInput{})
}

func getSendStatistics(ctx context.Context, client *ses.Client) ([]types.SendDataPoint, error) {
	resp, err := client.GetSendStatistics(ctx, &ses.GetSendStatisticsInput{})
	if err != nil {
		return nil, err
	}
	return resp.SendDataPoints, nil
}

func deleteTemplate(ctx context.Context, client *ses.Client, templateName string) error {
	_, err := client.DeleteTemplate(ctx, &ses.DeleteTemplateInput{TemplateName: aws.String(templateName)})
	return err
}

func deleteIdentity(ctx context.Context, client *ses.Client, identity string) error {
	_, err := client.DeleteIdentity(ctx, &ses.DeleteIdentityInput{Identity: aws.String(identity)})
	return err
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
