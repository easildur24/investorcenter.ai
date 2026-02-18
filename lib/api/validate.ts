/**
 * API response validation wrapper.
 *
 * Validates data against a Zod schema at the API boundary.
 * On failure it logs a warning and passes the data through
 * unchanged — never breaks the app.
 */

import type { ZodSchema } from 'zod';

export function validateResponse<T>(schema: ZodSchema<T>, data: unknown, endpoint: string): T {
  const result = schema.safeParse(data);
  if (!result.success) {
    console.warn(`[API Validation] ${endpoint}:`, result.error.issues);
    return data as T;
  }
  return result.data;
}
