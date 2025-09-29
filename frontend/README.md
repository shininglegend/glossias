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

## RTL Text Support

When implementing new story pages, ensure proper handling of right-to-left (RTL) text indentation:

1. Detect RTL languages: `["he", "ar", "fa", "ur"]`
2. Process text for leading tabs and convert to padding
3. Apply right padding for RTL text instead of left padding
4. Example implementation pattern:

```javascript
const processTextForRTL = (text, isRTL) => {
  if (!isRTL || typeof text !== "string") {
    return { displayText: text, indentLevel: 0 };
  }
  
  const leadingTabs = text.match(/^\t*/)?.[0] || "";
  const tabCount = leadingTabs.length;
  const textWithoutTabs = text.slice(tabCount);
  
  return {
    displayText: textWithoutTabs,
    indentLevel: tabCount,
  };
};

// Apply padding: paddingRight: `${indentLevel * 2}em`
```

See `StoriesVocab.tsx` and `StoriesGrammar.tsx` for reference implementations.

---

Built with ❤️ using React Router.
