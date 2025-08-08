# Admin API

Base path: `/admin`

CORS: Allows `http://localhost:3000`, `http://localhost:5173`. Methods: GET, PUT, DELETE, OPTIONS. Headers: Content-Type.

## Stories

Base path: `/admin/stories`

### GET `/admin` [UNUSED]
- Renders HTML admin home. Supplanted by React route `/admin`.

### GET `/admin/stories/{id}/annotate` [UNUSED]
- Renders HTML annotator page. Supplanted by React route `/admin/stories/:id/annotate`.

### GET `/admin/stories/api/{id}`
Returns current story content (for annotator).

Response:
```json
{
  "content": {
    "lines": [
      {
        "lineNumber": 1,
        "text": "...",
        "vocabulary": [{"word": "...", "lexicalForm": "...", "position": [0, 3]}],
        "grammar": [{"text": "...", "position": [10, 15]}],
        "footnotes": [{"id": 1, "text": "...", "references": ["..."]}]
      }
    ]
  }
}
```

### PUT `/admin/stories/api/{id}`
Adds a single annotation to a line.

Request (one of vocabulary/grammar/footnote):
```json
{
  "lineNumber": 3,
  "vocabulary": {"word": "form", "lexicalForm": "forma", "position": [5, 9]}
}
```
Response:
```json
{"success": true}
```

### DELETE `/admin/stories/api/{id}`
Clears all annotations for a story.

Response:
```json
{"success": true}
```

### GET `/admin/stories/{id}`
- If `Accept: text/html` (default): renders HTML edit page.
- If `Accept: application/json`: returns story JSON.

Response (JSON):
```json
{
  "Success": true,
  "Story": { "Metadata": {"StoryID": 1, "WeekNumber": 1, "DayLetter": "A", "Title": {"en": "..."}}, "Content": {"Lines": []}}
}
```

### PUT `/admin/stories/{id}`
Updates the full story.

Request body: `models.Story` JSON.
Response:
```json
{ "Success": true, "Story": { /* updated story */ } }
```

### GET `/admin/stories/{id}/metadata`
- If `Accept: text/html`: renders HTML metadata page.
- If `Accept: application/json`: returns metadata via same wrapper as edit.

Response (JSON):
```json
{ "Success": true, "Story": { "Metadata": { /* ... */ } } }
```

### PUT `/admin/stories/{id}/metadata`
Updates metadata only.

Request body: `models.StoryMetadata` JSON.
Response:
```json
{"success": true}
```

### GET `/admin/stories/add` [UNUSED]
- Renders HTML add story form. Supplanted by React `/admin/stories/add`.

### POST `/admin/stories/add`
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
{"success": true, "storyId": 123}
```

### GET `/admin/stories/delete/{id}`
### DELETE `/admin/stories/delete/{id}`
Deletes the story; logs deletion to `logs/deletions_YYYY-MM.log`.

Response:
```json
{"success": true}
```


