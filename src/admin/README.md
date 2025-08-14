# Admin API

Base path: `/api/admin`

## Stories

Base path: `/api/admin/stories`

### GET `/api/admin/stories/{id}`

Returns current story content

Response:

```json
{
  "content": {
    "lines": [
      {
        "lineNumber": 1,
        "text": "...",
        "vocabulary": [
          { "word": "...", "lexicalForm": "...", "position": [0, 3] }
        ],
        "grammar": [{ "text": "...", "position": [10, 15] }],
        "footnotes": [{ "id": 1, "text": "...", "references": ["..."] }]
      }
    ]
  }
}
```

### GET `/api/admin/stories/{id}/annotations`

Returns all annotations for a story, optionally filtered by line number.

Query parameters:
- `line` (optional): Get annotations for specific line number

Response (all annotations):

```json
{
  "1": {
    "lineNumber": 1,
    "vocabulary": [
      { "word": "form", "lexicalForm": "forma", "position": [5, 9] }
    ],
    "grammar": [
      { "text": "בקר", "position": [10, 15] }
    ],
    "footnotes": [
      { "id": 1, "text": "Note about this line", "references": ["ref1"] }
    ]
  },
  "3": {
    "lineNumber": 3,
    "vocabulary": [],
    "grammar": [],
    "footnotes": []
  }
}
```

Response (specific line with `?line=1`):

```json
{
  "lineNumber": 1,
  "vocabulary": [
    { "word": "form", "lexicalForm": "forma", "position": [5, 9] }
  ],
  "grammar": [
    { "text": "בקר", "position": [10, 15] }
  ],
  "footnotes": [
    { "id": 1, "text": "Note about this line", "references": ["ref1"] }
  ]
}
```

### POST `/api/admin/stories/{id}/annotations`

Adds a single annotation to a line.

Request (one of vocabulary/grammar/footnote):

```json
{
  "lineNumber": 3,
  "vocabulary": { "word": "form", "lexicalForm": "forma", "position": [5, 9] }
}
```

Response:

```json
{ "success": true }
```

### PUT `/api/admin/stories/{id}/annotations`

Edits an existing annotation on a line. Requires both the annotation data and an identifier.

Request examples:

Edit vocabulary by position (requires vocabularyPosition):
--DISABLED

Edit vocabulary lexical form only (by word):
```json
{
  "lineNumber": 3,
  "vocabulary": { "word": "form", "lexicalForm": "updated_forma" }
}
```

Edit grammar (requires grammarPosition):
--DISABLED.

Edit footnote (requires footnoteId):
```json
{
  "lineNumber": 3,
  "footnote": { "text": "Updated footnote text", "references": ["ref1"] },
  "footnoteId": 42
}
```

Response:

```json
{ "success": true }
```

### DELETE `/api/admin/stories/{id}/annotations`

Clears all annotations for a story, or optionally for a specific line.

Query parameters:
- `line` (optional): Delete annotations for specific line number only

Response:

```json
{ "success": true }
```

Examples:
- `DELETE /api/admin/stories/123/annotations` - Clears all annotations for story 123
- `DELETE /api/admin/stories/123/annotations?line=5` - Clears annotations only for line 5

### GET `/api/admin/stories/{id}`

Returns story JSON.

Response:

```json
{
  "Success": true,
  "Story": {
    "Metadata": {
      "StoryID": 1,
      "WeekNumber": 1,
      "DayLetter": "A",
      "Title": { "en": "..." }
    },
    "Content": { "Lines": [...] }
  }
}
```

### PUT `/api/admin/stories/{id}`

Updates the full story.

Request body: `models.Story` JSON.
Response:

```json
{
  "Success": true,
  "Story": {
    /* updated story */
  }
}
```

### GET `/api/admin/stories/{id}/metadata`

Returns metadata via same wrapper as edit.

Response (JSON):

```json
{
  "Success": true,
  "Story": {
    "Metadata": {
      /* ... */
    }
  }
}
```

### PUT `/api/admin/stories/{id}/metadata`

Updates metadata only.

Request body: `models.StoryMetadata` JSON.
Response:

```json
{ "success": true }
```

### POST `/api/admin/stories`

Creates a new story.

- Accepts `application/json`:

```json
{
  "titleEn": "...",
  "languageCode": "en",
  "authorName": "...",
  "weekNumber": 1,
  "dayLetter": "A",
  "descriptionText": "...",
  "storyText": "line 1\nline 2"
}
```

Response:

```json
{ "success": true, "storyId": 123 }
```

### DELETE `/api/admin/stories/{id}`

Deletes the story; logs deletion.

Response:

```json
{ "success": true }
```
