# This directory contains the frontend for this project.

## Getting Started

### Installation

Install the dependencies:

```bash
npm install
```

### Development

Start the development server with HMR:

```bash
npm run dev
```

### Admin Annotator

- The annotator UI from `annotator/` has been migrated into this app under `app/components/Annotator/*`.
- SPA route: `/admin/stories/:id/annotate` renders the annotator and talks to existing admin endpoints:
  - `GET /api/admin/stories/:id`
  - `PUT /api/admin/stories/:id`
  - `DELETE /api/admin/stories/:id`

Templated admin routes still served by Go (not migrated):

- `/admin` (home)
- `/admin/stories/add` (GET/POST)
- `/admin/stories/{id}` (GET non-JSON)/PUT
- `/admin/stories/{id}/metadata` (GET/PUT)
- `/admin/stories/delete/{id}` (GET/DELETE)

### Environment configuration

Your application will be available at `http://localhost:5173`.

## Building for Production

Create a production build:

```bash
npm run build
```

## Deployment

### Docker Deployment

To build and run using Docker:

```bash
docker build -t my-app .

# Run the container
docker run -p 3000:3000 my-app
```

The containerized application can be deployed to any platform that supports Docker, including:

- AWS ECS
- Google Cloud Run
- Azure Container Apps
- Digital Ocean App Platform
- Fly.io
- Railway

### DIY Deployment

If you're familiar with deploying Node applications, the built-in app server is production-ready.

Make sure to deploy the output of `npm run build`

```
├── package.json
├── package-lock.json (or pnpm-lock.yaml, or bun.lockb)
├── build/
│   ├── client/    # Static assets
│   └── server/    # Server-side code
```

## Styling

This template comes with [Tailwind CSS](https://tailwindcss.com/) already configured for a simple default starting experience. You can use whatever CSS framework you prefer.

---

Built with ❤️ using React Router.
