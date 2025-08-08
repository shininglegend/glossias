# Server-rendered Story Pages (legacy)

Base path: `/`

These endpoints render HTML templates for the public story pages. The React frontend now handles these paths; these remain for reference/testing.

## Endpoints

- GET `/stories/{id}/page1` — Reading page with audio
- GET `/stories/{id}/page2` — Vocabulary exercise page
- GET `/stories/{id}/page3` — Grammar highlights page
- POST `/stories/{id}/check-vocab` — Checks vocabulary answers; returns JSON
- GET `/` — Index page with a list of stories

See: `story_templates.go`, `page1.go`, `page2.go`, `page3.go`.
