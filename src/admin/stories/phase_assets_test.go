package stories

import "testing"

func TestValidateAssetPath(t *testing.T) {
	const storyID = 12
	const ownerID = 34

	tests := []struct {
		name       string
		kind       phaseAssetKind
		path       string
		wantBucket string
		wantOK     bool
	}{
		{
			name:       "a path minted for this word's image is accepted",
			kind:       assetTargetVocabImage,
			path:       "stories/12/image_target_vocab_34_1700000000_dog.png",
			wantBucket: imagesBucket,
			wantOK:     true,
		},
		{
			name:       "a path minted for this word's audio is accepted",
			kind:       assetTargetVocabAudio,
			path:       "stories/12/word_audio_34_1700000000_dog.mp3",
			wantBucket: bucket,
			wantOK:     true,
		},
		{
			name:       "a path minted for this recall sentence is accepted",
			kind:       assetRecallImage,
			path:       "stories/12/image_recall_34_1700000000_scene.png",
			wantBucket: imagesBucket,
			wantOK:     true,
		},
		{
			name:       "an empty path clears the asset",
			kind:       assetTargetVocabImage,
			path:       "",
			wantBucket: "",
			wantOK:     true,
		},
		{
			name:   "another story's path is rejected",
			kind:   assetTargetVocabImage,
			path:   "stories/99/image_target_vocab_34_1700000000_dog.png",
			wantOK: false,
		},
		{
			name:   "another word's path is rejected",
			kind:   assetTargetVocabImage,
			path:   "stories/12/image_target_vocab_35_1700000000_dog.png",
			wantOK: false,
		},
		{
			name:   "an audio path cannot fill an image slot",
			kind:   assetTargetVocabImage,
			path:   "stories/12/word_audio_34_1700000000_dog.mp3",
			wantOK: false,
		},
		{
			name:   "an image path cannot fill an audio slot",
			kind:   assetTargetVocabAudio,
			path:   "stories/12/image_target_vocab_34_1700000000_dog.png",
			wantOK: false,
		},
		{
			name:   "a recall image cannot be attached to a target word",
			kind:   assetTargetVocabImage,
			path:   "stories/12/image_recall_34_1700000000_scene.png",
			wantOK: false,
		},
		{
			name:   "a line audio file cannot become word audio",
			kind:   assetTargetVocabAudio,
			path:   "stories/12/line_3_complete_1700000000_line.mp3",
			wantOK: false,
		},
		{
			name:   "a traversal attempt is rejected",
			kind:   assetTargetVocabImage,
			path:   "../../stories/12/image_target_vocab_34_x.png",
			wantOK: false,
		},
		{
			name:   "an unknown kind is rejected",
			kind:   phaseAssetKind("story_illustration"),
			path:   "stories/12/image_target_vocab_34_1700000000_dog.png",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBucket, gotOK := validateAssetPath(tt.kind, storyID, ownerID, tt.path)

			if gotOK != tt.wantOK {
				t.Fatalf("ok = %v, want %v", gotOK, tt.wantOK)
			}
			if gotOK && gotBucket != tt.wantBucket {
				t.Errorf("bucket = %q, want %q", gotBucket, tt.wantBucket)
			}
		})
	}
}

// The owner ID is part of the minted prefix, so a word's own path must not be
// accepted for a different owner even within the same story.
func TestAssetPathPrefixIsOwnerScoped(t *testing.T) {
	spec, ok := specForKind(assetTargetVocabImage)
	if !ok {
		t.Fatal("target vocab image kind must have a spec")
	}

	first := spec.assetPathPrefix(1, 10)
	second := spec.assetPathPrefix(1, 11)
	if first == second {
		t.Errorf("prefixes for different owners collide: %q", first)
	}

	// "..._1_" must not be a prefix of "..._10_" or attaching word 10's upload
	// to word 1 would pass validation.
	if _, ok := validateAssetPath(assetTargetVocabImage, 1, 1, spec.assetPathPrefix(1, 10)+"x.png"); ok {
		t.Error("word 10's path was accepted for word 1")
	}
}

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"dog.png", "dog.png"},
		{"../../etc/passwd", "etcpasswd"},
		{"sub/dir/dog.png", "subdirdog.png"},
		{`windows\path\dog.png`, "windowspathdog.png"},
		{"..", ""},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := sanitizeFileName(tt.in); got != tt.want {
				t.Errorf("sanitizeFileName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
