import { type RouteConfig, index, route } from "@react-router/dev/routes";

export default [
  index("routes/home.tsx"),
  route("stories/:id/video", "routes/stories-video.tsx"),
  // route("stories/:id/audio", "routes/stories-audio.tsx"),
  route("stories/:id/vocab", "routes/stories-vocab.tsx"),
  route("stories/:id/identify", "routes/stories-identify.tsx"),
  route("stories/:id/grammar", "routes/stories-grammar.tsx"),
  route("stories/:id/translate", "routes/stories-translate.tsx"),
  route("stories/:id/produce", "routes/stories-produce.tsx"),
  route("stories/:id/recall", "routes/stories-recall.tsx"),
  route("stories/:id/score", "routes/stories-score.tsx"),
  // Admin SPA routes replacing server templates
  route("admin", "routes/admin.index.tsx"),
  route("admin/courses", "routes/admin.courses.tsx"),
  route("admin/courses/:id/students", "routes/admin.courses.$id.students.tsx"),
  route(
    "admin/stories/:id/students/:userId",
    "routes/admin.stories.$id.students.$userId.tsx",
  ),
  route("admin/performance", "routes/admin.performance.tsx"),
  route("admin/stories/add", "routes/admin.stories.add.tsx"),
  route("admin/stories/:id", "routes/admin.stories.$id.tsx"),
  route("admin/stories/:id/metadata", "routes/admin.stories.$id.metadata.tsx"),
  route(
    "admin/stories/:id/translate",
    "routes/admin.stories.$id.translate.tsx",
  ),
  route("admin/stories/:id/annotate", "routes/admin.stories.$id.annotate.tsx"),
  // Summer 2026 phase authoring (T7)
  route(
    "admin/stories/:id/target-vocab",
    "routes/admin.stories.$id.target-vocab.tsx",
  ),
  route("admin/stories/:id/produce", "routes/admin.stories.$id.produce.tsx"),
  route("admin/stories/:id/recall", "routes/admin.stories.$id.recall.tsx"),
  route("admin/users", "routes/admin.users.tsx"),
  route("admin/system", "routes/admin.system.tsx"),
] satisfies RouteConfig;
