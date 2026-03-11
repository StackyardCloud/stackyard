# Azure Content Moderator (`content-moderator-v1.0`) Staged Plan

## Objective

Emulate Azure Content Moderator REST API (`v1.0`) with deterministic local behavior across all documented operation groups while preserving existing specialized image/text/list-management behavior already implemented in Stackyard.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/operation-groups?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/image-moderation?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-image?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-image-lists?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-term?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-term-lists?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/reviews?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/text-moderation?view=rest-cognitiveservices-contentmoderator-v1.0`

Method pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/image-moderation/evaluate-file-input?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/image-moderation/evaluate-url-input?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/image-moderation/find-faces-file-input?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/image-moderation/find-faces-url-input?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/image-moderation/match-file-input?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/image-moderation/match-url-input?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/image-moderation/ocr-file-input?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/image-moderation/ocr-url-input?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/image-moderation/create-reviews?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/image-moderation/create-video-reviews?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/image-moderation/get-review?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/image-moderation/get-video-review?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-image/add-image?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-image/delete-all-images?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-image/delete-image?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-image/get-all-image-ids?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-image/match-url-input?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-image/refresh-search-index?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-image-lists/create-image-list?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-image-lists/delete-image-list?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-image-lists/get-all-image-lists?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-image-lists/get-image-list-details?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-image-lists/refresh-image-index?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-image-lists/update-image-list?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-term/add-term?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-term/delete-all-terms?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-term/delete-term?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-term/get-all-terms?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-term-lists/create-term-list?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-term-lists/delete-term-list?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-term-lists/get-all-term-lists?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-term-lists/get-term-list-details?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-term-lists/refresh-term-index?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-term-lists/update-term-list?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/reviews/add-video-frame?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/reviews/add-video-frame-url?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/reviews/create-job?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/reviews/get-job-details?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/reviews/get-review-details?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/reviews/get-reviews?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/reviews/publish-review?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/reviews/publish-video-review?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/reviews/transcode-video?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/reviews/get-video-frame?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/reviews/get-video-frame-ids?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/reviews/get-video-frame-url?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/text-moderation/detect-language?view=rest-cognitiveservices-contentmoderator-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/text-moderation/screen-text?view=rest-cognitiveservices-contentmoderator-v1.0`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not regress existing deterministic behavior for:
  - Image moderation
  - Text moderation
  - Image list management
- Do not require live Azure resources for local examples.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/contentmoderator/moderate/v1.0/*`
- `/azure/contentmoderator/lists/v1.0/*`
- `/azure/contentmoderator/review/v1.0/*`

## API Surface and Contract Notes

Operation groups covered:
- Image Moderation
- List Management Image
- List Management Image Lists
- List Management Term
- List Management Term Lists
- Reviews
- Text Moderation

Contract characteristics from method pages:
- Required auth headers include subscription or equivalent cognitive-services authorization.
- Success responses vary across `200`, `201`, and `204`.
- Error responses use typed error envelopes for invalid requests, auth/authz failures, not found, conflict, and service errors.

## Stage 0: Contract Skeleton

- Add an umbrella Azure Content Moderator router for route ownership over uncovered groups (especially term-lists/terms/reviews).
- Keep existing specialized routers first in dispatch to preserve current behavior.
- Add route-recognition tests for representative operations from each uncovered group.

Exit criteria:
- Uncovered Content Moderator routes are deterministic and routed by provider contracts.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape when supplied.
- Preserve existing validation behavior in specialized handlers.
- Ensure unknown nested routes under the prefix remain deterministic.

Exit criteria:
- Invalid API version returns stable `400 InvalidRequest` for umbrella routes.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success payloads for umbrella-recognized routes.
- Continue returning rich typed payloads for existing specialized implemented operations.

Exit criteria:
- Route handling is deterministic across all operation groups.

## Stage 3: Example and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/ai-services/content-moderator-v1.0`.
- Exercise image moderation, text/list-management, term-list, and reviews surfaces.
- Update Azure contract, IO, and doc coverage scripts with aliases and plan mapping for the new service.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve `content-moderator-v1.0`.
