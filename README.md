# Glossias

A web application for engaging with stories to increase the fluency of introductionary-level students.

## Installation & Setup
1. Run steps 1-3 of `Go`

### Part 1: Go
1. Install Go (1.21 or later) from [golang.org](https://golang.org)
2. Clone this repository:
   ```bash
   git clone https://github.com/shininglegend/glossias
   cd glossias
   ```
3. Install dependencies:
   ```bash
   go mod tidy
   ```
4. ```bash
   go run main.go
   ```

### To stop:
1. Ctrl-c


## Adding Content

### Stories
Add stories via the admin interface at `/admin`.

## Credits

### Content
- Most story text and audio files are by Dr. Jesse Scheumann, all rights reserved, used with permission
- All other story text and audio files were created by Titus Murphy, all rights reserved.

### Development
- Code written by Titus unless otherwise noted.
- AI assistance provided by claude.ai, GitHub Copilot, and Ollama using multiple models. Some documentation is in AiUsage.md. Developed with the Zed IDE.

## Architecture

### Frontend versus backend
This project is modular and split up into at least two main parts. The frontend is in `/frontend/*`, and is a react vite SPA. See `./frontend/routes.ts` for the available routes. The backend most of the rest of the code, but runs `./main.go` to provide the APIs for the frontend.
This uses the supabase APIs.

### Database Layer Architecture

The application uses a layered architecture for database access:

```
┌─────────────────────────────────────────────────────────────┐
│                    HTTP Handlers                            │
│              (admin/handler.go, apis/*)                     │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      │ calls functions
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                 Models Package                              │
│                (src/pkg/models/*)                           │
│  ┌─────────────────┬────────────────────────────────────┐   │
│  │ UpsertUser()    │ GetStoryData()    │ SaveStory()    │   │
│  │ CanUserAccess() │ GetAllStories()   │ DeleteStory()  │   │
│  │ IsUserAdmin()   │ GetLineText()     │ EditStory()    │   │
│  └─────────────────┴────────────────────────────────────┘   │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      │ uses SQLC queries
                      ▼
┌─────────────────────────────────────────────────────────────┐
│            Generated SQLC Queries                           │
│            (src/pkg/generated/db/*)                          │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ queries.UpsertUser()    │ queries.GetStoryData()    │    │
│  │ queries.CanUserAccess() │ queries.GetAllStories()   │    │
│  │ queries.IsUserAdmin()   │ queries.SaveStory()       │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      │ uses database connection
                      ▼
┌─────────────────────────────────────────────────────────────┐
│          Database Connection                                │
│          (pgxpool.Pool)                                     │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      │ executes type-safe SQL
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                   Supabase                                  │
└─────────────────────────────────────────────────────────────┘
```

**Key Points:**
- HTTP handlers call model functions, never database directly
- Models package uses generated SQLC queries for type-safe database operations
- SQLC generates Go code from SQL queries, providing compile-time safety
- Models package adds business logic layer on top of generated queries
- Authentication middleware calls model functions for user operations

### Academic Context
This project was, in its first part, developed under the oversight of Dr. Derrick Tate for academic credit at Sattler College.
