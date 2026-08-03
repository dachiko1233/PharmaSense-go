const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:3001";

// Hard ceiling on any request. Without this, a connection that is accepted but
// never answered (a stalled proxy, a half-open socket) leaves fetch pending
// forever, which keeps react-query's isLoading true forever and hangs the page
// on its loading skeleton with no way out.
const DEFAULT_TIMEOUT_MS = 15000;

export async function apiFetch<T>(
  path: string,
  options: RequestInit = {},
  timeoutMs: number = DEFAULT_TIMEOUT_MS
): Promise<T> {
  const token = typeof window !== "undefined" ? localStorage.getItem("token") : null;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);

  let res: Response;
  try {
    res = await fetch(`${API_URL}/api/v1${path}`, {
      ...options,
      headers,
      // Honor a caller-supplied signal if present; otherwise use our timeout.
      signal: options.signal ?? controller.signal,
    });
  } catch (err) {
    if (err instanceof DOMException && err.name === "AbortError") {
      throw new Error("Request timed out. Please try again.");
    }
    throw err;
  } finally {
    clearTimeout(timer);
  }

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: "Request failed" }));
    throw new Error(error.error ?? `HTTP ${res.status}`);
  }

  return res.json();
}

export function apiUrl(path: string): string {
  return `${API_URL}/api/v1${path}`;
}
