package stories

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"glossias/src/auth"
	"glossias/src/pkg/models"

	"github.com/gorilla/mux"
)

// Asset handling for the Summer 2026 phases.
//
// Bytes move exactly the way T4 moves them — a signed Supabase upload URL, a
// direct PUT from the browser, then a confirm step — but the confirm step writes
// the path onto the row that owns the asset (target_vocabulary.audio_path /
// .correct_image_path, recall_sentences.image_path) instead of inserting a
// story_images row. Those columns are the source of truth for which asset
// belongs to which word or sentence: the relationship is 1:1, so a FK-bearing
// row expresses it better than a label string. story_images is left untouched by
// the authoring editors.
//
// Unlike T4's generic /image/upload, the path here is derived from the owning
// row rather than a caller-supplied label, and the bucket is never taken from
// the request. That means a caller cannot invent a path outside its story's
// prefix, and cannot attach an audio file where an image is expected.

// phaseAssetKind identifies which column an upload is destined for.
type phaseAssetKind string

const (
	assetTargetVocabImage phaseAssetKind = "target_vocab_image"
	assetTargetVocabAudio phaseAssetKind = "target_vocab_audio"
	assetRecallImage      phaseAssetKind = "recall_image"

	// signedURLExpiry matches the student-facing expiry in src/apis/handlers.
	signedURLExpiry = 60 * 60
)

// assetSpec is the storage location an asset kind writes to. The filename prefix
// doubles as the validation prefix when a path is later attached to its row, so
// a path minted for one kind can never be accepted for another.
type assetSpec struct {
	bucket string
	prefix string // file-name prefix within stories/{storyID}/
}

func specForKind(kind phaseAssetKind) (assetSpec, bool) {
	switch kind {
	case assetTargetVocabImage:
		return assetSpec{bucket: imagesBucket, prefix: "image_target_vocab_"}, true
	case assetTargetVocabAudio:
		return assetSpec{bucket: bucket, prefix: "word_audio_"}, true
	case assetRecallImage:
		return assetSpec{bucket: imagesBucket, prefix: "image_recall_"}, true
	default:
		return assetSpec{}, false
	}
}

// assetPathPrefix is the full path prefix every upload of this kind for this
// owner must start with.
func (s assetSpec) assetPathPrefix(storyID, ownerID int) string {
	return "stories/" + strconv.Itoa(storyID) + "/" + s.prefix + strconv.Itoa(ownerID) + "_"
}

// sanitizeFileName strips the separators and parent references that would let a
// filename escape its story prefix.
func sanitizeFileName(name string) string {
	name = strings.ReplaceAll(name, "/", "")
	name = strings.ReplaceAll(name, "\\", "")
	name = strings.ReplaceAll(name, "..", "")
	return name
}

type phaseAssetUploadRequest struct {
	Kind     phaseAssetKind `json:"kind"`
	OwnerID  int            `json:"ownerId"`
	FileName string         `json:"fileName"`
}

type phaseAssetUploadResponse struct {
	UploadURL  string `json:"uploadUrl"`
	FilePath   string `json:"filePath"`
	FileBucket string `json:"fileBucket"`
}

// phaseAssetUploadHandler mints a signed upload URL for a target word's audio or
// picture, or a recall sentence's picture. The caller PUTs the bytes to
// UploadURL and then attaches FilePath via the owning row's editor endpoint.
func (h *Handler) phaseAssetUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	storyID, ok := h.authorizeStoryEdit(w, r)
	if !ok {
		return
	}

	var req phaseAssetUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	spec, known := specForKind(req.Kind)
	if !known {
		writeJSONError(w, "Unknown asset kind", http.StatusBadRequest)
		return
	}

	fileName := sanitizeFileName(strings.TrimSpace(req.FileName))
	if fileName == "" {
		writeJSONError(w, "fileName is required", http.StatusBadRequest)
		return
	}

	// The owner row must exist and belong to this story, so an upload can never
	// be minted into another story's prefix.
	if err := h.verifyAssetOwner(r, req.Kind, storyID, req.OwnerID); err != nil {
		h.writeOwnerError(w, err, req.Kind)
		return
	}

	filePath := spec.assetPathPrefix(storyID, req.OwnerID) +
		strconv.FormatInt(time.Now().Unix(), 10) + "_" + fileName

	uploadURL, err := models.GenerateSignedUploadURL(r.Context(), spec.bucket, filePath)
	if err != nil {
		h.log.Error("Failed to generate signed upload URL", "error", err, "storyID", storyID, "kind", req.Kind)
		writeJSONError(w, "Failed to generate upload URL", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(phaseAssetUploadResponse{
		UploadURL:  uploadURL,
		FilePath:   filePath,
		FileBucket: spec.bucket,
	})
}

// verifyAssetOwner confirms the target word or recall sentence exists and
// belongs to the story in the URL.
func (h *Handler) verifyAssetOwner(r *http.Request, kind phaseAssetKind, storyID, ownerID int) error {
	if ownerID <= 0 {
		return models.ErrNotFound
	}

	switch kind {
	case assetTargetVocabImage, assetTargetVocabAudio:
		word, err := models.GetTargetVocabulary(r.Context(), ownerID)
		if err != nil {
			return err
		}
		if word.StoryID != storyID {
			return models.ErrNotFound
		}
	case assetRecallImage:
		sentence, err := models.GetRecallSentence(r.Context(), ownerID)
		if err != nil {
			return err
		}
		if sentence.StoryID != storyID {
			return models.ErrNotFound
		}
	}

	return nil
}

func (h *Handler) writeOwnerError(w http.ResponseWriter, err error, kind phaseAssetKind) {
	if err == models.ErrNotFound {
		writeJSONError(w, "Asset owner not found for this story", http.StatusNotFound)
		return
	}
	h.log.Error("Failed to verify asset owner", "error", err, "kind", kind)
	writeJSONError(w, "Internal server error", http.StatusInternalServerError)
}

// validateAssetPath accepts an uploaded path for one asset kind and owner, and
// returns the bucket it must be recorded under. An empty path clears the asset.
func validateAssetPath(kind phaseAssetKind, storyID, ownerID int, path string) (bucket string, ok bool) {
	if path == "" {
		return "", true
	}

	spec, known := specForKind(kind)
	if !known {
		return "", false
	}

	if !strings.HasPrefix(path, spec.assetPathPrefix(storyID, ownerID)) {
		return "", false
	}

	return spec.bucket, true
}

// authorizeStoryEdit resolves the {id} path variable and checks the caller may
// edit that story. The admin middleware has already established that the caller
// is some kind of admin; this narrows it to this story's course.
func (h *Handler) authorizeStoryEdit(w http.ResponseWriter, r *http.Request) (int, bool) {
	storyID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		writeJSONError(w, "Invalid story ID", http.StatusBadRequest)
		return 0, false
	}

	userID, ok := auth.GetUserIDWithOk(r)
	if !ok || !models.CanUserEditStory(r.Context(), userID, int32(storyID)) {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return 0, false
	}

	return storyID, true
}
