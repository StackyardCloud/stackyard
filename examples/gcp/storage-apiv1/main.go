package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/iam"
	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func main() {
	ctx := context.Background()
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	emulatorHost := getenv("STORAGE_EMULATOR_HOST", endpoint+"/gcp")
	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	bucketName := getenv("STACKYARD_GCP_STORAGE_BUCKET", "stackyard-storage-bucket")
	objectName := getenv("STACKYARD_GCP_STORAGE_OBJECT", "orders/2026-03-06.json")

	if err := os.Setenv("STORAGE_EMULATOR_HOST", emulatorHost); err != nil {
		exitf("failed to set STORAGE_EMULATOR_HOST: %v", err)
	}

	fmt.Printf("Stackyard GCP Cloud Storage client using %s\n", emulatorHost)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "storage-apiv1",
		},
	}

	client, err := storage.NewClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create storage client: %v", err)
	}
	defer closeClient(client)

	bucket := client.Bucket(bucketName)
	movedObject := "orders/moved.json"
	composedObject := "orders/composed.json"

	must("CreateBucket", bucket.Create(ctx, projectID, &storage.BucketAttrs{
		Location:          "US",
		StorageClass:      "STANDARD",
		VersioningEnabled: true,
	}))

	bucketAttrs, err := bucket.Attrs(ctx)
	must("GetBucketAttrs", err)
	assert(bucketAttrs.Name == bucketName, "bucket name mismatch: %q", bucketAttrs.Name)
	assert(bucketAttrs.Location != "", "bucket location is empty")
	logf("GetBucketAttrs succeeded")

	updatedBucketAttrs, err := bucket.Update(ctx, storage.BucketAttrsToUpdate{
		StorageClass:      "NEARLINE",
		VersioningEnabled: true,
	})
	must("UpdateBucket", err)
	assert(strings.EqualFold(updatedBucketAttrs.StorageClass, "NEARLINE"), "bucket storage class mismatch: %q", updatedBucketAttrs.StorageClass)
	logf("UpdateBucket succeeded")

	foundBucket := false
	bucketIt := client.Buckets(ctx, projectID)
	for {
		attrs, err := bucketIt.Next()
		if err == iterator.Done {
			break
		}
		must("ListBuckets", err)
		if attrs.Name == bucketName {
			foundBucket = true
		}
	}
	assert(foundBucket, "bucket %q not found in project list", bucketName)
	logf("ListBuckets succeeded")

	object := bucket.Object(objectName)
	writer := object.NewWriter(ctx)
	writer.ContentType = "application/json"
	if _, err := writer.Write([]byte(`{"orderId":"o-1","status":"created"}`)); err != nil {
		_ = writer.Close()
		exitf("UploadObject failed: %v", err)
	}
	must("UploadObject", writer.Close())

	rangeReader, err := object.NewRangeReader(ctx, 0, 10)
	must("RangeReadObject", err)
	rangePayload, err := io.ReadAll(rangeReader)
	_ = rangeReader.Close()
	must("RangeReadObject", err)
	assert(len(rangePayload) > 0, "range read returned empty payload")
	logf("RangeReadObject succeeded")

	reader, err := object.NewReader(ctx)
	must("ReadObject", err)
	payload, err := io.ReadAll(reader)
	_ = reader.Close()
	must("ReadObject", err)
	assert(strings.Contains(string(payload), `"orderId":"o-1"`), "object payload mismatch: %s", string(payload))
	logf("ReadObject succeeded")

	objectAttrs, err := object.Attrs(ctx)
	must("GetObjectAttrs", err)
	assert(objectAttrs.Bucket == bucketName, "object bucket mismatch: %q", objectAttrs.Bucket)
	assert(objectAttrs.Generation > 0, "object generation is invalid")
	logf("GetObjectAttrs succeeded")

	updatedObjectAttrs, err := object.Update(ctx, storage.ObjectAttrsToUpdate{
		ContentType: "application/vnd.stackyard+json",
		Metadata: map[string]string{
			"env": "test",
		},
	})
	must("UpdateObject", err)
	assert(updatedObjectAttrs.ContentType == "application/vnd.stackyard+json", "object contentType mismatch: %q", updatedObjectAttrs.ContentType)
	logf("UpdateObject succeeded")

	foundObject := false
	objectIt := bucket.Objects(ctx, &storage.Query{Prefix: "orders/"})
	for {
		attrs, err := objectIt.Next()
		if err == iterator.Done {
			break
		}
		must("ListObjects", err)
		if attrs.Name == objectName {
			foundObject = true
		}
	}
	assert(foundObject, "object %q not found in list", objectName)
	logf("ListObjects succeeded")

	copiedObject := bucket.Object("orders/copied.json")
	_, err = copiedObject.CopierFrom(object).Run(ctx)
	must("CopyObject", err)
	logf("CopyObject succeeded")

	composed := bucket.Object(composedObject)
	composedAttrs, err := composed.ComposerFrom(object, copiedObject).Run(ctx)
	must("ComposeObject", err)
	assert(composedAttrs.Name == composedObject, "composed object name mismatch: %q", composedAttrs.Name)
	logf("ComposeObject succeeded")

	movedAttrs, err := copiedObject.Move(ctx, storage.MoveObjectDestination{Object: movedObject})
	must("MoveObject", err)
	assert(movedAttrs.Name == movedObject, "moved object name mismatch: %q", movedAttrs.Name)
	logf("MoveObject succeeded")

	moved := bucket.Object(movedObject)
	must("DeleteMovedObject", moved.Delete(ctx))
	restoredAttrs, err := moved.Generation(movedAttrs.Generation).Restore(ctx, &storage.RestoreOptions{})
	must("RestoreMovedObject", err)
	assert(restoredAttrs.Name == movedObject, "restored object mismatch: %q", restoredAttrs.Name)
	logf("RestoreMovedObject succeeded")

	bucketACL := bucket.ACL()
	must("SetBucketACL", bucketACL.Set(ctx, storage.AllUsers, storage.RoleReader))
	bucketACLRules, err := bucketACL.List(ctx)
	must("ListBucketACL", err)
	assert(len(bucketACLRules) > 0, "bucket ACL list is empty")
	must("DeleteBucketACL", bucketACL.Delete(ctx, storage.AllUsers))
	logf("BucketACL succeeded")

	objectACL := object.ACL()
	must("SetObjectACL", objectACL.Set(ctx, storage.AllUsers, storage.RoleReader))
	objectACLRules, err := objectACL.List(ctx)
	must("ListObjectACL", err)
	assert(len(objectACLRules) > 0, "object ACL list is empty")
	must("DeleteObjectACL", objectACL.Delete(ctx, storage.AllUsers))
	logf("ObjectACL succeeded")

	iamHandle := bucket.IAM()
	policy, err := iamHandle.Policy(ctx)
	must("GetBucketIAMPolicy", err)
	policy.Add("user:stackyard@example.com", iam.RoleName("roles/storage.objectViewer"))
	must("SetBucketIAMPolicy", iamHandle.SetPolicy(ctx, policy))
	permissions, err := iamHandle.TestPermissions(ctx, []string{"storage.buckets.get", "storage.objects.list"})
	must("TestBucketIAMPermissions", err)
	assert(len(permissions) > 0, "expected non-empty IAM permissions")
	logf("BucketIAM succeeded")

	notification, err := bucket.AddNotification(ctx, &storage.Notification{
		TopicProjectID: projectID,
		TopicID:        "stackyard-storage-topic",
		EventTypes:     []string{"OBJECT_FINALIZE"},
		PayloadFormat:  "JSON_API_V1",
	})
	must("AddNotification", err)
	assert(notification.ID != "", "notification ID is empty")
	notifications, err := bucket.Notifications(ctx)
	must("ListNotifications", err)
	assert(len(notifications) > 0, "notification list is empty")
	must("DeleteNotification", bucket.DeleteNotification(ctx, notification.ID))
	logf("BucketNotifications succeeded")

	serviceAccountEmail, err := client.ServiceAccount(ctx, projectID)
	must("GetServiceAccount", err)
	assert(strings.Contains(serviceAccountEmail, "@"), "service account email is invalid: %q", serviceAccountEmail)
	logf("GetServiceAccount succeeded")

	hmacKey, err := client.CreateHMACKey(ctx, projectID, serviceAccountEmail)
	must("CreateHMACKey", err)
	assert(hmacKey.AccessID != "", "hmac accessId is empty")
	hmacIter := client.ListHMACKeys(ctx, projectID)
	_, err = hmacIter.Next()
	must("ListHMACKeys", err)
	hmacHandle := client.HMACKeyHandle(projectID, hmacKey.AccessID)
	_, err = hmacHandle.Get(ctx)
	must("GetHMACKey", err)
	_, err = hmacHandle.Update(ctx, storage.HMACKeyAttrsToUpdate{State: storage.Inactive})
	must("UpdateHMACKey", err)
	must("DeleteHMACKey", hmacHandle.Delete(ctx))
	logf("HMACKeyLifecycle succeeded")

	must("DeleteComposedObject", composed.Delete(ctx))
	must("DeleteMovedObjectAfterRestore", moved.Delete(ctx))
	must("DeletePrimaryObject", object.Delete(ctx))
	must("DeleteBucket", bucket.Delete(ctx))

	fmt.Println("Done.")
}

func must(name string, err error) {
	if err != nil {
		exitf("%s failed: %v", name, err)
	}
	logf("%s succeeded", name)
}

func assert(condition bool, format string, args ...any) {
	if condition {
		return
	}
	exitf(format, args...)
}

func closeClient(client *storage.Client) {
	if err := client.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close storage client: %v\n", err)
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

type stackyardHeaderTransport struct {
	base        http.RoundTripper
	serviceName string
}

func (t stackyardHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	clone.Header.Set("X-Stackyard-GCP-Service", t.serviceName)
	return base.RoundTrip(clone)
}
