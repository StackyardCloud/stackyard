package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	ecrsvc "github.com/stackyard/stackyard/internal/services/ecr"
)

func (st *ecrPublicStore) initiateLayerUpload(in ecrPublicInput) (any, *ecrPublicError) {
	repo, err := st.repository(in.RepositoryName)
	if err != nil {
		return nil, err
	}
	if len(st.uploads) >= ecrPublicMaxUploads {
		return nil, ecrPublicFail("LimitExceededException", fmt.Sprintf("at most %d layer uploads can be in progress", ecrPublicMaxUploads))
	}
	engineID, _, ierr := st.engine.InitiateLayerUpload(repo.RepositoryName)
	if ierr != nil {
		return nil, ecrPublicEngineErr(ierr)
	}
	id := uuid.NewString()
	st.uploads[id] = &ecrPublicUpload{repo: repo.RepositoryName, engineID: engineID}
	return map[string]any{"uploadId": id, "partSize": ecrPublicPartSize}, nil
}

// upload returns an in-progress upload belonging to the named repository.
func (st *ecrPublicStore) upload(in ecrPublicInput) (ecrsvc.Repository, *ecrPublicUpload, *ecrPublicError) {
	repo, err := st.repository(in.RepositoryName)
	if err != nil {
		return repo, nil, err
	}
	up := st.uploads[in.UploadID]
	if up == nil || up.repo != repo.RepositoryName {
		return repo, nil, ecrPublicFail("UploadNotFoundException", "upload "+in.UploadID+" does not exist")
	}
	return repo, up, nil
}

func (st *ecrPublicStore) uploadLayerPart(in ecrPublicInput) (any, *ecrPublicError) {
	if in.PartFirstByte == nil || in.PartLastByte == nil || in.LayerPartBlob == nil {
		return nil, ecrPublicInvalid("partFirstByte, partLastByte and layerPartBlob are required")
	}
	repo, up, err := st.upload(in)
	if err != nil {
		return nil, err
	}
	first, last := *in.PartFirstByte, *in.PartLastByte
	size := int64(len(in.LayerPartBlob))
	if first != up.received || last < first || last-first+1 != size {
		return nil, ecrPublicFail("InvalidLayerPartException", fmt.Sprintf("the part must cover bytes %d-%d of the upload, matching layerPartBlob's length", up.received, up.received+size-1))
	}
	if last+1 > ecrPublicMaxLayer {
		return nil, ecrPublicFail("LimitExceededException", fmt.Sprintf("a layer can be at most %d bytes", ecrPublicMaxLayer))
	}
	if st.storedBytes+size > ecrPublicMaxStored {
		return nil, ecrPublicFail("LimitExceededException", "the registry has reached its storage limit")
	}
	if _, perr := st.engine.UploadLayerPart(repo.RepositoryName, up.engineID, first, last, in.LayerPartBlob); perr != nil {
		return nil, ecrPublicEngineErr(perr)
	}
	up.received = last + 1
	st.storedBytes += size
	return map[string]any{"registryId": ecrPublicAccountID, "repositoryName": repo.RepositoryName, "uploadId": in.UploadID, "lastByteReceived": last}, nil
}

func (st *ecrPublicStore) completeLayerUpload(in ecrPublicInput) (any, *ecrPublicError) {
	if len(in.LayerDigests) == 0 || len(in.LayerDigests) > 100 {
		return nil, ecrPublicInvalid("layerDigests must have 1-100 entries")
	}
	repo, up, err := st.upload(in)
	if err != nil {
		return nil, err
	}
	if up.received == 0 {
		return nil, ecrPublicFail("EmptyUploadException", "the upload has no layer parts")
	}
	if layers, _, cerr := st.engine.BatchCheckLayerAvailability(repo.RepositoryName, in.LayerDigests); cerr == nil {
		for _, l := range layers {
			if l.LayerAvailability == "AVAILABLE" {
				return nil, ecrPublicFail("LayerAlreadyExistsException", "layer "+l.LayerDigest+" already exists in the repository")
			}
		}
	}
	digest, cerr := st.engine.CompleteLayerUpload(repo.RepositoryName, up.engineID, in.LayerDigests)
	if cerr != nil {
		return nil, ecrPublicFail("InvalidLayerException", "the uploaded bytes don't match any of layerDigests")
	}
	st.repoBytes[repo.RepositoryName] += up.received
	delete(st.uploads, in.UploadID)
	return map[string]any{"registryId": ecrPublicAccountID, "repositoryName": repo.RepositoryName, "uploadId": in.UploadID, "layerDigest": digest}, nil
}

func (st *ecrPublicStore) batchCheckLayerAvailability(in ecrPublicInput) (any, *ecrPublicError) {
	if len(in.LayerDigests) == 0 || len(in.LayerDigests) > 100 {
		return nil, ecrPublicInvalid("layerDigests must have 1-100 entries")
	}
	repo, err := st.repository(in.RepositoryName)
	if err != nil {
		return nil, err
	}
	layers, failures := []map[string]any{}, []map[string]any{}
	var valid []string
	for _, d := range in.LayerDigests {
		if d == "" {
			failures = append(failures, map[string]any{"layerDigest": d, "failureCode": "MissingLayerDigest", "failureReason": "layer digest is required"})
		} else if !ecrPublicDigestPattern.MatchString(d) {
			failures = append(failures, map[string]any{"layerDigest": d, "failureCode": "InvalidLayerDigest", "failureReason": "layer digest is malformed"})
		} else {
			valid = append(valid, d)
		}
	}
	if len(valid) > 0 {
		found, missing, cerr := st.engine.BatchCheckLayerAvailability(repo.RepositoryName, valid)
		if cerr != nil {
			return nil, ecrPublicEngineErr(cerr)
		}
		for _, l := range found {
			layers = append(layers, map[string]any{"layerDigest": l.LayerDigest, "layerAvailability": l.LayerAvailability, "layerSize": l.LayerSize, "mediaType": l.MediaType})
		}
		for _, f := range missing {
			// The model's LayerFailureCode has no "not found"; AWS reports a
			// missing layer as an UNAVAILABLE layer.
			layers = append(layers, map[string]any{"layerDigest": f.LayerDigest, "layerAvailability": "UNAVAILABLE"})
		}
	}
	return map[string]any{"layers": layers, "failures": failures}, nil
}

// ecrPublicManifest is the part of an image manifest or manifest list that
// PutImage checks.
type ecrPublicManifest struct {
	SchemaVersion int `json:"schemaVersion"`
	Config        *struct {
		Digest string `json:"digest"`
		Size   int64  `json:"size"`
	} `json:"config"`
	Layers []struct {
		Digest string `json:"digest"`
		Size   int64  `json:"size"`
	} `json:"layers"`
	Manifests []struct {
		Digest string `json:"digest"`
	} `json:"manifests"`
}

func (st *ecrPublicStore) putImage(in ecrPublicInput) (any, *ecrPublicError) {
	manifest := strings.TrimSpace(in.ImageManifest)
	if manifest == "" || len(in.ImageManifest) > ecrPublicMaxManifest {
		return nil, ecrPublicInvalid("imageManifest is required and can be at most 4 MiB")
	}
	if len(in.ImageTag) > 300 {
		return nil, ecrPublicInvalid("imageTag can be at most 300 characters")
	}
	repo, err := st.repository(in.RepositoryName)
	if err != nil {
		return nil, err
	}
	var m ecrPublicManifest
	if json.Unmarshal([]byte(manifest), &m) != nil || m.SchemaVersion == 0 {
		return nil, ecrPublicInvalid("imageManifest must be a JSON image manifest with a schemaVersion")
	}
	if len(m.Layers)+len(m.Manifests) > 1000 {
		return nil, ecrPublicInvalid("imageManifest references too many layers or images")
	}
	var size int64
	var blobs []string
	if m.Config != nil && m.Config.Digest != "" {
		blobs = append(blobs, m.Config.Digest)
		size += m.Config.Size
	}
	for _, l := range m.Layers {
		blobs = append(blobs, l.Digest)
		size += l.Size
	}
	if missing := st.missingLayers(repo.RepositoryName, blobs); len(missing) > 0 {
		return nil, ecrPublicFail("LayersNotFoundException", "layers not found: "+strings.Join(missing, ", "))
	}
	for _, ref := range m.Manifests {
		if _, _, ierr := st.engine.DescribeImages(repo.RepositoryName, []ecrsvc.ImageIdentifier{{ImageDigest: ref.Digest}}, "", "", 0); ierr != nil {
			return nil, ecrPublicFail("ReferencedImagesNotFoundException", "referenced image "+ref.Digest+" does not exist")
		}
	}
	sum := sha256.Sum256([]byte(manifest))
	digest := "sha256:" + hex.EncodeToString(sum[:])
	if existing, _, ierr := st.engine.DescribeImages(repo.RepositoryName, []ecrsvc.ImageIdentifier{{ImageDigest: digest}}, "", "", 0); ierr == nil && len(existing) > 0 {
		tagged := in.ImageTag == ""
		for _, t := range existing[0].ImageTags {
			tagged = tagged || t == in.ImageTag
		}
		if tagged {
			return nil, ecrPublicFail("ImageAlreadyExistsException", "an image with this manifest and tag already exists")
		}
	}
	key := repo.RepositoryName + "@" + digest
	_, known := st.images[key]
	if !known && st.storedBytes+int64(len(manifest)) > ecrPublicMaxStored {
		return nil, ecrPublicFail("LimitExceededException", "the registry has reached its storage limit")
	}
	image, perr := st.engine.PutImage(repo.RepositoryName, in.ImageManifest, in.ImageManifestMediaType, in.ImageTag, in.ImageDigest)
	if perr != nil {
		return nil, ecrPublicEngineErr(perr)
	}
	if !known {
		st.storedBytes += int64(len(manifest))
	}
	st.images[key] = ecrPublicImage{layerBytes: size, manifestBytes: int64(len(manifest))}
	return map[string]any{"image": map[string]any{
		"registryId":             ecrPublicAccountID,
		"repositoryName":         repo.RepositoryName,
		"imageId":                ecrPublicImageID{ImageDigest: image.ImageID.ImageDigest, ImageTag: image.ImageID.ImageTag},
		"imageManifest":          image.ImageManifest,
		"imageManifestMediaType": image.ImageManifestMediaType,
	}}, nil
}

func (st *ecrPublicStore) missingLayers(repo string, digests []string) []string {
	if len(digests) == 0 {
		return nil
	}
	found, failures, err := st.engine.BatchCheckLayerAvailability(repo, digests)
	if err != nil {
		return digests
	}
	var missing []string
	for _, f := range failures {
		missing = append(missing, f.LayerDigest)
	}
	for _, l := range found {
		if l.LayerAvailability != "AVAILABLE" {
			missing = append(missing, l.LayerDigest)
		}
	}
	return missing
}

func (st *ecrPublicStore) imageDetailOut(d ecrsvc.ImageDetail) map[string]any {
	size := d.ImageSizeInBytes
	if img, ok := st.images[d.RepositoryName+"@"+d.ImageDigest]; ok {
		size = img.layerBytes
	}
	return map[string]any{
		"registryId":             ecrPublicAccountID,
		"repositoryName":         d.RepositoryName,
		"imageDigest":            d.ImageDigest,
		"imageTags":              append([]string{}, d.ImageTags...),
		"imageSizeInBytes":       size,
		"imagePushedAt":          ecrPublicTime(d.ImagePushedAt),
		"imageManifestMediaType": d.ImageManifestMediaType,
	}
}

func ecrPublicEngineIDs(ids []ecrPublicImageID) []ecrsvc.ImageIdentifier {
	out := make([]ecrsvc.ImageIdentifier, 0, len(ids))
	for _, id := range ids {
		out = append(out, ecrsvc.ImageIdentifier{ImageDigest: id.ImageDigest, ImageTag: id.ImageTag})
	}
	return out
}

func (st *ecrPublicStore) describeImages(in ecrPublicInput) (any, *ecrPublicError) {
	if len(in.ImageIDs) > 100 {
		return nil, ecrPublicInvalid("imageIds can have at most 100 entries")
	}
	repo, err := st.repository(in.RepositoryName)
	if err != nil {
		return nil, err
	}
	var maxResults int32
	if in.MaxResults != nil {
		maxResults = *in.MaxResults
	}
	details, next, derr := st.engine.DescribeImages(repo.RepositoryName, ecrPublicEngineIDs(in.ImageIDs), "ANY", in.NextToken, maxResults)
	if derr != nil {
		return nil, ecrPublicEngineErr(derr)
	}
	out := make([]map[string]any, 0, len(details))
	for _, d := range details {
		out = append(out, st.imageDetailOut(d))
	}
	resp := map[string]any{"imageDetails": out}
	if next != "" {
		resp["nextToken"] = next
	}
	return resp, nil
}

func (st *ecrPublicStore) describeImageTags(in ecrPublicInput) (any, *ecrPublicError) {
	repo, err := st.repository(in.RepositoryName)
	if err != nil {
		return nil, err
	}
	var all []ecrsvc.ImageDetail
	token := ""
	for page := 0; page < ecrPublicMaxPageWalk; page++ {
		details, next, derr := st.engine.DescribeImages(repo.RepositoryName, nil, "TAGGED", token, 1000)
		if derr != nil {
			return nil, ecrPublicEngineErr(derr)
		}
		all = append(all, details...)
		if next == "" {
			break
		}
		token = next
	}
	type tagged struct {
		tag    string
		detail ecrsvc.ImageDetail
	}
	var tags []tagged
	for _, d := range all {
		for _, t := range d.ImageTags {
			tags = append(tags, tagged{t, d})
		}
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].tag < tags[j].tag })
	start, end, perr := ecrPublicPage(in.NextToken, in.MaxResults, len(tags))
	if perr != nil {
		return nil, perr
	}
	out := make([]map[string]any, 0, end-start)
	for _, t := range tags[start:end] {
		d := st.imageDetailOut(t.detail)
		delete(d, "registryId")
		delete(d, "repositoryName")
		delete(d, "imageTags")
		out = append(out, map[string]any{"imageTag": t.tag, "createdAt": d["imagePushedAt"], "imageDetail": d})
	}
	return map[string]any{"imageTagDetails": out, "nextToken": ecrPublicNext(end, len(tags))}, nil
}

func (st *ecrPublicStore) batchDeleteImage(in ecrPublicInput) (any, *ecrPublicError) {
	if len(in.ImageIDs) == 0 || len(in.ImageIDs) > 100 {
		return nil, ecrPublicInvalid("imageIds must have 1-100 entries")
	}
	repo, err := st.repository(in.RepositoryName)
	if err != nil {
		return nil, err
	}
	deleted, failures, derr := st.engine.BatchDeleteImage(repo.RepositoryName, ecrPublicEngineIDs(in.ImageIDs))
	if derr != nil {
		return nil, ecrPublicEngineErr(derr)
	}
	ids := make([]ecrPublicImageID, 0, len(deleted))
	for _, d := range deleted {
		ids = append(ids, ecrPublicImageID{ImageDigest: d.ImageDigest, ImageTag: d.ImageTag})
		key := repo.RepositoryName + "@" + d.ImageDigest
		if img, ok := st.images[key]; ok {
			if _, _, gone := st.engine.DescribeImages(repo.RepositoryName, []ecrsvc.ImageIdentifier{{ImageDigest: d.ImageDigest}}, "", "", 0); gone != nil {
				st.storedBytes -= img.manifestBytes
				delete(st.images, key)
			}
		}
	}
	fails := make([]map[string]any, 0, len(failures))
	for _, f := range failures {
		fails = append(fails, map[string]any{
			"imageId":       ecrPublicImageID{ImageDigest: f.ImageID.ImageDigest, ImageTag: f.ImageID.ImageTag},
			"failureCode":   f.FailureCode,
			"failureReason": f.FailureReason,
		})
	}
	return map[string]any{"imageIds": ids, "failures": fails}, nil
}
