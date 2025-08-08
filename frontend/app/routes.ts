import { type RouteConfig, index, route } from "@react-router/dev/routes";

export default [
  index("routes/home.tsx"),
  route("stories/:id/page1", "routes/page1.tsx"),
  route("stories/:id/page2", "routes/page2.tsx"),
  route("stories/:id/page3", "routes/page3.tsx"),
  route("stories/:id/page4", "routes/page4.tsx"),
  // Admin annotator SPA route. Go also serves /admin/stories/{id}/annotate via templates.
  route("admin/stories/:id/annotate", "routes/admin.annotate.$id.tsx"),
  // Admin SPA routes replacing server templates
  route("admin", "routes/admin.index.tsx"),
  route("admin/stories/add", "routes/admin.stories.add.tsx"),
  route("admin/stories/:id", "routes/admin.stories.$id.tsx"),
  route("admin/stories/:id/metadata", "routes/admin.stories.$id.metadata.tsx"),
] satisfies RouteConfig;
