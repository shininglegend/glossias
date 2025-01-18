// src/config.ts
export const API_BASE_URL = "http://localhost:8080";
export const ANNOTATIONS_ENDPOINT = (id: number) =>
  `${API_BASE_URL}/admin/stories/${id}/annotate`;
