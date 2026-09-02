export interface User {
  id: string;
  username: string;
  email: string;
  first_name: string;
  last_name: string;
}

export interface Item {
  id: string;
  name: string;
  description: string;
  price: number;
  image_url: string;
  created_at: string;
  updated_at: string;
}

export interface CartItem {
  id: string;
  item: Item;
  quantity: number;
  created_at: string;
  updated_at: string;
}

export interface BalanceResponse {
  balance: number;
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
  code?: string;
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

type RequestOptions = {
  method?: "GET" | "POST" | "DELETE";
  body?: unknown;
  token?: string;
};

async function request<T>(
  path: string,
  { method = "GET", body, token }: RequestOptions = {},
): Promise<T> {
  let response: Response;
  try {
    const headers = new Headers();
    if (body !== undefined) {
      headers.set("Content-Type", "application/json");
    }
    if (token) {
      headers.set("Authorization", `Bearer ${token}`);
    }

    response = await fetch(path, {
      method,
      headers,
      credentials: "include",
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(
      "Сервис недоступен. Проверьте подключение и повторите попытку.",
      0,
    );
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const payload = (await response.json().catch(() => ({}))) as ApiErrorPayload;
  if (!response.ok) {
    throw new ApiError(
      payload.message ?? "Не удалось выполнить запрос",
      response.status,
      payload.error ?? payload.code,
    );
  }

  return payload as T;
}

export function login(username: string, password: string) {
  return request<AuthResponse>("/auth/login", {
    method: "POST",
    body: { username, password },
  });
}

export function register(payload: RegisterPayload) {
  return request<AuthResponse>("/auth/register", {
    method: "POST",
    body: payload,
  });
}

export function getCurrentUser(token: string) {
  return request<User>("/auth/me", { token });
}

export function logout() {
  return request<void>("/auth/logout", { method: "POST" });
}

export function getItems() {
  return request<Item[]>("/items");
}

export function getBalance(token: string) {
  return request<BalanceResponse>("/wallet", { token });
}

export function getCart(token: string) {
  return request<CartItem[]>("/cart", { token });
}

export function addToCart(token: string, itemId: string, quantity = 1) {
  return request<CartItem>("/cart", {
    method: "POST",
    token,
    body: { item_id: itemId, quantity },
  });
}

export function removeFromCart(token: string, itemId: string) {
  return request<void>(`/cart/${encodeURIComponent(itemId)}`, {
    method: "DELETE",
    token,
  });
}
