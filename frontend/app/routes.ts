import { type RouteConfig, index, route } from "@react-router/dev/routes";

export default [
  index("routes/home.tsx"),
  route("stories/:id/page1", "routes/page1.tsx"),
  route("stories/:id/page2", "routes/page2.tsx"),
] satisfies RouteConfig;
