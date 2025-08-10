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

### PUT `/api/admin/stories/{id}/annotations`

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

### DELETE `/api/admin/stories/{id}/annotations`

Clears all annotations for a story.

Response:

```json
{ "success": true }
```

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
    "Content": { "Lines": [] }
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

- Or `application/x-www-form-urlencoded` with equivalent fields.

Response:

```json
{ "success": true, "storyId": 123 }
```

### DELETE `/api/admin/stories/{id}`

Deletes the story; logs deletion to `logs/deletions_YYYY-MM.log`.

Response:

```json
{ "success": true }
```
