# Migrating to a next.js frontend
This defines how we plan to switch to a next.js frontend from our current go templating.

Based on the repository structure and files shown, I can help you create a plan for gradually migrating from Go templates to Next.js while maintaining existing functionality. I notice you already have a `frontend` directory with some Next.js configuration files, which is a good start.

Here's a step-by-step migration plan:

0. **Initial Next.js Setup (Done)**
I see you already have the basic Next.js structure in the `frontend` directory with:
- `next.config.ts`
- `package.json`
- `app/` directory using the App Router
- Basic layout and page components

A. **Phase 1 - Public Routes First**
i. Start with the simplest public routes (non-admin):
  - Home page (`src/templates/index.html` → `frontend/app/page.tsx`)
- Keep the Go server running on port 8080
- Run Next.js on a different port (default 3000)
ii. Update the Go server to proxy requests to Next.js for migrated routes

B. **Phase 2 - Story Display**
i. Migrate the story viewing functionality
    - Go templating (`page1.html`, `page2.html`, `page3.html` → corresponding Next.js routes)
ii. Create API endpoints in Go for story data
iii. Implement story components in Next.js

C. **Phase 3 - Admin Interface**
i. Migrate admin functionality last since it's more complex
ii. Move the annotator React app into the Next.js project
iii. Create admin-specific layouts and components
