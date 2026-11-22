package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	ecrsvc "github.com/stackyard/stackyard/internal/services/ecr"
)

const (
	ecrPublicAccountID = ecrsvc.DefaultAccountID
	// ponytail: one fixed default alias; real registries get a random 8-char alias.
	ecrPublicAlias        = "stackyard"
	ecrPublicPartSize     = int64(10 << 20)  // advertised part size; fits the 25 MiB body limit as base64
	ecrPublicMaxLayer     = int64(256 << 20) // bytes in one layer upload
	ecrPublicMaxStored    = int64(1 << 30)   // layer bytes held across all repositories
	ecrPublicMaxRepos     = 1000             // ponytail: AWS allows 10,000; enough for an emulator
	ecrPublicMaxTags      = 50               // tags per repository
	ecrPublicMaxPolicy    = 10240            // repository policy characters
	ecrPublicMaxManifest  = 4 << 20          // ImageManifest max length
	ecrPublicMaxLogoBytes = 512000           // LogoImageBlob max length
	ecrPublicMaxPageWalk  = 1000             // pages read from the engine for DescribeImageTags
	// ponytail: uploads never expire here (AWS drops them after 24 hours), so
	// in-flight ones are capped instead; deleting the repository frees them.
	ecrPublicMaxUploads = 1000
)

var (
	ecrPublicRepoNamePattern = regexp.MustCompile(`^(?:[a-z0-9]+(?:[._-][a-z0-9]+)*/)*[a-z0-9]+(?:[._-][a-z0-9]+)*$`)
	ecrPublicRegistryPattern = regexp.MustCompile(`^[0-9]{12}$`)
	ecrPublicRepoARNPattern  = regexp.MustCompile(`^arn:aws:ecr-public::([0-9]{12}):repository/(.+)$`)
	ecrPublicDigestPattern   = regexp.MustCompile(`^[a-zA-Z0-9-_+.]+:[a-fA-F0-9]+$`)
)

// ecrPublicStore puts the public registry's model (ecr-public ARNs and URIs,
// catalog data, the registry alias, UUID upload ids, byte caps) over a
// private-ECR engine instance, which keeps repositories, layers, manifests,
// tags and policies. The engine is a separate instance from s.ecr, so public
// and private repositories never mix.
type ecrPublicStore struct {
	mu          sync.Mutex
	engine      *ecrsvc.Service
	catalog     map[string]ecrPublicCatalog // by repository name
	displayName string
	uploads     map[string]*ecrPublicUpload // by public upload id
	repoBytes   map[string]int64            // completed layer bytes per repository
	storedBytes int64                       // completed layers plus in-flight uploads
	images      map[string]ecrPublicImage   // repository + "@" + digest
}

// ecrPublicImage is what the store tracks per pushed image: the layer bytes
// reported as imageSizeInBytes, and the manifest bytes it holds (counted in
// storedBytes so pushes are bounded too).
type ecrPublicImage struct {
	layerBytes    int64
	manifestBytes int64
}

type ecrPublicCatalog struct {
	Description      string
	Architectures    []string
	OperatingSystems []string
	LogoURL          string
	AboutText        string
	UsageText        string
}

type ecrPublicUpload struct {
	repo     string
	engineID string
	received int64
}

func newECRPublicStore() *ecrPublicStore {
	return &ecrPublicStore{
		engine:    ecrsvc.NewService(),
		catalog:   map[string]ecrPublicCatalog{},
		uploads:   map[string]*ecrPublicUpload{},
		repoBytes: map[string]int64{},
		images:    map[string]ecrPublicImage{},
	}
}

func ecrPublicRepositoryARN(name string) string {
	return fmt.Sprintf("arn:aws:ecr-public::%s:repository/%s", ecrPublicAccountID, name)
}

func ecrPublicEngineARN(name string) string {
	return fmt.Sprintf("arn:aws:ecr:%s:%s:repository/%s", ecrsvc.DefaultRegion, ecrsvc.DefaultAccountID, name)
}

func ecrPublicRegistryURI() string { return "public.ecr.aws/" + ecrPublicAlias }

func ecrPublicTime(t time.Time) float64 { return float64(t.UnixNano()) / 1e9 }

// handle runs one operation and marshals its result while the store lock is
// held, so nothing read from the store is used after unlock.
func (st *ecrPublicStore) handle(action string, in ecrPublicInput) ([]byte, *ecrPublicError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if in.RegistryID != nil {
		if err := ecrPublicCheckRegistry(action, *in.RegistryID); err != nil {
			return nil, err
		}
	}
	if in.MaxResults != nil && (*in.MaxResults < 1 || *in.MaxResults > 1000) {
		return nil, ecrPublicInvalid("maxResults must be between 1 and 1000")
	}
	var out any
	var err *ecrPublicError
	switch action {
	case "GetAuthorizationToken":
		out, err = st.getAuthorizationToken()
	case "CreateRepository":
		out, err = st.createRepository(in)
	case "DescribeRepositories":
		out, err = st.describeRepositories(in)
	case "DeleteRepository":
		out, err = st.deleteRepository(in)
	case "GetRepositoryCatalogData":
		out, err = st.getRepositoryCatalogData(in)
	case "PutRepositoryCatalogData":
		out, err = st.putRepositoryCatalogData(in)
	case "GetRegistryCatalogData":
		out = map[string]any{"registryCatalogData": map[string]any{"displayName": st.displayName}}
	case "PutRegistryCatalogData":
		out, err = st.putRegistryCatalogData(in)
	case "DescribeRegistries":
		out, err = st.describeRegistries(in)
	case "SetRepositoryPolicy", "GetRepositoryPolicy", "DeleteRepositoryPolicy":
		out, err = st.repositoryPolicy(action, in)
	case "ListTagsForResource", "TagResource", "UntagResource":
		out, err = st.tags(action, in)
	case "InitiateLayerUpload":
		out, err = st.initiateLayerUpload(in)
	case "UploadLayerPart":
		out, err = st.uploadLayerPart(in)
	case "CompleteLayerUpload":
		out, err = st.completeLayerUpload(in)
	case "BatchCheckLayerAvailability":
		out, err = st.batchCheckLayerAvailability(in)
	case "PutImage":
		out, err = st.putImage(in)
	case "DescribeImages":
		out, err = st.describeImages(in)
	case "DescribeImageTags":
		out, err = st.describeImageTags(in)
	case "BatchDeleteImage":
		out, err = st.batchDeleteImage(in)
	default:
		return nil, ecrPublicInvalid("unknown action " + action)
	}
	if err != nil {
		return nil, err
	}
	body, mErr := json.Marshal(out)
	if mErr != nil {
		return nil, &ecrPublicError{http.StatusInternalServerError, "ServerException", mErr.Error()}
	}
	return body, nil
}

// ecrPublicCheckRegistry validates an explicit registryId: it must be this
// account's. Upload and push operations model RegistryNotFoundException.
func ecrPublicCheckRegistry(action, id string) *ecrPublicError {
	if !ecrPublicRegistryPattern.MatchString(id) {
		return ecrPublicInvalid("registryId must be a 12-digit account id")
	}
	if id == ecrPublicAccountID {
		return nil
	}
	switch action {
	case "BatchCheckLayerAvailability", "CompleteLayerUpload", "InitiateLayerUpload", "PutImage", "UploadLayerPart":
		return ecrPublicFail("RegistryNotFoundException", "registry "+id+" does not exist")
	}
	return ecrPublicFail("RepositoryNotFoundException", "the repository does not exist in registry "+id)
}

func ecrPublicEngineErr(err error) *ecrPublicError {
	switch {
	case errors.Is(err, ecrsvc.ErrRepositoryNotFound):
		return ecrPublicFail("RepositoryNotFoundException", "the repository does not exist")
	case errors.Is(err, ecrsvc.ErrRepositoryAlreadyExists):
		return ecrPublicFail("RepositoryAlreadyExistsException", "the repository already exists")
	case errors.Is(err, ecrsvc.ErrRepositoryPolicyNotFound):
		return ecrPublicFail("RepositoryPolicyNotFoundException", "the repository has no policy")
	case errors.Is(err, ecrsvc.ErrUploadNotFound):
		return ecrPublicFail("UploadNotFoundException", "the upload does not exist")
	case errors.Is(err, ecrsvc.ErrImageNotFound):
		return ecrPublicFail("ImageNotFoundException", "the image does not exist")
	case errors.Is(err, ecrsvc.ErrImageDigestDoesNotMatch):
		return ecrPublicFail("ImageDigestDoesNotMatchException", "imageDigest does not match the manifest")
	case errors.Is(err, ecrsvc.ErrInvalidParameter):
		return ecrPublicInvalid(err.Error())
	}
	return &ecrPublicError{http.StatusInternalServerError, "ServerException", err.Error()}
}

func ecrPublicRepoName(name string) *ecrPublicError {
	if len(name) < 2 || len(name) > 205 || !ecrPublicRepoNamePattern.MatchString(name) {
		return ecrPublicInvalid("repositoryName must be 2-205 lowercase characters matching (?:[a-z0-9]+(?:[._-][a-z0-9]+)*/)*[a-z0-9]+(?:[._-][a-z0-9]+)*")
	}
	return nil
}

// repository returns the engine's repository or RepositoryNotFoundException.
func (st *ecrPublicStore) repository(name string) (ecrsvc.Repository, *ecrPublicError) {
	if err := ecrPublicRepoName(name); err != nil {
		return ecrsvc.Repository{}, err
	}
	repos, _, err := st.engine.DescribeRepositories([]string{name}, "", 0)
	if err != nil || len(repos) == 0 {
		return ecrsvc.Repository{}, ecrPublicFail("RepositoryNotFoundException", "repository "+name+" does not exist")
	}
	return repos[0], nil
}

func ecrPublicRepositoryOut(repo ecrsvc.Repository) map[string]any {
	return map[string]any{
		"repositoryArn":  ecrPublicRepositoryARN(repo.RepositoryName),
		"registryId":     ecrPublicAccountID,
		"repositoryName": repo.RepositoryName,
		"repositoryUri":  ecrPublicRegistryURI() + "/" + repo.RepositoryName,
		"createdAt":      ecrPublicTime(repo.CreatedAt),
	}
}

func (st *ecrPublicStore) getAuthorizationToken() (any, *ecrPublicError) {
	data, err := st.engine.GetAuthorizationToken(nil)
	if err != nil || len(data) == 0 {
		return nil, &ecrPublicError{http.StatusInternalServerError, "ServerException", "no authorization token"}
	}
	return map[string]any{"authorizationData": map[string]any{
		"authorizationToken": data[0].AuthorizationToken,
		"expiresAt":          ecrPublicTime(data[0].ExpiresAt),
	}}, nil
}

func ecrPublicCheckTags(tags []ecrPublicTag) (map[string]string, *ecrPublicError) {
	if len(tags) > 200 {
		return nil, ecrPublicFail("TooManyTagsException", "a request can carry at most 200 tags")
	}
	out := make(map[string]string, len(tags))
	for _, t := range tags {
		if len(t.Key) < 1 || len(t.Key) > 128 || len(t.Value) > 256 {
			return nil, ecrPublicFail("InvalidTagParameterException", "tag keys must be 1-128 characters and values at most 256")
		}
		if strings.HasPrefix(strings.ToLower(t.Key), "aws:") {
			return nil, ecrPublicFail("InvalidTagParameterException", "tag keys can't start with aws:")
		}
		out[t.Key] = t.Value
	}
	return out, nil
}

func (st *ecrPublicStore) createRepository(in ecrPublicInput) (any, *ecrPublicError) {
	if err := ecrPublicRepoName(in.RepositoryName); err != nil {
		return nil, err
	}
	tags, terr := ecrPublicCheckTags(in.Tags)
	if terr != nil {
		return nil, terr
	}
	if len(tags) > ecrPublicMaxTags {
		return nil, ecrPublicFail("TooManyTagsException", fmt.Sprintf("a repository can have at most %d tags", ecrPublicMaxTags))
	}
	catalog, cerr := ecrPublicCatalogFrom(in.RepositoryName, ecrPublicCatalog{}, in.CatalogData)
	if cerr != nil {
		return nil, cerr
	}
	if len(st.catalog) >= ecrPublicMaxRepos {
		return nil, ecrPublicFail("LimitExceededException", fmt.Sprintf("a registry can hold at most %d repositories", ecrPublicMaxRepos))
	}
	repo, err := st.engine.CreateRepository(in.RepositoryName, "MUTABLE", nil, "", "", tags)
	if err != nil {
		return nil, ecrPublicEngineErr(err)
	}
	st.catalog[repo.RepositoryName] = catalog
	return map[string]any{"repository": ecrPublicRepositoryOut(repo), "catalogData": catalog.out()}, nil
}

func (c ecrPublicCatalog) out() map[string]any {
	out := map[string]any{
		"architectures":        append([]string{}, c.Architectures...),
		"operatingSystems":     append([]string{}, c.OperatingSystems...),
		"marketplaceCertified": false,
	}
	for k, v := range map[string]string{"description": c.Description, "aboutText": c.AboutText, "usageText": c.UsageText, "logoUrl": c.LogoURL} {
		if v != "" {
			out[k] = v
		}
	}
	return out
}

// ecrPublicCatalogFrom applies catalog input over prev, checking the modeled
// limits. A logo blob is stored only as a URL. ponytail: nothing serves it.
func ecrPublicCatalogFrom(repo string, prev ecrPublicCatalog, in *ecrPublicCatalogInput) (ecrPublicCatalog, *ecrPublicError) {
	if in == nil {
		return prev, nil
	}
	list := func(name string, v []string) ([]string, *ecrPublicError) {
		if len(v) > 50 {
			return nil, ecrPublicInvalid(name + " can have at most 50 entries")
		}
		for _, s := range v {
			if len(s) < 1 || len(s) > 50 {
				return nil, ecrPublicInvalid(name + " entries must be 1-50 characters")
			}
		}
		return append([]string{}, v...), nil
	}
	text := func(name string, v *string, max int, cur string) (string, *ecrPublicError) {
		if v == nil {
			return cur, nil
		}
		if len(*v) > max {
			return "", ecrPublicInvalid(fmt.Sprintf("%s can be at most %d characters", name, max))
		}
		return *v, nil
	}
	out := ecrPublicCatalog{LogoURL: prev.LogoURL}
	var err *ecrPublicError
	if out.Description, err = text("description", in.Description, 1024, prev.Description); err != nil {
		return out, err
	}
	if out.AboutText, err = text("aboutText", in.AboutText, 25600, prev.AboutText); err != nil {
		return out, err
	}
	if out.UsageText, err = text("usageText", in.UsageText, 25600, prev.UsageText); err != nil {
		return out, err
	}
	if out.Architectures, err = list("architectures", in.Architectures); err != nil {
		return out, err
	}
	if out.OperatingSystems, err = list("operatingSystems", in.OperatingSystems); err != nil {
		return out, err
	}
	if len(in.LogoImageBlob) > ecrPublicMaxLogoBytes {
		return out, ecrPublicInvalid(fmt.Sprintf("logoImageBlob can be at most %d bytes", ecrPublicMaxLogoBytes))
	}
	if len(in.LogoImageBlob) > 0 {
		sum := sha256.Sum256(in.LogoImageBlob)
		out.LogoURL = "https://" + ecrPublicRegistryURI() + "/logos/" + repo + "/" + hex.EncodeToString(sum[:8]) + ".png"
	}
	return out, nil
}

func ecrPublicPage(token string, maxResults *int32, total int) (int, int, *ecrPublicError) {
	start := 0
	if token != "" {
		n, err := strconv.ParseInt(token, 10, 32)
		if err != nil || n < 0 || int(n) > total {
			return 0, 0, ecrPublicInvalid("nextToken is invalid")
		}
		start = int(n)
	}
	limit := 100
	if maxResults != nil {
		limit = int(*maxResults)
	}
	return start, min(total, start+limit), nil
}

func ecrPublicNext(end, total int) any {
	if end < total {
		return strconv.Itoa(end)
	}
	return nil
}

func (st *ecrPublicStore) describeRepositories(in ecrPublicInput) (any, *ecrPublicError) {
	if len(in.RepositoryNames) > 100 {
		return nil, ecrPublicInvalid("repositoryNames can have at most 100 entries")
	}
	if len(in.RepositoryNames) > 0 && (in.NextToken != "" || in.MaxResults != nil) {
		return nil, ecrPublicInvalid("repositoryNames can't be combined with nextToken or maxResults")
	}
	var repos []ecrsvc.Repository
	if len(in.RepositoryNames) > 0 {
		for _, name := range in.RepositoryNames {
			repo, err := st.repository(name)
			if err != nil {
				return nil, err
			}
			repos = append(repos, repo)
		}
		return map[string]any{"repositories": ecrPublicRepos(repos)}, nil
	}
	all, _, err := st.engine.DescribeRepositories(nil, "", 1000)
	if err != nil {
		return nil, ecrPublicEngineErr(err)
	}
	start, end, perr := ecrPublicPage(in.NextToken, in.MaxResults, len(all))
	if perr != nil {
		return nil, perr
	}
	return map[string]any{"repositories": ecrPublicRepos(all[start:end]), "nextToken": ecrPublicNext(end, len(all))}, nil
}

func ecrPublicRepos(repos []ecrsvc.Repository) []map[string]any {
	out := make([]map[string]any, 0, len(repos))
	for _, r := range repos {
		out = append(out, ecrPublicRepositoryOut(r))
	}
	return out
}

func (st *ecrPublicStore) deleteRepository(in ecrPublicInput) (any, *ecrPublicError) {
	repo, err := st.repository(in.RepositoryName)
	if err != nil {
		return nil, err
	}
	if !in.Force {
		if images, _, ierr := st.engine.DescribeImages(repo.RepositoryName, nil, "ANY", "", 1); ierr == nil && len(images) > 0 {
			return nil, ecrPublicFail("RepositoryNotEmptyException", "the repository contains images; delete them or set force")
		}
	}
	if _, derr := st.engine.DeleteRepository(repo.RepositoryName, true); derr != nil {
		return nil, ecrPublicEngineErr(derr)
	}
	delete(st.catalog, repo.RepositoryName)
	st.storedBytes -= st.repoBytes[repo.RepositoryName]
	delete(st.repoBytes, repo.RepositoryName)
	for id, up := range st.uploads {
		if up.repo == repo.RepositoryName {
			st.storedBytes -= up.received
			delete(st.uploads, id)
		}
	}
	prefix := repo.RepositoryName + "@"
	for k, img := range st.images {
		if strings.HasPrefix(k, prefix) {
			st.storedBytes -= img.manifestBytes
			delete(st.images, k)
		}
	}
	return map[string]any{"repository": ecrPublicRepositoryOut(repo)}, nil
}

func (st *ecrPublicStore) getRepositoryCatalogData(in ecrPublicInput) (any, *ecrPublicError) {
	repo, err := st.repository(in.RepositoryName)
	if err != nil {
		return nil, err
	}
	return map[string]any{"catalogData": st.catalog[repo.RepositoryName].out()}, nil
}

func (st *ecrPublicStore) putRepositoryCatalogData(in ecrPublicInput) (any, *ecrPublicError) {
	repo, err := st.repository(in.RepositoryName)
	if err != nil {
		return nil, err
	}
	if in.CatalogData == nil {
		return nil, ecrPublicInvalid("catalogData is required")
	}
	catalog, cerr := ecrPublicCatalogFrom(repo.RepositoryName, st.catalog[repo.RepositoryName], in.CatalogData)
	if cerr != nil {
		return nil, cerr
	}
	st.catalog[repo.RepositoryName] = catalog
	return map[string]any{"catalogData": catalog.out()}, nil
}

func (st *ecrPublicStore) putRegistryCatalogData(in ecrPublicInput) (any, *ecrPublicError) {
	if in.DisplayName != nil {
		if len(*in.DisplayName) > 100 {
			return nil, ecrPublicInvalid("displayName can be at most 100 characters")
		}
		st.displayName = *in.DisplayName
	}
	return map[string]any{"registryCatalogData": map[string]any{"displayName": st.displayName}}, nil
}

func (st *ecrPublicStore) describeRegistries(in ecrPublicInput) (any, *ecrPublicError) {
	start, end, err := ecrPublicPage(in.NextToken, in.MaxResults, 1)
	if err != nil {
		return nil, err
	}
	registries := []map[string]any{{
		"registryId":  ecrPublicAccountID,
		"registryArn": fmt.Sprintf("arn:aws:ecr-public::%s:registry/%s", ecrPublicAccountID, ecrPublicAccountID),
		"registryUri": ecrPublicRegistryURI(),
		"verified":    false,
		"aliases": []map[string]any{{
			"name": ecrPublicAlias, "status": "ACTIVE", "primaryRegistryAlias": true, "defaultRegistryAlias": true,
		}},
	}}
	return map[string]any{"registries": registries[start:end], "nextToken": ecrPublicNext(end, 1)}, nil
}

func (st *ecrPublicStore) repositoryPolicy(action string, in ecrPublicInput) (any, *ecrPublicError) {
	repo, err := st.repository(in.RepositoryName)
	if err != nil {
		return nil, err
	}
	var text string
	var perr error
	switch action {
	case "SetRepositoryPolicy":
		if in.PolicyText == "" || len(in.PolicyText) > ecrPublicMaxPolicy || !json.Valid([]byte(in.PolicyText)) {
			return nil, ecrPublicInvalid(fmt.Sprintf("policyText must be a JSON policy of at most %d characters", ecrPublicMaxPolicy))
		}
		_, text, perr = st.engine.SetRepositoryPolicy(repo.RepositoryName, in.PolicyText, in.Force)
	case "GetRepositoryPolicy":
		_, text, perr = st.engine.GetRepositoryPolicy(repo.RepositoryName)
	default:
		_, text, perr = st.engine.DeleteRepositoryPolicy(repo.RepositoryName)
	}
	if perr != nil {
		return nil, ecrPublicEngineErr(perr)
	}
	return map[string]any{"registryId": ecrPublicAccountID, "repositoryName": repo.RepositoryName, "policyText": text}, nil
}

func (st *ecrPublicStore) tags(action string, in ecrPublicInput) (any, *ecrPublicError) {
	m := ecrPublicRepoARNPattern.FindStringSubmatch(in.ResourceArn)
	if m == nil {
		return nil, ecrPublicInvalid("resourceArn must be an ECR Public repository ARN")
	}
	if m[1] != ecrPublicAccountID {
		return nil, ecrPublicFail("RepositoryNotFoundException", "the repository does not exist")
	}
	repo, err := st.repository(m[2])
	if err != nil {
		return nil, err
	}
	engineARN := ecrPublicEngineARN(repo.RepositoryName)
	current, lerr := st.engine.ListTagsForResource(engineARN)
	if lerr != nil {
		return nil, ecrPublicEngineErr(lerr)
	}
	switch action {
	case "TagResource":
		tags, terr := ecrPublicCheckTags(in.Tags)
		if terr != nil {
			return nil, terr
		}
		if len(tags) == 0 {
			return nil, ecrPublicInvalid("tags is required")
		}
		merged := len(current)
		for k := range tags {
			if _, ok := current[k]; !ok {
				merged++
			}
		}
		if merged > ecrPublicMaxTags {
			return nil, ecrPublicFail("TooManyTagsException", fmt.Sprintf("a repository can have at most %d tags", ecrPublicMaxTags))
		}
		if e := st.engine.TagResource(engineARN, tags); e != nil {
			return nil, ecrPublicEngineErr(e)
		}
		return map[string]any{}, nil
	case "UntagResource":
		if len(in.TagKeys) == 0 || len(in.TagKeys) > 200 {
			return nil, ecrPublicInvalid("tagKeys must have 1-200 entries")
		}
		if e := st.engine.UntagResource(engineARN, in.TagKeys); e != nil {
			return nil, ecrPublicEngineErr(e)
		}
		return map[string]any{}, nil
	}
	keys := make([]string, 0, len(current))
	for k := range current {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]map[string]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, map[string]string{"Key": k, "Value": current[k]})
	}
	return map[string]any{"tags": out}, nil
}
