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

### GET `/api/stories/{id}/audio`
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

### GET `/api/stories/{id}/vocab`
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

### GET `/api/stories/{id}/grammar`
Grammar exercise with blank story lines and grammar point info.

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
        "audio_files": [],
        "has_vocab_or_grammar": true
      }
    ],
    "grammar_point": "Present Perfect Tense"
  }
}
```

### GET `/api/stories/{id}/translate`
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
Check vocabulary answers for one line.

**Request:**
```json
{
  "answers": [
    {
      "line_number": 0,
      "answers": ["word1"]
    }
  ]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "correct": true
  }
}
```

### POST `/api/stories/{id}/check-grammar`
Check grammar answers for multiple lines of same grammar point.

**Request:**
```json
{
  "grammar_point_id": 1,
  "answers": [
    {
      "line_number": 0,
      "positions": [5, 12, 20]
    },
    {
      "line_number": 2,
      "positions": [8]
    }
  ]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "correct": 3,
    "wrong": 1,
    "total_answers": 4,
    "results": [
      {
        "line_number": 0,
        "position": [5, 9],
        "text": "verb",
        "correct": true
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

