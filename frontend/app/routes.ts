import { type RouteConfig, index, route } from "@react-router/dev/routes";

export default [
  index("routes/home.tsx"),
  route("stories/:id/page1", "routes/page1.tsx"),
] satisfies RouteConfig;
