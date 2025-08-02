# React Conversion Plan

Converting the existing Go web application with server-side templates to a React frontend + Go backend API architecture.

## Current Architecture
- Go web server with Gorilla Mux routing
- Server-side HTML templates in `src/templates/`
- SQLite database with models in `internal/pkg/models/`
- Static assets served from `/static/`
- Story handlers in `internal/stories/`

## Target Architecture
- Go REST API backend
- React SPA frontend
- Same SQLite database
- Static assets served through React

## Phase 1: Backend API Conversion

### 1.1 Strip Template Dependencies
- Remove template engine initialization from `main.go`
- Remove template engine from handler constructors
- Update imports to remove template dependencies

### 1.2 Convert Handlers to JSON API Endpoints
Transform existing routes:
- `/` → `/api/stories` (GET) - Return JSON array of stories
- `/stories/{id}/page1` → `/api/stories/{id}/page1` (GET) - Return JSON page data
- `/stories/{id}/page2` → `/api/stories/{id}/page2` (GET) - Return JSON page data  
- `/stories/{id}/page3` → `/api/stories/{id}/page3` (GET) - Return JSON page data
- `/stories/{id}/check-vocab` → `/api/stories/{id}/check-vocab` (POST) - Accept/return JSON

### 1.3 Add CORS Middleware
```go
func corsMiddleware() mux.MiddlewareFunc {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Access-Control-Allow-Origin", "*")
            w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
            
            if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

### 1.4 Update Response Structures
Modify handlers to return JSON:
```go
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}
```

## Phase 2: React Frontend Setup

### 2.1 Initialize React App
```bash
cd logos-stories
npx create-react-app frontend
cd frontend
npm install axios react-router-dom
```

### 2.2 Project Structure
```
frontend/
├── src/
│   ├── components/
│   │   ├── StoryList.js
│   │   ├── StoryPage1.js
│   │   ├── StoryPage2.js
│   │   ├── StoryPage3.js
│   │   └── VocabChecker.js
│   ├── services/
│   │   └── api.js
│   ├── App.js
│   └── index.js
└── public/
```

### 2.3 Core Components
- **StoryList**: Display all available stories (replaces index template)
- **StoryPage1/2/3**: Story reading interfaces (replaces page templates)
- **VocabChecker**: Vocabulary quiz component
- **API Service**: Centralized API calls to Go backend

### 2.4 Routing Setup
```jsx
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<StoryList />} />
        <Route path="/stories/:id/page1" element={<StoryPage1 />} />
        <Route path="/stories/:id/page2" element={<StoryPage2 />} />
        <Route path="/stories/:id/page3" element={<StoryPage3 />} />
      </Routes>
    </Router>
  );
}
```

### 2.5 API Integration
```javascript
// services/api.js
import axios from 'axios';

const API_BASE = process.env.REACT_APP_API_URL || 'http://localhost:8080/api';

export const api = {
  getStories: () => axios.get(`${API_BASE}/stories`),
  getStoryPage: (id, page) => axios.get(`${API_BASE}/stories/${id}/page${page}`),
  checkVocab: (id, answers) => axios.post(`${API_BASE}/stories/${id}/check-vocab`, answers)
};
```

## Phase 3: Static Asset Migration

### 3.1 Move Assets
- Copy `/static/stories/` to `/frontend/public/stories/`
- Copy other static assets to `/frontend/public/`
- Update audio file paths in React components

### 3.2 Build Integration
- Configure React build to output to Go-servable directory
- Update Go to serve React build files for non-API routes

## Phase 4: Deployment Configuration

### 4.1 Production Build
```bash
# Build React app
cd frontend && npm run build

# Serve React build from Go
# Update main.go to serve build files for SPA routing
```

### 4.2 Go Server Updates
```go
// Serve React build files
r.PathPrefix("/").Handler(http.FileServer(http.Dir("frontend/build/")))

// Handle SPA routing - serve index.html for non-API routes
r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if !strings.HasPrefix(r.URL.Path, "/api/") {
        http.ServeFile(w, r, "frontend/build/index.html")
        return
    }
    http.Error(w, "Not Found", http.StatusNotFound)
})
```

## Implementation Order
1. Phase 1: Convert Go handlers to API endpoints
2. Phase 2: Build React frontend with API integration
3. Phase 3: Migrate static assets and test integration
4. Phase 4: Configure production build and deployment

## Testing Strategy
- Test API endpoints with curl/Postman before React integration
- Develop React components with mock data first
- Integration testing with both servers running
- End-to-end testing of complete user flows

## Rollback Plan
- Keep original templates and handlers in separate branch
- Feature flag to switch between template and API modes
- Database schema remains unchanged for easy rollback