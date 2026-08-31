export interface User {
  id: string;
  username: string;
  email: string;
  first_name: string;
  last_name: string;
}

export interface AuthResponse {
  user: User;
  access_token: string;
  token_type: string;
  expires_at: string;
}

export interface RegisterPayload {
  username: string;
  email: string;
  password: string;
  first_name: string;
  last_name: string;
}

interface ApiErrorPayload {
  error?: string;
  message?: string;
}

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly code?: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function request<T>(path: string, body: unknown): Promise<T> {
  let response: Response;
  try {
    response = await fetch(path, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify(body),
    });
  } catch {
    throw new ApiError(
      "Сервис авторизации недоступен. Проверьте, что backend запущен.",
      0,
    );
  }

  const payload = (await response.json().catch(() => ({}))) as ApiErrorPayload;
  if (!response.ok) {
    throw new ApiError(
      payload.message ?? "Не удалось выполнить запрос",
      response.status,
      payload.error,
    );
  }

  return payload as T;
}

export function login(username: string, password: string) {
  return request<AuthResponse>("/auth/login", { username, password });
}

export function register(payload: RegisterPayload) {
  return request<AuthResponse>("/auth/register", payload);
}

export async function closeRegistrationSession() {
  await fetch("/auth/logout", {
    method: "POST",
    credentials: "include",
  });
}
