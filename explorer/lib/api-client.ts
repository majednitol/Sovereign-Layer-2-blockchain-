import { z } from "zod";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

export interface ApiError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}

export class ExplorerApiError extends Error {
  public code: string;
  public details?: Record<string, unknown>;

  constructor(error: ApiError) {
    super(error.message);
    this.name = "ExplorerApiError";
    this.code = error.code;
    this.details = error.details;
  }
}

async function request<T>(
  endpoint: string,
  schema: z.ZodType<T>,
  options?: RequestInit
): Promise<T> {
  const url = `${API_BASE}${endpoint}`;
  const response = await fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(options?.headers || {}),
    },
  });

  if (!response.ok) {
    let errorBody: ApiError = {
      code: "HTTP_ERROR",
      message: `Request failed with status ${response.status}`,
    };
    try {
      const parsed = await response.json();
      if (parsed && typeof parsed === "object" && "message" in parsed) {
        errorBody = parsed as ApiError;
      }
    } catch {
      // Use fallback error body
    }
    throw new ExplorerApiError(errorBody);
  }

  const json = await response.json();
  const parsed = schema.safeParse(json);
  if (!parsed.success) {
    console.error("Zod schema validation failed for endpoint:", endpoint, parsed.error);
    throw new ExplorerApiError({
      code: "SCHEMA_VALIDATION_ERROR",
      message: "API response format validation failed",
      details: { errors: parsed.error.format() },
    });
  }

  return parsed.data;
}

export const apiClient = {
  get: <T>(endpoint: string, schema: z.ZodType<T>, options?: RequestInit) =>
    request(endpoint, schema, { ...options, method: "GET" }),
  post: <T>(endpoint: string, schema: z.ZodType<T>, body?: unknown, options?: RequestInit) =>
    request(endpoint, schema, {
      ...options,
      method: "POST",
      body: body ? JSON.stringify(body) : undefined,
    }),
};
