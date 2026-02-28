package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage2SDKLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(testRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(testAccessKey, testSecretKey, "")),
	)
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}
	client := awsec2.NewFromConfig(cfg, func(o *awsec2.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
	})

	createSGOut, err := client.CreateSecurityGroup(ctx, &awsec2.CreateSecurityGroupInput{
		GroupName:   aws.String("stage2-sg"),
		Description: aws.String("stage2"),
		VpcId:       aws.String("vpc-00000001"),
	})
	if err != nil || createSGOut.GroupId == nil {
		t.Fatalf("create security group: %v", err)
	}
	sgID := aws.ToString(createSGOut.GroupId)

	if _, err := client.AuthorizeSecurityGroupEgress(ctx, &awsec2.AuthorizeSecurityGroupEgressInput{
		GroupId: aws.String(sgID),
		IpPermissions: []awsec2types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(443),
				ToPort:     aws.Int32(443),
				IpRanges:   []awsec2types.IpRange{{CidrIp: aws.String("0.0.0.0/0")}},
			},
		},
	}); err != nil {
		t.Fatalf("authorize security group egress: %v", err)
	}
	if _, err := client.RevokeSecurityGroupEgress(ctx, &awsec2.RevokeSecurityGroupEgressInput{
		GroupId: aws.String(sgID),
		IpPermissions: []awsec2types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(443),
				ToPort:     aws.Int32(443),
				IpRanges:   []awsec2types.IpRange{{CidrIp: aws.String("0.0.0.0/0")}},
			},
		},
	}); err != nil {
		t.Fatalf("revoke security group egress: %v", err)
	}

	createKeyPairOut, err := client.CreateKeyPair(ctx, &awsec2.CreateKeyPairInput{KeyName: aws.String("stage2-key")})
	if err != nil || createKeyPairOut.KeyName == nil {
		t.Fatalf("create key pair: %v", err)
	}
	if _, err := client.ImportKeyPair(ctx, &awsec2.ImportKeyPairInput{
		KeyName:           aws.String("stage2-key-import"),
		PublicKeyMaterial: []byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDcstage2 test@example.com"),
	}); err != nil {
		t.Fatalf("import key pair: %v", err)
	}
	if _, err := client.DescribeKeyPairs(ctx, &awsec2.DescribeKeyPairsInput{KeyNames: []string{"stage2-key", "stage2-key-import"}}); err != nil {
		t.Fatalf("describe key pairs: %v", err)
	}

	runOut, err := client.RunInstances(ctx, &awsec2.RunInstancesInput{
		ImageId:          aws.String("ami-12345678"),
		MinCount:         aws.Int32(1),
		MaxCount:         aws.Int32(1),
		InstanceType:     awsec2types.InstanceTypeT3Micro,
		SubnetId:         aws.String("subnet-00000001"),
		SecurityGroupIds: []string{sgID},
		KeyName:          aws.String("stage2-key"),
	})
	if err != nil || len(runOut.Instances) == 0 || runOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runOut.Instances[0].InstanceId)

	associateOut, err := client.AssociateIamInstanceProfile(ctx, &awsec2.AssociateIamInstanceProfileInput{
		InstanceId: aws.String(instanceID),
		IamInstanceProfile: &awsec2types.IamInstanceProfileSpecification{
			Name: aws.String("stage2-profile"),
		},
	})
	if err != nil || associateOut.IamInstanceProfileAssociation == nil || associateOut.IamInstanceProfileAssociation.AssociationId == nil {
		t.Fatalf("associate iam instance profile: %v", err)
	}
	associationID := aws.ToString(associateOut.IamInstanceProfileAssociation.AssociationId)

	if _, err := client.DescribeIamInstanceProfileAssociations(ctx, &awsec2.DescribeIamInstanceProfileAssociationsInput{
		AssociationIds: []string{associationID},
	}); err != nil {
		t.Fatalf("describe iam instance profile associations: %v", err)
	}

	if _, err := client.ReplaceIamInstanceProfileAssociation(ctx, &awsec2.ReplaceIamInstanceProfileAssociationInput{
		AssociationId: aws.String(associationID),
		IamInstanceProfile: &awsec2types.IamInstanceProfileSpecification{
			Name: aws.String("stage2-profile-replaced"),
		},
	}); err != nil {
		t.Fatalf("replace iam instance profile association: %v", err)
	}

	if _, err := client.DisassociateIamInstanceProfile(ctx, &awsec2.DisassociateIamInstanceProfileInput{
		AssociationId: aws.String(associationID),
	}); err != nil {
		t.Fatalf("disassociate iam instance profile: %v", err)
	}

	if _, err := client.TerminateInstances(ctx, &awsec2.TerminateInstancesInput{InstanceIds: []string{instanceID}}); err != nil {
		t.Fatalf("terminate instance: %v", err)
	}
	if _, err := client.DeleteKeyPair(ctx, &awsec2.DeleteKeyPairInput{KeyName: aws.String("stage2-key")}); err != nil {
		t.Fatalf("delete key pair: %v", err)
	}
	if _, err := client.DeleteKeyPair(ctx, &awsec2.DeleteKeyPairInput{KeyName: aws.String("stage2-key-import")}); err != nil {
		t.Fatalf("delete imported key pair: %v", err)
	}
	if _, err := client.DeleteSecurityGroup(ctx, &awsec2.DeleteSecurityGroupInput{GroupId: aws.String(sgID)}); err != nil {
		t.Fatalf("delete security group: %v", err)
	}
}

func TestEC2Stage2ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"AuthorizeSecurityGroupEgress",
		"RevokeSecurityGroupEgress",
		"CreateKeyPair",
		"ImportKeyPair",
		"DescribeKeyPairs",
		"DeleteKeyPair",
		"AssociateIamInstanceProfile",
		"DescribeIamInstanceProfileAssociations",
		"DisassociateIamInstanceProfile",
		"ReplaceIamInstanceProfileAssociation",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for idx, action := range implemented {
		params := map[string]string{}
		switch action {
		case "AuthorizeSecurityGroupEgress", "RevokeSecurityGroupEgress":
			params["GroupId"] = "sg-00000000"
			params["IpPermissions.1.IpProtocol"] = "tcp"
			params["IpPermissions.1.FromPort"] = "443"
			params["IpPermissions.1.ToPort"] = "443"
			params["IpPermissions.1.IpRanges.1.CidrIp"] = "0.0.0.0/0"
		case "CreateKeyPair":
			params["KeyName"] = "stage2-key-" + strconv.Itoa(idx)
		case "ImportKeyPair":
			params["KeyName"] = "stage2-import-" + strconv.Itoa(idx)
			params["PublicKeyMaterial"] = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDcstage2 test@example.com"
		case "DescribeKeyPairs":
			params["KeyName.1"] = "stage2-key-" + strconv.Itoa(idx)
		case "DeleteKeyPair":
			params["KeyName"] = "stage2-key-" + strconv.Itoa(idx)
		case "AssociateIamInstanceProfile":
			params["InstanceId"] = "i-00000002"
			params["IamInstanceProfile.Name"] = "profile-" + strconv.Itoa(idx)
		case "DescribeIamInstanceProfileAssociations":
			params["AssociationId.1"] = "iip-assoc-00000001"
		case "DisassociateIamInstanceProfile", "ReplaceIamInstanceProfileAssociation":
			params["AssociationId"] = "iip-assoc-00000001"
			params["IamInstanceProfile.Name"] = "profile-" + strconv.Itoa(idx)
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
