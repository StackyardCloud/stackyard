package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/aws/smithy-go"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	source := getenv("STACKYARD_SOURCE_EMAIL", "sender@example.com")
	destination := getenv("STACKYARD_DEST_EMAIL", "recipient@example.com")
	domain := getenv("STACKYARD_DOMAIN", "example.com")
	configSet := getenv("STACKYARD_CONFIG_SET", "demo-config")
	templateName := getenv("STACKYARD_TEMPLATE", "welcome-template")
	contactList := getenv("STACKYARD_CONTACT_LIST", "audience")

	ctx := context.Background()
	client := newSESV2Client(ctx, endpoint)

	fmt.Printf("Stackyard SESv2 advanced client using %s\n", endpoint)

	if err := createConfigurationSet(ctx, client, configSet); err != nil {
		exitf("create configuration set: %v", err)
	}
	logf("created configuration set: %s", configSet)

	if err := createEmailIdentity(ctx, client, source, configSet); err != nil {
		exitf("create email identity: %v", err)
	}
	if err := putIdentityAttributes(ctx, client, source, domain, configSet); err != nil {
		exitf("put identity attributes: %v", err)
	}

	if err := createEmailTemplate(ctx, client, templateName); err != nil {
		exitf("create email template: %v", err)
	}
	if err := updateEmailTemplate(ctx, client, templateName); err != nil {
		exitf("update email template: %v", err)
	}

	templateSubject, err := getEmailTemplateSubject(ctx, client, templateName)
	if err != nil {
		exitf("get email template: %v", err)
	}
	logf("template subject: %s", templateSubject)

	rendered, err := testRenderTemplate(ctx, client, templateName)
	if err != nil {
		exitf("test render template: %v", err)
	}
	logf("rendered template: %s", rendered)

	templateMessageID, err := sendTemplatedEmail(ctx, client, source, destination, templateName)
	if err != nil {
		exitf("send templated email: %v", err)
	}
	logf("templated message id: %s", templateMessageID)

	bulkCount, err := sendBulkEmail(ctx, client, source, destination, templateName)
	if err != nil {
		exitf("send bulk email: %v", err)
	}
	logf("bulk entry results: %d", bulkCount)

	if err := suppressionLifecycle(ctx, client, destination); err != nil {
		exitf("suppression lifecycle: %v", err)
	}

	resourceARN := "arn:aws:ses:us-east-1:123456789012:configuration-set/" + configSet
	if err := tagsLifecycle(ctx, client, resourceARN); err != nil {
		exitf("tags lifecycle: %v", err)
	}

	if err := contactsLifecycle(ctx, client, contactList, destination); err != nil {
		exitf("contacts lifecycle: %v", err)
	}

	if err := toggleSending(ctx, client); err != nil {
		exitf("toggle sending attributes: %v", err)
	}

	if err := cleanup(ctx, client, source, templateName, configSet, contactList); err != nil {
		exitf("cleanup: %v", err)
	}

	fmt.Println("Done.")
}

func newSESV2Client(ctx context.Context, endpoint string) *sesv2.Client {
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, _ ...any) (aws.Endpoint, error) {
		if service == sesv2.ServiceID {
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

	return sesv2.NewFromConfig(cfg)
}

func createConfigurationSet(ctx context.Context, client *sesv2.Client, name string) error {
	_, err := client.CreateConfigurationSet(ctx, &sesv2.CreateConfigurationSetInput{ConfigurationSetName: aws.String(name)})
	return err
}

func createEmailIdentity(ctx context.Context, client *sesv2.Client, identity, configSet string) error {
	_, err := client.CreateEmailIdentity(ctx, &sesv2.CreateEmailIdentityInput{
		EmailIdentity:        aws.String(identity),
		ConfigurationSetName: aws.String(configSet),
	})
	return err
}

func putIdentityAttributes(ctx context.Context, client *sesv2.Client, identity, domain, configSet string) error {
	if _, err := client.PutEmailIdentityFeedbackAttributes(ctx, &sesv2.PutEmailIdentityFeedbackAttributesInput{
		EmailIdentity:          aws.String(identity),
		EmailForwardingEnabled: true,
	}); err != nil {
		return err
	}
	if _, err := client.PutEmailIdentityDkimAttributes(ctx, &sesv2.PutEmailIdentityDkimAttributesInput{
		EmailIdentity:  aws.String(identity),
		SigningEnabled: true,
	}); err != nil {
		return err
	}
	if _, err := client.PutEmailIdentityMailFromAttributes(ctx, &sesv2.PutEmailIdentityMailFromAttributesInput{
		EmailIdentity:       aws.String(identity),
		MailFromDomain:      aws.String("mail." + domain),
		BehaviorOnMxFailure: types.BehaviorOnMxFailureUseDefaultValue,
	}); err != nil {
		return err
	}
	_, err := client.PutEmailIdentityConfigurationSetAttributes(ctx, &sesv2.PutEmailIdentityConfigurationSetAttributesInput{
		EmailIdentity:        aws.String(identity),
		ConfigurationSetName: aws.String(configSet),
	})
	return err
}

func createEmailTemplate(ctx context.Context, client *sesv2.Client, name string) error {
	_, err := client.CreateEmailTemplate(ctx, &sesv2.CreateEmailTemplateInput{
		TemplateName: aws.String(name),
		TemplateContent: &types.EmailTemplateContent{
			Subject: aws.String("Welcome {{name}}"),
			Text:    aws.String("Hello {{name}} from stackyard sesv2 advanced"),
			Html:    aws.String("<h1>Hello {{name}} from stackyard sesv2 advanced</h1>"),
		},
	})
	return err
}

func updateEmailTemplate(ctx context.Context, client *sesv2.Client, name string) error {
	_, err := client.UpdateEmailTemplate(ctx, &sesv2.UpdateEmailTemplateInput{
		TemplateName: aws.String(name),
		TemplateContent: &types.EmailTemplateContent{
			Subject: aws.String("Updated Welcome {{name}}"),
			Text:    aws.String("Updated hello {{name}}"),
			Html:    aws.String("<h1>Updated hello {{name}}</h1>"),
		},
	})
	return err
}

func getEmailTemplateSubject(ctx context.Context, client *sesv2.Client, name string) (string, error) {
	resp, err := client.GetEmailTemplate(ctx, &sesv2.GetEmailTemplateInput{TemplateName: aws.String(name)})
	if err != nil {
		return "", err
	}
	if resp.TemplateContent == nil {
		return "", nil
	}
	return aws.ToString(resp.TemplateContent.Subject), nil
}

func testRenderTemplate(ctx context.Context, client *sesv2.Client, name string) (string, error) {
	resp, err := client.TestRenderEmailTemplate(ctx, &sesv2.TestRenderEmailTemplateInput{
		TemplateName: aws.String(name),
		TemplateData: aws.String(`{"name":"Stackyard"}`),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.RenderedTemplate), nil
}

func sendTemplatedEmail(ctx context.Context, client *sesv2.Client, source, destination, templateName string) (string, error) {
	resp, err := client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(source),
		Destination: &types.Destination{
			ToAddresses: []string{destination},
		},
		Content: &types.EmailContent{
			Template: &types.Template{
				TemplateName: aws.String(templateName),
				TemplateData: aws.String(`{"name":"Stackyard"}`),
			},
		},
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.MessageId), nil
}

func sendBulkEmail(ctx context.Context, client *sesv2.Client, source, destination, templateName string) (int, error) {
	resp, err := client.SendBulkEmail(ctx, &sesv2.SendBulkEmailInput{
		FromEmailAddress: aws.String(source),
		DefaultContent: &types.BulkEmailContent{
			Template: &types.Template{
				TemplateName: aws.String(templateName),
				TemplateData: aws.String(`{"name":"bulk-default"}`),
			},
		},
		BulkEmailEntries: []types.BulkEmailEntry{
			{Destination: &types.Destination{ToAddresses: []string{destination}}, ReplacementEmailContent: &types.ReplacementEmailContent{ReplacementTemplate: &types.ReplacementTemplate{ReplacementTemplateData: aws.String(`{"name":"one"}`)}}},
			{Destination: &types.Destination{ToAddresses: []string{destination}}, ReplacementEmailContent: &types.ReplacementEmailContent{ReplacementTemplate: &types.ReplacementTemplate{ReplacementTemplateData: aws.String(`{"name":"two"}`)}}},
		},
	})
	if err != nil {
		return 0, err
	}
	return len(resp.BulkEmailEntryResults), nil
}

func suppressionLifecycle(ctx context.Context, client *sesv2.Client, destination string) error {
	if _, err := client.PutSuppressedDestination(ctx, &sesv2.PutSuppressedDestinationInput{
		EmailAddress: aws.String(destination),
		Reason:       types.SuppressionListReasonBounce,
	}); err != nil {
		return err
	}

	if _, err := client.GetSuppressedDestination(ctx, &sesv2.GetSuppressedDestinationInput{EmailAddress: aws.String(destination)}); err != nil {
		return err
	}

	if _, err := client.ListSuppressedDestinations(ctx, &sesv2.ListSuppressedDestinationsInput{}); err != nil {
		return err
	}

	_, err := client.DeleteSuppressedDestination(ctx, &sesv2.DeleteSuppressedDestinationInput{EmailAddress: aws.String(destination)})
	return err
}

func tagsLifecycle(ctx context.Context, client *sesv2.Client, resourceARN string) error {
	if _, err := client.TagResource(ctx, &sesv2.TagResourceInput{
		ResourceArn: aws.String(resourceARN),
		Tags:        []types.Tag{{Key: aws.String("env"), Value: aws.String("test")}},
	}); err != nil {
		return err
	}
	if _, err := client.ListTagsForResource(ctx, &sesv2.ListTagsForResourceInput{ResourceArn: aws.String(resourceARN)}); err != nil {
		return err
	}
	_, err := client.UntagResource(ctx, &sesv2.UntagResourceInput{ResourceArn: aws.String(resourceARN), TagKeys: []string{"env"}})
	return err
}

func contactsLifecycle(ctx context.Context, client *sesv2.Client, listName, destination string) error {
	if _, err := client.CreateContactList(ctx, &sesv2.CreateContactListInput{ContactListName: aws.String(listName)}); err != nil {
		return err
	}
	if _, err := client.CreateContact(ctx, &sesv2.CreateContactInput{ContactListName: aws.String(listName), EmailAddress: aws.String(destination)}); err != nil {
		return err
	}
	if _, err := client.ListContacts(ctx, &sesv2.ListContactsInput{ContactListName: aws.String(listName)}); err != nil {
		return err
	}
	if _, err := client.GetContact(ctx, &sesv2.GetContactInput{ContactListName: aws.String(listName), EmailAddress: aws.String(destination)}); err != nil {
		return err
	}
	if _, err := client.UpdateContact(ctx, &sesv2.UpdateContactInput{ContactListName: aws.String(listName), EmailAddress: aws.String(destination), AttributesData: aws.String(`{"tier":"gold"}`)}); err != nil {
		return err
	}
	if _, err := client.DeleteContact(ctx, &sesv2.DeleteContactInput{ContactListName: aws.String(listName), EmailAddress: aws.String(destination)}); err != nil {
		return err
	}
	_, err := client.DeleteContactList(ctx, &sesv2.DeleteContactListInput{ContactListName: aws.String(listName)})
	return err
}

func toggleSending(ctx context.Context, client *sesv2.Client) error {
	if _, err := client.PutAccountSendingAttributes(ctx, &sesv2.PutAccountSendingAttributesInput{SendingEnabled: false}); err != nil {
		return err
	}
	if _, err := client.GetAccount(ctx, &sesv2.GetAccountInput{}); err != nil {
		return err
	}
	_, err := client.PutAccountSendingAttributes(ctx, &sesv2.PutAccountSendingAttributesInput{SendingEnabled: true})
	return err
}

func cleanup(ctx context.Context, client *sesv2.Client, source, templateName, configSet, contactList string) error {
	if _, err := client.DeleteEmailTemplate(ctx, &sesv2.DeleteEmailTemplateInput{TemplateName: aws.String(templateName)}); err != nil {
		return err
	}
	if _, err := client.DeleteEmailIdentity(ctx, &sesv2.DeleteEmailIdentityInput{EmailIdentity: aws.String(source)}); err != nil {
		return err
	}
	if _, err := client.DeleteConfigurationSet(ctx, &sesv2.DeleteConfigurationSetInput{ConfigurationSetName: aws.String(configSet)}); err != nil {
		return err
	}
	if _, err := client.DeleteContactList(ctx, &sesv2.DeleteContactListInput{ContactListName: aws.String(contactList)}); err != nil {
		if !isSESV2NotFound(err) {
			return err
		}
	}
	return nil
}

func isSESV2NotFound(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "NotFoundException"
	}
	return false
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
