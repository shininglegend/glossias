// Centralized config for API/admin base URLs from environment

function requiredEnv(name: string): string | undefined {
  const value = (import.meta as any).env?.[name];
  return typeof value === "string" && value.length > 0 ? value : undefined;
}

export function getApiBase(): string {
  const value = requiredEnv("VITE_API_URL");
  if (value) return value;
  throw new Error("VITE_API_URL is not set. Define it in your .env file.");
}

export function getAdminBase(): string {
  const explicit = requiredEnv("VITE_ADMIN_URL");
  if (explicit) return explicit;
  // Derive admin base origin from API base URL if not explicitly set
  const apiBase = getApiBase();
  try {
    const url = new URL(apiBase);
    return `${url.protocol}//${url.host}`;
  } catch {
    throw new Error("VITE_ADMIN_URL not set and VITE_API_URL is not a valid URL");
  }
}


