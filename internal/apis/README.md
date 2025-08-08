# Story API

Base URL: `/api`

## Endpoints

### GET `/api/stories`
List all stories.

**Response:**
```json
{
  "success": true,
  "data": {
    "stories": [
      {
        "id": 1,
        "title": "Story Title",
        "week_number": 1,
        "day_letter": "A"
      }
    ]
  }
}
```

### GET `/api/stories/{id}/page1`
Reading page with audio.

**Response:**
```json
{
  "success": true,
  "data": {
    "story_id": "1",
    "story_title": "Story Title",
    "lines": [
      {
        "text": ["Full line of text"],
        "audio_url": "/static/stories/stories_audio/en_1A/line1.mp3",
        "has_vocab_or_grammar": false
      }
    ]
  }
}
```

### GET `/api/stories/{id}/page2`
Vocabulary exercise with blanks (`%`) and word bank.

**Response:**
```json
{
  "success": true,
  "data": {
    "story_id": "1",
    "story_title": "Story Title",
    "lines": [
      {
        "text": ["Text with ", "%", " blanks"],
        "audio_url": "/static/stories/stories_audio/en_1A/line1.mp3",
        "has_vocab_or_grammar": true
      }
    ],
    "vocab_bank": ["word1", "word2", "word3"]
  }
}
```

### GET `/api/stories/{id}/page3`
Grammar lesson with highlights (`%grammar&`).

**Response:**
```json
{
  "success": true,
  "data": {
    "story_id": "1",
    "story_title": "Story Title",
    "lines": [
      {
        "text": ["Text with ", "%", "grammar", "&", " highlighted"],
        "audio_url": "/static/stories/stories_audio/en_1A/line1.mp3",
        "has_vocab_or_grammar": true
      }
    ],
    "grammar_point": "Present Perfect Tense"
  }
}
```

### GET `/api/stories/{id}/page4`
Translation page (empty translation field - not implemented).

**Response:**
```json
{
  "success": true,
  "data": {
    "story_id": "1",
    "story_title": "Story Title",
    "lines": [
      {
        "text": ["Full line of text"],
        "audio_url": "/static/stories/stories_audio/en_1A/line1.mp3",
        "has_vocab_or_grammar": false
      }
    ],
    "translation": ""
  }
}
```

### POST `/api/stories/{id}/check-vocab`
Check vocabulary answers.

**Request:**
```json
{
  "answers": [
    {
      "line_number": 0,
      "answers": ["word1", "word2"]
    }
  ]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "answers": [
      {
        "correct": true,
        "user_answer": "word1",
        "correct_answer": "word1",
        "line": 0
      }
    ]
  }
}
```

## Error Format
```json
{
  "success": false,
  "error": "Error message"
}
```

## CORS
- Middleware adds permissive CORS for `*` origins.
- Allowed methods: GET, POST, PUT, DELETE, OPTIONS
- Allowed headers: Content-Type, Authorization

