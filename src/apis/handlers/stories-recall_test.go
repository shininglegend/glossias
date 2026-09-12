package handlers

import (
	"glossias/src/apis/types"
	"glossias/src/pkg/models"
	"reflect"
	"testing"
)

func TestShuffledRecallCards(t *testing.T) {
	targetID := 7
	sentences := []models.RecallSentence{
		{ID: 1, SequenceOrder: 1, HebrewText: "הכלב רץ", ImageURL: "img-1", TargetVocabID: &targetID},
		{ID: 2, SequenceOrder: 2, HebrewText: "ב", ImageURL: "img-2"},
		{ID: 3, SequenceOrder: 3, HebrewText: "ג"},
	}
	lines := []models.StoryLine{
		{
			LineNumber: 1,
			Text:       "אתמול הכלב רץ הביתה",
			Vocabulary: []models.VocabularyItem{{LexicalForm: "כלב", Position: [2]int{6, 10}}},
		},
	}
	words := []models.TargetVocabulary{{ID: 7, LexicalForm: "כלב"}}
	audio := map[int]string{1: "line-1-audio"}
	reverse := func(n int, swap func(i, j int)) {
		for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
			swap(i, j)
		}
	}

	cards := shuffledRecallCards(sentences, lines, words, audio, reverse)

	if got := []int{cards[0].ID, cards[1].ID, cards[2].ID}; !reflect.DeepEqual(got, []int{3, 2, 1}) {
		t.Errorf("order = %v, want reversed", got)
	}
	if cards[2].ImageURL != "img-1" || cards[2].HebrewText != "הכלב רץ" {
		t.Errorf("card fields not carried over: %+v", cards[2])
	}
	if !reflect.DeepEqual(cards[2].AudioURLs, []string{"line-1-audio"}) {
		t.Errorf("audio URLs = %v, want [line-1-audio]", cards[2].AudioURLs)
	}
	if len(cards[2].Text) != 2 || cards[2].Text[0].Type != "target" || cards[2].Text[0].Text != "הכלב" {
		t.Errorf("highlighted text = %+v, want target הכלב then remainder", cards[2].Text)
	}
	// The payload type has no position field, so SequenceOrder cannot leak;
	// this guards against someone adding one later.
	if _, has := reflect.TypeOf(cards[0]).FieldByName("SequenceOrder"); has {
		t.Error("RecallCard must not expose SequenceOrder")
	}
}

func TestRecallSentenceAudio(t *testing.T) {
	lines := []models.StoryLine{
		{LineNumber: 1, Text: "שורה אחת"},
		{LineNumber: 2, Text: "שורה שתיים"},
	}
	audio := map[int]string{1: "a1", 2: "a2"}

	if got := recallSentenceAudio(models.RecallSentence{HebrewText: "שורה שתיים"}, lines, audio); !reflect.DeepEqual(got, []string{"a2"}) {
		t.Errorf("exact line match = %v, want [a2]", got)
	}
	if got := recallSentenceAudio(models.RecallSentence{HebrewText: "שורה אחת שורה שתיים"}, lines, audio); !reflect.DeepEqual(got, []string{"a1", "a2"}) {
		t.Errorf("joined-sentence chain = %v, want [a1 a2]", got)
	}
	if got := recallSentenceAudio(models.RecallSentence{HebrewText: "אין"}, lines, audio); len(got) != 0 {
		t.Errorf("unknown sentence = %v, want empty", got)
	}
	if got := recallSentenceAudio(models.RecallSentence{HebrewText: "שורה אחת", AudioURL: "override"}, lines, audio); !reflect.DeepEqual(got, []string{"override"}) {
		t.Errorf("override = %v, want [override]", got)
	}
}

func TestHighlightRecallTextFallbackLexicalForm(t *testing.T) {
	targetID := 3
	s := models.RecallSentence{HebrewText: "ראיתי כלב גדול", TargetVocabID: &targetID}
	words := []models.TargetVocabulary{{ID: 3, LexicalForm: "כלב"}}

	got := highlightRecallText(s, nil, words)
	if len(got) != 3 || got[1].Type != "target" || got[1].Text != "כלב" || got[1].TargetVocabID != 3 {
		t.Errorf("fallback highlight = %+v, want text/target/text around כלב", got)
	}
}

func TestHighlightRecallTextInflectedSurfaceForm(t *testing.T) {
	targetID := 4
	// הלך is not a substring of הולך; the annotation's Word is.
	s := models.RecallSentence{HebrewText: "עכשיו הוא הולך", TargetVocabID: &targetID}
	words := []models.TargetVocabulary{{ID: 4, LexicalForm: "הלך"}}
	lines := []models.StoryLine{{
		LineNumber: 1,
		Text:       "מי הולך שם",
		Vocabulary: []models.VocabularyItem{{Word: "הולך", LexicalForm: "הלך", Position: [2]int{3, 7}}},
	}}

	got := highlightRecallText(s, lines, words)
	if texts := targetTexts(got); !reflect.DeepEqual(texts, []string{"הולך"}) {
		t.Errorf("inflected highlight = %+v, want target הולך", got)
	}
}

func TestHighlightRecallTextJoinedLines(t *testing.T) {
	targetID := 4
	s := models.RecallSentence{HebrewText: "בוקר טוב הוא הולך הביתה", TargetVocabID: &targetID}
	words := []models.TargetVocabulary{{ID: 4, LexicalForm: "הלך"}}
	lines := []models.StoryLine{
		{LineNumber: 1, Text: "בוקר טוב"},
		{
			LineNumber: 2,
			Text:       "הוא הולך הביתה",
			Vocabulary: []models.VocabularyItem{{Word: "הולך", LexicalForm: "הלך", Position: [2]int{4, 8}}},
		},
	}

	got := highlightRecallText(s, lines, words)
	if texts := targetTexts(got); !reflect.DeepEqual(texts, []string{"הולך"}) {
		t.Errorf("joined-line highlight = %+v, want target הולך", got)
	}
}

func TestHighlightRecallTextVocalizedLemma(t *testing.T) {
	targetID := 1
	// Joined lines; איש is unvocalized but the sentence and other-line
	// annotation use אִישׁ. The lemma is not a substring of the vocalized word.
	s := models.RecallSentence{
		HebrewText:    "וַיְהִי בַבֹּקֶר וַיָּבֹא אִישׁ אֶל־הָעִיר הַגְּדֹלָה",
		TargetVocabID: &targetID,
	}
	words := []models.TargetVocabulary{{ID: 1, LexicalForm: "איש"}}
	lines := []models.StoryLine{
		{LineNumber: 1, Text: "וַיְהִי בַבֹּקֶר"},
		{LineNumber: 2, Text: "וַיָּבֹא אִישׁ אֶל־הָעִיר הַגְּדֹלָה"},
		{
			LineNumber: 8,
			Text:       "        וְגַם נָפַל מִן־הָעָם כְּאַלְפַּיִם אִישׁ",
			Vocabulary: []models.VocabularyItem{{Word: "אִישׁ", LexicalForm: "איש", Position: [2]int{37, 42}}},
		},
	}

	got := highlightRecallText(s, lines, words)
	if texts := targetTexts(got); !reflect.DeepEqual(texts, []string{"אִישׁ"}) {
		t.Errorf("vocalized surface highlight = %+v, want target אִישׁ", got)
	}
}

func TestHighlightRecallTextUnvocalizedCard(t *testing.T) {
	targetID := 4
	s := models.RecallSentence{HebrewText: "עכשיו הוא הולך", TargetVocabID: &targetID}
	words := []models.TargetVocabulary{{ID: 4, LexicalForm: "הלך"}}
	lines := []models.StoryLine{{
		Text:       "מי הוֹלֵךְ שם",
		Vocabulary: []models.VocabularyItem{{Word: "הוֹלֵךְ", LexicalForm: "הלך"}},
	}}

	got := highlightRecallText(s, lines, words)
	if texts := targetTexts(got); !reflect.DeepEqual(texts, []string{"הולך"}) {
		t.Errorf("stripped surface highlight = %+v, want target הולך", got)
	}
}

func targetTexts(segs []types.TextSegment) []string {
	var out []string
	for _, s := range segs {
		if s.Type == "target" {
			out = append(out, s.Text)
		}
	}
	return out
}

func TestRecallAttempts(t *testing.T) {
	cases := []struct {
		name    string
		summary models.AnswerSummary
		count   int
		want    int
	}{
		{"no sentences", models.AnswerSummary{CorrectCount: 5}, 0, 0},
		{"no attempts", models.AnswerSummary{}, 5, 0},
		{"one attempt", models.AnswerSummary{CorrectCount: 3, IncorrectCount: 2}, 5, 1},
		{"three attempts", models.AnswerSummary{CorrectCount: 11, IncorrectCount: 4}, 5, 3},
	}
	for _, tc := range cases {
		if got := recallAttempts(tc.summary, tc.count); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestRecallCompleted(t *testing.T) {
	sentences := []models.RecallSentence{{ID: 1}, {ID: 2}, {ID: 3}}

	if recallCompleted(nil, []int{1, 2, 3}) {
		t.Error("a story with no sentences is never complete")
	}
	if recallCompleted(sentences, nil) {
		t.Error("no correct answers should not be complete")
	}
	if recallCompleted(sentences, []int{1, 2, 2}) {
		t.Error("missing a sentence should not be complete")
	}
	// Correct placements can come from different attempts; stale IDs from a
	// sentence since deleted are ignored.
	if !recallCompleted(sentences, []int{3, 1, 99, 2, 1}) {
		t.Error("expected complete once every sentence has been placed correctly")
	}
}
