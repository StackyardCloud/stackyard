package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type rekognitionStore struct {
	mu sync.Mutex

	nextID int64

	collections      map[string]*rekognitionCollection
	users            map[string]map[string]*rekognitionUser
	projects         map[string]*rekognitionProject
	datasets         map[string]*rekognitionDataset
	streamProcessors map[string]*rekognitionStreamProcessor
	jobs             map[string]*rekognitionJob
	projectPolicies  map[string][]string
	tags             map[string]map[string]string
	livenessSessions map[string]string
}

type rekognitionCollection struct {
	ID        string
	ARN       string
	CreatedAt string
	FaceIDs   map[string]struct{}
}

type rekognitionUser struct {
	UserID    string
	Status    string
	CreatedAt string
	UpdatedAt string
	FaceIDs   map[string]struct{}
}

type rekognitionProject struct {
	Name      string
	ARN       string
	Status    string
	CreatedAt string
}

type rekognitionDataset struct {
	ARN        string
	ProjectARN string
	Type       string
	Status     string
	CreatedAt  string
}

type rekognitionStreamProcessor struct {
	Name      string
	ARN       string
	Status    string
	CreatedAt string
}

type rekognitionJob struct {
	ID        string
	Kind      string
	Status    string
	CreatedAt string
}

func newRekognitionStore() *rekognitionStore {
	now := time.Now().UTC().Format(time.RFC3339)
	collectionID := "stackyard-collection"
	collection := &rekognitionCollection{
		ID:        collectionID,
		ARN:       rekognitionCollectionARN(collectionID),
		CreatedAt: now,
		FaceIDs: map[string]struct{}{
			"face-000001": {},
		},
	}
	project := &rekognitionProject{
		Name:      "stackyard-project",
		ARN:       rekognitionProjectARN("stackyard-project"),
		Status:    "CREATED",
		CreatedAt: now,
	}
	dataset := &rekognitionDataset{
		ARN:        rekognitionDatasetARN("stackyard-project", "train"),
		ProjectARN: project.ARN,
		Type:       "TRAIN",
		Status:     "CREATE_COMPLETE",
		CreatedAt:  now,
	}
	processor := &rekognitionStreamProcessor{
		Name:      "stackyard-stream-processor",
		ARN:       rekognitionStreamProcessorARN("stackyard-stream-processor"),
		Status:    "STOPPED",
		CreatedAt: now,
	}

	return &rekognitionStore{
		nextID: 2,
		collections: map[string]*rekognitionCollection{
			collectionID: collection,
		},
		users: map[string]map[string]*rekognitionUser{
			collectionID: {
				"user-000001": {
					UserID:    "user-000001",
					Status:    "ACTIVE",
					CreatedAt: now,
					UpdatedAt: now,
					FaceIDs: map[string]struct{}{
						"face-000001": {},
					},
				},
			},
		},
		projects: map[string]*rekognitionProject{
			project.ARN: project,
		},
		datasets: map[string]*rekognitionDataset{
			dataset.ARN: dataset,
		},
		streamProcessors: map[string]*rekognitionStreamProcessor{
			processor.Name: processor,
		},
		jobs: map[string]*rekognitionJob{},
		projectPolicies: map[string][]string{
			project.ARN: {},
		},
		tags: map[string]map[string]string{
			collection.ARN: {"stackyard": "true"},
			project.ARN:    {"stackyard": "true"},
			dataset.ARN:    {"stackyard": "true"},
			processor.ARN:  {"stackyard": "true"},
		},
		livenessSessions: map[string]string{},
	}
}

func (s *rekognitionStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	collectionID := rekognitionPayloadString(payload, "CollectionId", "stackyard-collection")
	userID := rekognitionPayloadString(payload, "UserId", "user-000001")
	projectARN := rekognitionPayloadString(payload, "ProjectArn", rekognitionProjectARN("stackyard-project"))
	projectVersionARN := rekognitionPayloadString(payload, "ProjectVersionArn", rekognitionProjectVersionARN("stackyard-project", "stackyard-version"))
	datasetARN := rekognitionPayloadString(payload, "DatasetArn", rekognitionDatasetARN("stackyard-project", "train"))
	streamProcessorName := rekognitionPayloadString(payload, "Name", "stackyard-stream-processor")
	streamProcessorARN := rekognitionPayloadString(payload, "StreamProcessorArn", rekognitionStreamProcessorARN(streamProcessorName))
	resourceARN := rekognitionPayloadString(payload, "ResourceArn", rekognitionCollectionARN(collectionID))
	jobID := rekognitionPayloadString(payload, "JobId", "job-000001")

	collection := s.ensureCollectionLocked(collectionID)
	_ = s.ensureUserLocked(collectionID, userID)
	project := s.ensureProjectLocked(projectARN)
	dataset := s.ensureDatasetLocked(datasetARN, project.ARN)
	streamProcessor := s.ensureStreamProcessorLocked(streamProcessorName)
	if streamProcessorARN != "" {
		streamProcessor.ARN = streamProcessorARN
	}

	switch action {
	case "CreateCollection":
		collection = s.ensureCollectionLocked(collectionID)
		return map[string]any{"StatusCode": 200, "CollectionArn": collection.ARN}

	case "DeleteCollection":
		delete(s.collections, collectionID)
		delete(s.users, collectionID)
		return map[string]any{"StatusCode": 200}

	case "DescribeCollection":
		return map[string]any{
			"CollectionARN":     collection.ARN,
			"FaceCount":         len(collection.FaceIDs),
			"FaceModelVersion":  "7.0",
			"CreationTimestamp": collection.CreatedAt,
		}

	case "ListCollections":
		ids := make([]string, 0, len(s.collections))
		for id := range s.collections {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		items := make([]any, 0, len(ids))
		for _, id := range ids {
			items = append(items, id)
		}
		return map[string]any{"CollectionIds": items, "NextToken": ""}

	case "IndexFaces":
		faceID := s.nextTokenLocked("face")
		collection.FaceIDs[faceID] = struct{}{}
		return map[string]any{
			"FaceRecords": []any{
				map[string]any{
					"Face": map[string]any{
						"FaceId":          faceID,
						"ImageId":         s.nextTokenLocked("img"),
						"Confidence":      99.0,
						"BoundingBox":     map[string]any{"Width": 0.2, "Height": 0.2, "Left": 0.1, "Top": 0.1},
						"ExternalImageId": rekognitionPayloadString(payload, "ExternalImageId", "stackyard-image"),
					},
				},
			},
			"UnindexedFaces":        []any{},
			"FaceModelVersion":      "7.0",
			"OrientationCorrection": "ROTATE_0",
		}

	case "DeleteFaces":
		for _, faceID := range rekognitionPayloadStringSlice(payload, "FaceIds") {
			delete(collection.FaceIDs, faceID)
		}
		return map[string]any{"DeletedFaces": rekognitionPayloadStringSlice(payload, "FaceIds"), "UnsuccessfulFaceDeletions": []any{}}

	case "ListFaces":
		faceIDs := make([]string, 0, len(collection.FaceIDs))
		for faceID := range collection.FaceIDs {
			faceIDs = append(faceIDs, faceID)
		}
		sort.Strings(faceIDs)
		faces := make([]any, 0, len(faceIDs))
		for _, faceID := range faceIDs {
			faces = append(faces, map[string]any{
				"FaceId":      faceID,
				"Confidence":  99.0,
				"ImageId":     "img-000001",
				"BoundingBox": map[string]any{"Width": 0.2, "Height": 0.2, "Left": 0.1, "Top": 0.1},
			})
		}
		return map[string]any{"Faces": faces, "FaceModelVersion": "7.0", "NextToken": ""}

	case "SearchFaces", "SearchFacesByImage":
		return map[string]any{
			"SearchedFaceId": "face-000001",
			"FaceMatches": []any{
				map[string]any{
					"Similarity": 99.0,
					"Face": map[string]any{
						"FaceId":      "face-000001",
						"Confidence":  99.0,
						"BoundingBox": map[string]any{"Width": 0.2, "Height": 0.2, "Left": 0.1, "Top": 0.1},
					},
				},
			},
			"FaceModelVersion": "7.0",
		}

	case "CreateUser":
		u := s.ensureUserLocked(collectionID, userID)
		u.Status = "ACTIVE"
		u.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"UserStatus": u.Status}

	case "DeleteUser":
		if users, ok := s.users[collectionID]; ok {
			delete(users, userID)
		}
		return map[string]any{}

	case "ListUsers":
		items := []any{}
		if users, ok := s.users[collectionID]; ok {
			keys := make([]string, 0, len(users))
			for id := range users {
				keys = append(keys, id)
			}
			sort.Strings(keys)
			for _, id := range keys {
				u := users[id]
				items = append(items, map[string]any{"UserId": u.UserID, "UserStatus": u.Status})
			}
		}
		return map[string]any{"Users": items, "NextToken": ""}

	case "SearchUsers", "SearchUsersByImage":
		return map[string]any{"UserMatches": []any{map[string]any{"Similarity": 99.0, "User": map[string]any{"UserId": userID, "UserStatus": "ACTIVE"}}}}

	case "AssociateFaces":
		u := s.ensureUserLocked(collectionID, userID)
		associated := []any{}
		for _, faceID := range rekognitionPayloadStringSlice(payload, "FaceIds") {
			u.FaceIDs[faceID] = struct{}{}
			associated = append(associated, map[string]any{"FaceId": faceID})
		}
		return map[string]any{"AssociatedFaces": associated, "UnsuccessfulFaceAssociations": []any{}, "UserStatus": u.Status}

	case "DisassociateFaces":
		u := s.ensureUserLocked(collectionID, userID)
		disassociated := []any{}
		for _, faceID := range rekognitionPayloadStringSlice(payload, "FaceIds") {
			delete(u.FaceIDs, faceID)
			disassociated = append(disassociated, map[string]any{"FaceId": faceID})
		}
		return map[string]any{"DisassociatedFaces": disassociated, "UnsuccessfulFaceDisassociations": []any{}, "UserStatus": u.Status}

	case "CreateProject":
		project = s.ensureProjectLocked(projectARN)
		return map[string]any{"ProjectArn": project.ARN}

	case "DeleteProject":
		delete(s.projects, projectARN)
		return map[string]any{"Status": "DELETING"}

	case "DescribeProjects":
		items := make([]any, 0, len(s.projects))
		keys := make([]string, 0, len(s.projects))
		for arn := range s.projects {
			keys = append(keys, arn)
		}
		sort.Strings(keys)
		for _, arn := range keys {
			p := s.projects[arn]
			items = append(items, map[string]any{"ProjectArn": p.ARN, "Status": p.Status, "CreationTimestamp": p.CreatedAt})
		}
		return map[string]any{"ProjectDescriptions": items, "NextToken": ""}

	case "CreateProjectVersion":
		return map[string]any{"ProjectVersionArn": projectVersionARN, "Status": "TRAINING_IN_PROGRESS"}

	case "CopyProjectVersion":
		return map[string]any{"ProjectVersionArn": projectVersionARN, "Status": "COPYING_IN_PROGRESS"}

	case "DescribeProjectVersions":
		return map[string]any{
			"ProjectVersionDescriptions": []any{map[string]any{
				"ProjectVersionArn": projectVersionARN,
				"Status":            "RUNNING",
				"CreationTimestamp": project.CreatedAt,
			}},
			"NextToken": "",
		}

	case "StartProjectVersion":
		return map[string]any{"Status": "STARTING"}

	case "StopProjectVersion":
		return map[string]any{"Status": "STOPPING"}

	case "DeleteProjectVersion":
		return map[string]any{"Status": "DELETING"}

	case "CreateDataset":
		dataset = s.ensureDatasetLocked(datasetARN, projectARN)
		return map[string]any{"DatasetArn": dataset.ARN}

	case "DeleteDataset":
		delete(s.datasets, datasetARN)
		return map[string]any{"Status": "DELETE_IN_PROGRESS"}

	case "DescribeDataset":
		return map[string]any{
			"DatasetDescription": map[string]any{
				"DatasetArn":        dataset.ARN,
				"DatasetType":       dataset.Type,
				"Status":            dataset.Status,
				"CreationTimestamp": dataset.CreatedAt,
			},
		}

	case "ListDatasetEntries":
		return map[string]any{"DatasetEntries": []any{"{\"source-ref\":\"s3://stackyard-rekognition/image-000001.jpg\"}"}, "NextToken": ""}

	case "ListDatasetLabels":
		return map[string]any{"DatasetLabelDescriptions": []any{map[string]any{"LabelName": "Person", "LabelStats": map[string]any{"EntryCount": 1}}}, "NextToken": ""}

	case "DistributeDatasetEntries", "UpdateDatasetEntries":
		return map[string]any{"Status": "UPDATE_IN_PROGRESS"}

	case "CreateStreamProcessor":
		streamProcessor = s.ensureStreamProcessorLocked(streamProcessorName)
		return map[string]any{"StreamProcessorArn": streamProcessor.ARN}

	case "DeleteStreamProcessor":
		delete(s.streamProcessors, streamProcessorName)
		return map[string]any{}

	case "DescribeStreamProcessor":
		return map[string]any{"Name": streamProcessor.Name, "StreamProcessorArn": streamProcessor.ARN, "Status": streamProcessor.Status, "CreationTimestamp": streamProcessor.CreatedAt}

	case "ListStreamProcessors":
		names := make([]string, 0, len(s.streamProcessors))
		for name := range s.streamProcessors {
			names = append(names, name)
		}
		sort.Strings(names)
		items := make([]any, 0, len(names))
		for _, name := range names {
			sp := s.streamProcessors[name]
			items = append(items, map[string]any{"Name": sp.Name, "StreamProcessorArn": sp.ARN, "Status": sp.Status})
		}
		return map[string]any{"StreamProcessors": items, "NextToken": ""}

	case "StartStreamProcessor":
		streamProcessor.Status = "RUNNING"
		return map[string]any{}

	case "StopStreamProcessor":
		streamProcessor.Status = "STOPPED"
		return map[string]any{}

	case "UpdateStreamProcessor":
		return map[string]any{}

	case "CreateFaceLivenessSession":
		sessionID := s.nextTokenLocked("liveness")
		s.livenessSessions[sessionID] = "IN_PROGRESS"
		return map[string]any{"SessionId": sessionID}

	case "GetFaceLivenessSessionResults":
		sessionID := rekognitionPayloadString(payload, "SessionId", "liveness-000001")
		status := s.livenessSessions[sessionID]
		if status == "" {
			status = "SUCCEEDED"
		}
		s.livenessSessions[sessionID] = "SUCCEEDED"
		return map[string]any{"SessionId": sessionID, "Status": status, "Confidence": 99.0, "ReferenceImage": map[string]any{}}

	case "StartLabelDetection", "StartFaceDetection", "StartFaceSearch", "StartCelebrityRecognition", "StartContentModeration", "StartPersonTracking", "StartSegmentDetection", "StartTextDetection", "StartMediaAnalysisJob":
		job := s.ensureJobLocked(jobID, action)
		return map[string]any{"JobId": job.ID}

	case "GetLabelDetection", "GetFaceDetection", "GetFaceSearch", "GetCelebrityRecognition", "GetContentModeration", "GetPersonTracking", "GetSegmentDetection", "GetTextDetection", "GetMediaAnalysisJob":
		job := s.ensureJobLocked(jobID, action)
		payload := map[string]any{"JobStatus": job.Status, "StatusMessage": "ok", "VideoMetadata": map[string]any{"FrameRate": 30.0, "Codec": "h264", "DurationMillis": 1000, "Format": "mp4"}, "NextToken": ""}
		switch action {
		case "GetLabelDetection":
			payload["Labels"] = []any{map[string]any{"Timestamp": 0, "Label": map[string]any{"Name": "Person", "Confidence": 99.0}}}
		case "GetFaceDetection":
			payload["Faces"] = []any{map[string]any{"Timestamp": 0, "Face": map[string]any{"Confidence": 99.0}}}
		case "GetFaceSearch":
			payload["Persons"] = []any{map[string]any{"Timestamp": 0, "Person": map[string]any{"Index": 0}}}
		case "GetCelebrityRecognition":
			payload["Celebrities"] = []any{map[string]any{"Timestamp": 0, "Celebrity": map[string]any{"Name": "Stack Yard"}}}
		case "GetContentModeration":
			payload["ModerationLabels"] = []any{map[string]any{"Timestamp": 0, "ModerationLabel": map[string]any{"Name": "Safe"}}}
		case "GetPersonTracking":
			payload["Persons"] = []any{map[string]any{"Timestamp": 0, "Person": map[string]any{"Index": 0}}}
		case "GetSegmentDetection":
			payload["Segments"] = []any{map[string]any{"Type": "SHOT", "StartTimestampMillis": 0, "EndTimestampMillis": 1000}}
		case "GetTextDetection":
			payload["TextDetections"] = []any{map[string]any{"Timestamp": 0, "TextDetection": map[string]any{"DetectedText": "stackyard"}}}
		case "GetMediaAnalysisJob":
			payload = map[string]any{"JobId": job.ID, "JobStatus": job.Status, "StatusMessage": "ok", "CreationTimestamp": job.CreatedAt}
		}
		return payload

	case "ListMediaAnalysisJobs":
		items := []any{}
		keys := make([]string, 0, len(s.jobs))
		for id := range s.jobs {
			keys = append(keys, id)
		}
		sort.Strings(keys)
		for _, id := range keys {
			job := s.jobs[id]
			if strings.Contains(job.Kind, "MediaAnalysis") {
				items = append(items, map[string]any{"JobId": job.ID, "Status": job.Status, "CreationTimestamp": job.CreatedAt})
			}
		}
		return map[string]any{"MediaAnalysisJobs": items, "NextToken": ""}

	case "DetectLabels":
		return map[string]any{"Labels": []any{map[string]any{"Name": "Person", "Confidence": 99.0}}, "OrientationCorrection": "ROTATE_0", "LabelModelVersion": "3.0"}
	case "DetectFaces":
		return map[string]any{"FaceDetails": []any{map[string]any{"Confidence": 99.0, "BoundingBox": map[string]any{"Width": 0.2, "Height": 0.2, "Left": 0.1, "Top": 0.1}}}, "OrientationCorrection": "ROTATE_0"}
	case "DetectModerationLabels":
		return map[string]any{"ModerationLabels": []any{map[string]any{"Name": "Safe", "Confidence": 99.0}}, "ModerationModelVersion": "7.0"}
	case "DetectText":
		return map[string]any{"TextDetections": []any{map[string]any{"DetectedText": "stackyard", "Confidence": 99.0}}, "TextModelVersion": "3.0"}
	case "DetectCustomLabels":
		return map[string]any{"CustomLabels": []any{map[string]any{"Name": "CustomObject", "Confidence": 99.0}}, "ProjectVersion": projectVersionARN}
	case "DetectProtectiveEquipment":
		return map[string]any{"Persons": []any{map[string]any{"Id": 0, "Confidence": 99.0}}, "Summary": map[string]any{}}
	case "RecognizeCelebrities":
		return map[string]any{"CelebrityFaces": []any{map[string]any{"Name": "Stack Yard", "MatchConfidence": 99.0}}, "UnrecognizedFaces": []any{}, "OrientationCorrection": "ROTATE_0"}
	case "CompareFaces":
		return map[string]any{"SourceImageFace": map[string]any{"Confidence": 99.0}, "FaceMatches": []any{map[string]any{"Similarity": 99.0}}, "UnmatchedFaces": []any{}, "SourceImageOrientationCorrection": "ROTATE_0", "TargetImageOrientationCorrection": "ROTATE_0"}
	case "GetCelebrityInfo":
		id := rekognitionPayloadString(payload, "Id", "celeb-000001")
		return map[string]any{"Name": "Stack Yard", "Urls": []any{"https://example.com/celebrity/" + id}}

	case "PutProjectPolicy":
		name := rekognitionPayloadString(payload, "PolicyName", "default")
		policies := s.projectPolicies[projectARN]
		if !rekognitionContainsString(policies, name) {
			s.projectPolicies[projectARN] = append(policies, name)
		}
		return map[string]any{}

	case "DeleteProjectPolicy":
		name := rekognitionPayloadString(payload, "PolicyName", "default")
		policies := s.projectPolicies[projectARN]
		filtered := make([]string, 0, len(policies))
		for _, p := range policies {
			if p != name {
				filtered = append(filtered, p)
			}
		}
		s.projectPolicies[projectARN] = filtered
		return map[string]any{}

	case "ListProjectPolicies":
		items := make([]any, 0, len(s.projectPolicies[projectARN]))
		for _, name := range s.projectPolicies[projectARN] {
			items = append(items, name)
		}
		return map[string]any{"ProjectPolicies": items, "NextToken": ""}

	case "TagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for key, value := range rekognitionPayloadTags(payload, "Tags") {
			tags[key] = value
		}
		return map[string]any{}

	case "UntagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range rekognitionPayloadStringSlice(payload, "TagKeys") {
			delete(tags, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		out := map[string]any{}
		for key, value := range s.ensureTagsLocked(resourceARN) {
			out[key] = value
		}
		return map[string]any{"Tags": out}
	}

	return map[string]any{}
}

func (s *rekognitionStore) ensureCollectionLocked(collectionID string) *rekognitionCollection {
	if strings.TrimSpace(collectionID) == "" {
		collectionID = "stackyard-collection"
	}
	if c, ok := s.collections[collectionID]; ok {
		return c
	}
	c := &rekognitionCollection{
		ID:        collectionID,
		ARN:       rekognitionCollectionARN(collectionID),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		FaceIDs:   map[string]struct{}{},
	}
	s.collections[collectionID] = c
	if _, ok := s.users[collectionID]; !ok {
		s.users[collectionID] = map[string]*rekognitionUser{}
	}
	return c
}

func (s *rekognitionStore) ensureUserLocked(collectionID, userID string) *rekognitionUser {
	if strings.TrimSpace(userID) == "" {
		userID = "user-000001"
	}
	if _, ok := s.users[collectionID]; !ok {
		s.users[collectionID] = map[string]*rekognitionUser{}
	}
	if u, ok := s.users[collectionID][userID]; ok {
		return u
	}
	now := time.Now().UTC().Format(time.RFC3339)
	u := &rekognitionUser{UserID: userID, Status: "ACTIVE", CreatedAt: now, UpdatedAt: now, FaceIDs: map[string]struct{}{}}
	s.users[collectionID][userID] = u
	return u
}

func (s *rekognitionStore) ensureProjectLocked(projectARN string) *rekognitionProject {
	if strings.TrimSpace(projectARN) == "" {
		projectARN = rekognitionProjectARN("stackyard-project")
	}
	if p, ok := s.projects[projectARN]; ok {
		return p
	}
	name := projectARN
	if parts := strings.Split(projectARN, "/"); len(parts) > 0 {
		name = parts[len(parts)-1]
	}
	p := &rekognitionProject{Name: name, ARN: projectARN, Status: "CREATED", CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	s.projects[projectARN] = p
	return p
}

func (s *rekognitionStore) ensureDatasetLocked(datasetARN, projectARN string) *rekognitionDataset {
	if strings.TrimSpace(datasetARN) == "" {
		datasetARN = rekognitionDatasetARN("stackyard-project", "train")
	}
	if d, ok := s.datasets[datasetARN]; ok {
		return d
	}
	typ := "TRAIN"
	if strings.Contains(strings.ToLower(datasetARN), "/test") {
		typ = "TEST"
	}
	d := &rekognitionDataset{ARN: datasetARN, ProjectARN: projectARN, Type: typ, Status: "CREATE_COMPLETE", CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	s.datasets[datasetARN] = d
	return d
}

func (s *rekognitionStore) ensureStreamProcessorLocked(name string) *rekognitionStreamProcessor {
	if strings.TrimSpace(name) == "" {
		name = "stackyard-stream-processor"
	}
	if sp, ok := s.streamProcessors[name]; ok {
		return sp
	}
	sp := &rekognitionStreamProcessor{Name: name, ARN: rekognitionStreamProcessorARN(name), Status: "STOPPED", CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	s.streamProcessors[name] = sp
	return sp
}

func (s *rekognitionStore) ensureJobLocked(jobID, kind string) *rekognitionJob {
	if strings.TrimSpace(jobID) == "" {
		jobID = s.nextTokenLocked("job")
	}
	if j, ok := s.jobs[jobID]; ok {
		return j
	}
	j := &rekognitionJob{ID: jobID, Kind: kind, Status: "SUCCEEDED", CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	s.jobs[jobID] = j
	return j
}

func (s *rekognitionStore) ensureTagsLocked(resourceARN string) map[string]string {
	if strings.TrimSpace(resourceARN) == "" {
		resourceARN = rekognitionCollectionARN("stackyard-collection")
	}
	if tags, ok := s.tags[resourceARN]; ok {
		return tags
	}
	tags := map[string]string{"stackyard": "true"}
	s.tags[resourceARN] = tags
	return tags
}

func (s *rekognitionStore) nextTokenLocked(prefix string) string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%s-%06d", prefix, id)
}

func rekognitionCollectionARN(collectionID string) string {
	return "arn:aws:rekognition:us-east-1:123456789012:collection/" + strings.TrimSpace(collectionID)
}

func rekognitionProjectARN(projectName string) string {
	return "arn:aws:rekognition:us-east-1:123456789012:project/" + strings.TrimSpace(projectName)
}

func rekognitionProjectVersionARN(projectName, versionName string) string {
	if strings.TrimSpace(projectName) == "" {
		projectName = "stackyard-project"
	}
	if strings.TrimSpace(versionName) == "" {
		versionName = "stackyard-version"
	}
	return "arn:aws:rekognition:us-east-1:123456789012:project/" + strings.TrimSpace(projectName) + "/version/" + strings.TrimSpace(versionName) + "/1700000000000"
}

func rekognitionDatasetARN(projectName, datasetType string) string {
	if strings.TrimSpace(projectName) == "" {
		projectName = "stackyard-project"
	}
	datasetType = strings.ToLower(strings.TrimSpace(datasetType))
	if datasetType == "" {
		datasetType = "train"
	}
	return "arn:aws:rekognition:us-east-1:123456789012:project/" + strings.TrimSpace(projectName) + "/dataset/" + datasetType
}

func rekognitionStreamProcessorARN(name string) string {
	if strings.TrimSpace(name) == "" {
		name = "stackyard-stream-processor"
	}
	return "arn:aws:rekognition:us-east-1:123456789012:streamprocessor/" + strings.TrimSpace(name)
}

func rekognitionPayloadValue(payload map[string]any, key string) (any, bool) {
	if payload == nil {
		return nil, false
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return nil, false
}

func rekognitionPayloadString(payload map[string]any, key, fallback string) string {
	value, ok := rekognitionPayloadValue(payload, key)
	if !ok || value == nil {
		return fallback
	}
	out := strings.TrimSpace(fmt.Sprintf("%v", value))
	if out == "" || out == "<nil>" {
		return fallback
	}
	return out
}

func rekognitionPayloadStringSlice(payload map[string]any, key string) []string {
	value, ok := rekognitionPayloadValue(payload, key)
	if !ok || value == nil {
		return nil
	}
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		v := strings.TrimSpace(fmt.Sprintf("%v", item))
		if v != "" && v != "<nil>" {
			out = append(out, v)
		}
	}
	return out
}

func rekognitionPayloadTags(payload map[string]any, key string) map[string]string {
	value, ok := rekognitionPayloadValue(payload, key)
	if !ok || value == nil {
		return map[string]string{}
	}
	out := map[string]string{}
	switch typed := value.(type) {
	case map[string]any:
		for k, v := range typed {
			out[strings.TrimSpace(k)] = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	case map[string]string:
		for k, v := range typed {
			out[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return out
}

func rekognitionContainsString(items []string, needle string) bool {
	for _, item := range items {
		if item == needle {
			return true
		}
	}
	return false
}
