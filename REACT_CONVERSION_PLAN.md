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

## ~~Phase 1: Add API Routes (Parallel to Templates)~~ Done.

### 1.1 Add API Routes Alongside Existing Routes
Create new API handlers without touching existing template handlers:
- Add `/api/stories` (GET) - Return JSON array of stories
- Add `/api/stories/{id}/page1` (GET) - Return JSON page data
- Add `/api/stories/{id}/page2` (GET) - Return JSON page data
- Add `/api/stories/{id}/page3` (GET) - Return JSON page data
- Add `/api/stories/{id}/check-vocab` (POST) - Accept/return JSON

**Testing**: Each API endpoint can be tested independently with curl/Postman

### 1.2 Create API-Specific Handlers
Copy existing handlers to new API versions in `internal/api/` package:
- `api/stories.go` - JSON versions of story handlers
- `api/responses.go` - Common response structures
- Keep existing template handlers untouched

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

## Phase 2: React Frontend Development (Independent)

### 2.1 Initialize React App with Mock Data
```bash
cd logos-stories
npx create-react-router@latest frontend
cd frontend
npm install axios react-router-dom
```

**Testing**: Use mock data files to develop components independently of backend

### 2.2 Project Structure
```
frontend/
├── src/
│   ├── components/
│   │   ├── StoryPages/
│   │   │   ├── Page1.jsx
│   │   │   ├── Page2.jsx
│   │   │   └── Page3.jsx
│   │   ├── StoryList.jsx
│   │   └── VocabChecker.jsx
│   ├── services/
│   │   └── api.jsx
│   ├── App.jsx
│   └── index.js
└── public/
```

### 2.3 Core Components with Mock Data First
- **StoryList**: Build with mock JSON data, test independently
- **StoryPage1/2/3**: Build with mock story data, test UI/UX
- **VocabChecker**: Build with mock vocab data, test interactions
- **API Service**: Start with mock responses, switch to real API later

**Testing**: Each component testable in isolation with mock data

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

### 2.5 API Integration with Feature Flag
```javascript
// services/api.js
import axios from 'axios';
import mockData from './mockData';

const USE_MOCK = process.env.REACT_APP_USE_MOCK === 'true';
const API_BASE = process.env.REACT_APP_API_URL || 'http://localhost:8080/api';

export const api = {
  getStories: () => USE_MOCK ? Promise.resolve({data: mockData.stories}) : axios.get(`${API_BASE}/stories`),
  getStoryPage: (id, page) => USE_MOCK ? Promise.resolve({data: mockData.pages[id][page]}) : axios.get(`${API_BASE}/stories/${id}/page${page}`),
  checkVocab: (id, answers) => USE_MOCK ? Promise.resolve({data: mockData.vocabResults}) : axios.post(`${API_BASE}/stories/${id}/check-vocab`, answers)
};
```

**Testing**: Switch between mock and real API with environment variable

## Phase 3: Gradual Page Migration

### 3.1 Single Page Migration
Start with one page (e.g., story list):
- Add feature flag to Go handlers: `?react=true` query parameter
- Serve React build for flagged requests
- Keep template version as default
- Copy only required assets to React public folder

**Testing**: Compare template vs React versions side-by-side

### 3.2 Asset Duplication Strategy
- Serve assets from both `/static/` and `/frontend/public/`
- Use relative paths in React that work with either location
- Gradual migration of individual asset folders

**Testing**: Verify assets load from both locations

## Phase 4: Feature Flag Integration

### 4.1 Route-Level Feature Flags
Add feature flags to existing routes:
```go
func (h *Handler) ServeIndex(w http.ResponseWriter, r *http.Request) {
    if r.URL.Query().Get("react") == "true" {
        http.ServeFile(w, r, "frontend/build/index.html")
        return
    }
    // Existing template logic
}
```

**Testing**: A/B test template vs React on same routes

### 4.2 Gradual Migration Strategy
- Enable React for specific story IDs first
- Use user preferences or admin flags
- Monitor performance and user feedback
- Roll back individual features if needed

**Testing**: Incremental rollout with immediate rollback capability

## Implementation Order (Incremental & Testable)
1. **Phase 1**: Add API routes parallel to templates - test each endpoint
2. **Phase 2**: Build React with mocks - test components independently
3. **Phase 3**: Single page migration with feature flags - A/B test
4. **Phase 4**: Gradual rollout with monitoring - incremental deployment

## Testing Strategy (Per Phase)
- **Phase 1**: curl/Postman test each API endpoint independently
- **Phase 2**: Jest/React Testing Library with mock data
- **Phase 3**: Compare template vs React with `?react=true` flag
- **Phase 4**: Monitor real users, immediate rollback capability

## Rollback Plan (Always Available)
- Original templates always available (no removal until Phase 4 complete)
- Feature flags allow instant rollback per route/user
- API and template handlers coexist safely
- Database unchanged - zero migration risk
- Static assets served from both locations during transition

## Modularity Benefits
- Each API endpoint developed and tested independently
- React components built with mocks, no backend dependency
- Feature flags enable granular testing and rollout
- No big-bang deployment - gradual, reversible migration
