package server

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

func ecrpDigest(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// ecrpPushLayer uploads data as one layer and returns its digest.
func ecrpPushLayer(t *testing.T, ts *httptest.Server, repo string, data []byte) string {
	t.Helper()
	up := ecrpOK(t, ts, "InitiateLayerUpload", `{"repositoryName":"`+repo+`"}`)
	id := up["uploadId"].(string)
	if len(id) != 36 || up["partSize"].(float64) <= 0 {
		t.Fatalf("initiate: %v", up)
	}
	ecrpOK(t, ts, "UploadLayerPart", fmt.Sprintf(`{"repositoryName":%q,"uploadId":%q,"partFirstByte":0,"partLastByte":%d,"layerPartBlob":%q}`, repo, id, len(data)-1, base64.StdEncoding.EncodeToString(data)))
	digest := ecrpDigest(data)
	done := ecrpOK(t, ts, "CompleteLayerUpload", fmt.Sprintf(`{"repositoryName":%q,"uploadId":%q,"layerDigests":[%q]}`, repo, id, digest))
	if done["layerDigest"] != digest {
		t.Fatalf("complete: %v", done)
	}
	return digest
}

func ecrpManifest(config, layer string, layerSize int) string {
	return fmt.Sprintf(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":2,"digest":%q},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":%d,"digest":%q}]}`, config, layerSize, layer)
}

func TestECRPublicStage1RepositoriesAndCatalog(t *testing.T) {
	ts := newECRPublicTestServer(t)
	created := ecrpOK(t, ts, "CreateRepository", `{"repositoryName":"team/web","catalogData":{"description":"web server","architectures":["ARM 64"],"operatingSystems":["Linux"],"aboutText":"about","logoImageBlob":"iVBORw0K"},"tags":[{"Key":"env","Value":"dev"}]}`)
	repo := created["repository"].(map[string]any)
	if repo["repositoryArn"] != "arn:aws:ecr-public::123456789012:repository/team/web" || repo["repositoryUri"] != "public.ecr.aws/stackyard/team/web" {
		t.Fatalf("repository: %v", repo)
	}
	catalog := created["catalogData"].(map[string]any)
	if catalog["description"] != "web server" || !strings.HasPrefix(catalog["logoUrl"].(string), "https://public.ecr.aws/") {
		t.Fatalf("catalog: %v", catalog)
	}

	for name, body := range map[string]string{
		"duplicate":          `{"repositoryName":"team/web"}`,
		"bad name":           `{"repositoryName":"Bad_Name"}`,
		"aws tag":            `{"repositoryName":"other","tags":[{"Key":"aws:x","Value":"y"}]}`,
		"long description":   `{"repositoryName":"other","catalogData":{"description":"` + strings.Repeat("d", 1025) + `"}}`,
		"too many platforms": `{"repositoryName":"other","catalogData":{"architectures":[` + strings.TrimSuffix(strings.Repeat(`"x",`, 51), ",") + `]}}`,
	} {
		want := map[string]string{"duplicate": "RepositoryAlreadyExistsException", "aws tag": "InvalidTagParameterException"}[name]
		if want == "" {
			want = "InvalidParameterException"
		}
		if code := ecrpFail(t, ts, "CreateRepository", body); code != want {
			t.Fatalf("%s: %s (want %s)", name, code, want)
		}
	}
	if code := ecrpFail(t, ts, "DescribeRepositories", `{"repositoryNames":["missing"]}`); code != "RepositoryNotFoundException" {
		t.Fatalf("missing repository: %s", code)
	}
	if code := ecrpFail(t, ts, "DescribeRepositories", `{"maxResults":0}`); code != "InvalidParameterException" {
		t.Fatalf("maxResults 0: %s", code)
	}
	if code := ecrpFail(t, ts, "DescribeRepositories", `{"registryId":"999999999999"}`); code != "RepositoryNotFoundException" {
		t.Fatalf("other registry: %s", code)
	}

	ecrpOK(t, ts, "CreateRepository", `{"repositoryName":"aaa"}`)
	page := ecrpOK(t, ts, "DescribeRepositories", `{"maxResults":1}`)
	if len(page["repositories"].([]any)) != 1 || page["nextToken"] != "1" {
		t.Fatalf("page 1: %v", page)
	}
	page = ecrpOK(t, ts, "DescribeRepositories", `{"maxResults":1,"nextToken":"1"}`)
	if page["repositories"].([]any)[0].(map[string]any)["repositoryName"] != "team/web" || page["nextToken"] != nil {
		t.Fatalf("page 2: %v", page)
	}

	// Catalog updates keep fields that aren't given.
	put := ecrpOK(t, ts, "PutRepositoryCatalogData", `{"repositoryName":"team/web","catalogData":{"usageText":"docker pull"}}`)["catalogData"].(map[string]any)
	if put["description"] != "web server" || put["usageText"] != "docker pull" {
		t.Fatalf("catalog update: %v", put)
	}
	got := ecrpOK(t, ts, "GetRepositoryCatalogData", `{"repositoryName":"team/web"}`)["catalogData"].(map[string]any)
	if got["usageText"] != "docker pull" {
		t.Fatalf("catalog get: %v", got)
	}

	if ecrpOK(t, ts, "PutRegistryCatalogData", `{"displayName":"Stackyard"}`)["registryCatalogData"].(map[string]any)["displayName"] != "Stackyard" {
		t.Fatal("registry catalog put")
	}
	if code := ecrpFail(t, ts, "PutRegistryCatalogData", `{"displayName":"`+strings.Repeat("n", 101)+`"}`); code != "InvalidParameterException" {
		t.Fatalf("long display name: %s", code)
	}
	reg := ecrpOK(t, ts, "DescribeRegistries", `{}`)["registries"].([]any)[0].(map[string]any)
	alias := reg["aliases"].([]any)[0].(map[string]any)
	if reg["registryUri"] != "public.ecr.aws/stackyard" || alias["defaultRegistryAlias"] != true {
		t.Fatalf("registry: %v", reg)
	}
	auth := ecrpOK(t, ts, "GetAuthorizationToken", `{}`)["authorizationData"].(map[string]any)
	if auth["authorizationToken"] == "" {
		t.Fatalf("auth: %v", auth)
	}

	// Policies.
	if code := ecrpFail(t, ts, "GetRepositoryPolicy", `{"repositoryName":"team/web"}`); code != "RepositoryPolicyNotFoundException" {
		t.Fatalf("no policy: %s", code)
	}
	if code := ecrpFail(t, ts, "SetRepositoryPolicy", `{"repositoryName":"team/web","policyText":"not json"}`); code != "InvalidParameterException" {
		t.Fatalf("bad policy: %s", code)
	}
	policy := `{\"Version\":\"2012-10-17\",\"Statement\":[]}`
	ecrpOK(t, ts, "SetRepositoryPolicy", `{"repositoryName":"team/web","policyText":"`+policy+`"}`)
	if ecrpOK(t, ts, "GetRepositoryPolicy", `{"repositoryName":"team/web"}`)["policyText"] == "" {
		t.Fatal("policy get")
	}
	ecrpOK(t, ts, "DeleteRepositoryPolicy", `{"repositoryName":"team/web"}`)

	// Tags by public ARN.
	arn := "arn:aws:ecr-public::123456789012:repository/team/web"
	ecrpOK(t, ts, "TagResource", `{"resourceArn":"`+arn+`","tags":[{"Key":"team","Value":"a"}]}`)
	tags := ecrpOK(t, ts, "ListTagsForResource", `{"resourceArn":"`+arn+`"}`)["tags"].([]any)
	if len(tags) != 2 {
		t.Fatalf("tags: %v", tags)
	}
	ecrpOK(t, ts, "UntagResource", `{"resourceArn":"`+arn+`","tagKeys":["env"]}`)
	if len(ecrpOK(t, ts, "ListTagsForResource", `{"resourceArn":"`+arn+`"}`)["tags"].([]any)) != 1 {
		t.Fatal("untag")
	}
	if code := ecrpFail(t, ts, "ListTagsForResource", `{"resourceArn":"arn:aws:ecr:us-east-1:123456789012:repository/team/web"}`); code != "InvalidParameterException" {
		t.Fatalf("private ARN: %s", code)
	}
	many := make([]string, 0, 50)
	for i := 0; i < 50; i++ {
		many = append(many, fmt.Sprintf(`{"Key":"k%d","Value":"v"}`, i))
	}
	if code := ecrpFail(t, ts, "TagResource", `{"resourceArn":"`+arn+`","tags":[`+strings.Join(many, ",")+`]}`); code != "TooManyTagsException" {
		t.Fatalf("51 tags: %s", code)
	}
}

func TestECRPublicStage1LayersAndImages(t *testing.T) {
	ts := newECRPublicTestServer(t)
	ecrpOK(t, ts, "CreateRepository", `{"repositoryName":"app"}`)

	config := ecrpPushLayer(t, ts, "app", []byte("{}"))
	layer := ecrpPushLayer(t, ts, "app", []byte("layer-bytes"))

	// Upload errors.
	up := ecrpOK(t, ts, "InitiateLayerUpload", `{"repositoryName":"app"}`)["uploadId"].(string)
	if code := ecrpFail(t, ts, "CompleteLayerUpload", `{"repositoryName":"app","uploadId":"`+up+`","layerDigests":["sha256:00"]}`); code != "EmptyUploadException" {
		t.Fatalf("empty upload: %s", code)
	}
	if code := ecrpFail(t, ts, "UploadLayerPart", `{"repositoryName":"app","uploadId":"`+up+`","partFirstByte":5,"partLastByte":5,"layerPartBlob":"eA=="}`); code != "InvalidLayerPartException" {
		t.Fatalf("gap in parts: %s", code)
	}
	if code := ecrpFail(t, ts, "UploadLayerPart", `{"repositoryName":"app","uploadId":"00000000-0000-0000-0000-000000000000","partFirstByte":0,"partLastByte":0,"layerPartBlob":"eA=="}`); code != "UploadNotFoundException" {
		t.Fatalf("unknown upload: %s", code)
	}
	if code := ecrpFail(t, ts, "UploadLayerPart", fmt.Sprintf(`{"repositoryName":"app","uploadId":%q,"partFirstByte":0,"partLastByte":%d,"layerPartBlob":"eA=="}`, up, ecrPublicMaxLayer)); code != "InvalidLayerPartException" {
		t.Fatalf("length mismatch: %s", code)
	}
	ecrpOK(t, ts, "UploadLayerPart", `{"repositoryName":"app","uploadId":"`+up+`","partFirstByte":0,"partLastByte":10,"layerPartBlob":"`+base64.StdEncoding.EncodeToString([]byte("layer-bytes"))+`"}`)
	if code := ecrpFail(t, ts, "CompleteLayerUpload", `{"repositoryName":"app","uploadId":"`+up+`","layerDigests":["`+layer+`"]}`); code != "LayerAlreadyExistsException" {
		t.Fatalf("duplicate layer: %s", code)
	}

	check := ecrpOK(t, ts, "BatchCheckLayerAvailability", `{"repositoryName":"app","layerDigests":["`+layer+`","sha256:`+strings.Repeat("0", 64)+`","bad digest"]}`)
	if len(check["layers"].([]any)) != 2 || len(check["failures"].([]any)) != 1 {
		t.Fatalf("availability: %v", check)
	}

	if code := ecrpFail(t, ts, "PutImage", `{"repositoryName":"app","imageManifest":"`+strings.ReplaceAll(ecrpManifest(config, "sha256:"+strings.Repeat("1", 64), 1), `"`, `\"`)+`"}`); code != "LayersNotFoundException" {
		t.Fatalf("missing layer: %s", code)
	}
	if code := ecrpFail(t, ts, "PutImage", `{"repositoryName":"app","imageManifest":"not json"}`); code != "InvalidParameterException" {
		t.Fatalf("bad manifest: %s", code)
	}
	manifest := ecrpManifest(config, layer, 11)
	escaped := strings.ReplaceAll(manifest, `"`, `\"`)
	put := ecrpOK(t, ts, "PutImage", `{"repositoryName":"app","imageManifest":"`+escaped+`","imageTag":"v1"}`)["image"].(map[string]any)
	digest := put["imageId"].(map[string]any)["imageDigest"].(string)
	if digest != ecrpDigest([]byte(manifest)) {
		t.Fatalf("image digest: %v", put)
	}
	if code := ecrpFail(t, ts, "PutImage", `{"repositoryName":"app","imageManifest":"`+escaped+`","imageTag":"v1"}`); code != "ImageAlreadyExistsException" {
		t.Fatalf("same image and tag: %s", code)
	}
	ecrpOK(t, ts, "PutImage", `{"repositoryName":"app","imageManifest":"`+escaped+`","imageTag":"latest"}`)

	details := ecrpOK(t, ts, "DescribeImages", `{"repositoryName":"app"}`)["imageDetails"].([]any)
	if len(details) != 1 || details[0].(map[string]any)["imageSizeInBytes"].(float64) != 13 {
		t.Fatalf("describe images: %v", details)
	}
	if code := ecrpFail(t, ts, "DescribeImages", `{"repositoryName":"app","imageIds":[{"imageTag":"nope"}]}`); code != "ImageNotFoundException" {
		t.Fatalf("missing image: %s", code)
	}
	tags := ecrpOK(t, ts, "DescribeImageTags", `{"repositoryName":"app","maxResults":1}`)
	first := tags["imageTagDetails"].([]any)[0].(map[string]any)
	if first["imageTag"] != "latest" || tags["nextToken"] != "1" || first["imageDetail"].(map[string]any)["imageDigest"] != digest {
		t.Fatalf("image tags: %v", tags)
	}

	if code := ecrpFail(t, ts, "DeleteRepository", `{"repositoryName":"app"}`); code != "RepositoryNotEmptyException" {
		t.Fatalf("non-empty delete: %s", code)
	}
	deleted := ecrpOK(t, ts, "BatchDeleteImage", `{"repositoryName":"app","imageIds":[{"imageDigest":"`+digest+`"},{"imageTag":"nope"}]}`)
	if len(deleted["imageIds"].([]any)) != 1 || len(deleted["failures"].([]any)) != 1 {
		t.Fatalf("batch delete: %v", deleted)
	}
	ecrpOK(t, ts, "DeleteRepository", `{"repositoryName":"app"}`)
	if code := ecrpFail(t, ts, "GetRepositoryCatalogData", `{"repositoryName":"app"}`); code != "RepositoryNotFoundException" {
		t.Fatalf("deleted repository: %s", code)
	}
}

// Layer and manifest bytes count against one registry-wide budget, released
// when the repository is deleted.
func TestECRPublicStage1StorageLimit(t *testing.T) {
	st := newECRPublicStore()
	call := func(op string, in ecrPublicInput) *ecrPublicError {
		_, err := st.handle(op, in)
		return err
	}
	if err := call("CreateRepository", ecrPublicInput{RepositoryName: "big"}); err != nil {
		t.Fatal(err.msg)
	}
	body, _ := st.handle("InitiateLayerUpload", ecrPublicInput{RepositoryName: "big"})
	id := strings.Split(strings.Split(string(body), `"uploadId":"`)[1], `"`)[0]
	first, last := int64(0), int64(3)
	part := ecrPublicInput{RepositoryName: "big", UploadID: id, PartFirstByte: &first, PartLastByte: &last, LayerPartBlob: []byte("abcd")}
	st.storedBytes = ecrPublicMaxStored - 2
	if err := call("UploadLayerPart", part); err == nil || err.code != "LimitExceededException" {
		t.Fatalf("over the storage budget: %v", err)
	}
	st.storedBytes = 0
	if err := call("UploadLayerPart", part); err != nil {
		t.Fatal(err.msg)
	}
	if err := call("DeleteRepository", ecrPublicInput{RepositoryName: "big", Force: true}); err != nil {
		t.Fatal(err.msg)
	}
	if st.storedBytes != 0 {
		t.Fatalf("bytes after delete: %d", st.storedBytes)
	}

	// In-flight uploads are capped, since they never expire here.
	if err := call("CreateRepository", ecrPublicInput{RepositoryName: "many"}); err != nil {
		t.Fatal(err.msg)
	}
	for i := 0; i < ecrPublicMaxUploads; i++ {
		if err := call("InitiateLayerUpload", ecrPublicInput{RepositoryName: "many"}); err != nil {
			t.Fatalf("upload %d: %s", i, err.msg)
		}
	}
	if err := call("InitiateLayerUpload", ecrPublicInput{RepositoryName: "many"}); err == nil || err.code != "LimitExceededException" {
		t.Fatalf("upload past the cap: %v", err)
	}
	if err := call("DeleteRepository", ecrPublicInput{RepositoryName: "many", Force: true}); err != nil || len(st.uploads) != 0 {
		t.Fatalf("uploads after delete: %d %v", len(st.uploads), err)
	}
}
