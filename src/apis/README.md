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

### GET `/api/stories/{id}/story-with-audio`
Story with audio.

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
Grammar exercise with story lines, grammar point info, and user progress.

**Response:**
```json
{
  "success": true,
  "data": {
    "story_id": "1",
    "story_title": "Story Title",
    "lines": [
      {
        "text": "Full line of text"
      }
    ],
    "grammar_point": "Present Perfect Tense",
    "grammar_description": "Used for actions completed in the past",
    "instances_count": 3,
    "found_instances": [
      {
        "line_number": 1,
        "position": [5, 9],
        "text": "verb"
      }
    ],
    "incorrect_instances": [
      {
        "line_number": 2,
        "position": [10, 11],
        "text": "a",
        "correct": false
      }
    ],
    "next_grammar_point": 17
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
Check single grammar selection (one click at a time).

**Request:**
```json
{
  "grammar_point_id": 1,
  "line_number": 1,
  "position": 5
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "correct": true,
    "matched_position": [5, 9],
    "total_instances": 3,
    "next_grammar_point": 17
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

