import { HttpException, Injectable } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';

/**
 * Thin HTTP client the Admin Service uses to aggregate data from sibling
 * services (Auth, Listing). The caller's admin Bearer token is forwarded so the
 * upstream service performs its own RS256 verification + role check.
 */
@Injectable()
export class UpstreamService {
  private readonly authBase: string;
  private readonly listingBase: string;

  constructor(config: ConfigService) {
    this.authBase = config.getOrThrow<string>('auth.baseUrl').replace(/\/+$/, '');
    this.listingBase = config.getOrThrow<string>('listing.baseUrl').replace(/\/+$/, '');
  }

  authGet<T>(path: string, authHeader?: string): Promise<T> {
    return this.request<T>(this.authBase, path, 'GET', undefined, authHeader);
  }

  authPatch<T>(path: string, body: unknown, authHeader?: string): Promise<T> {
    return this.request<T>(this.authBase, path, 'PATCH', body, authHeader);
  }

  listingGet<T>(path: string, authHeader?: string): Promise<T> {
    return this.request<T>(this.listingBase, path, 'GET', undefined, authHeader);
  }

  private async request<T>(
    base: string,
    path: string,
    method: string,
    body: unknown,
    authHeader?: string,
  ): Promise<T> {
    const headers: Record<string, string> = { Accept: 'application/json' };
    if (body !== undefined) headers['Content-Type'] = 'application/json';
    if (authHeader) headers.Authorization = authHeader;

    let res: Awaited<ReturnType<typeof fetch>>;
    try {
      res = await fetch(`${base}${path}`, {
        method,
        headers,
        body: body !== undefined ? JSON.stringify(body) : undefined,
      });
    } catch {
      throw new HttpException('upstream service unavailable', 502);
    }

    const text = await res.text();
    let data: unknown = null;
    if (text) {
      try {
        data = JSON.parse(text);
      } catch {
        data = null;
      }
    }

    if (!res.ok) {
      const message = (data as { error?: string } | null)?.error ?? `upstream error (${res.status})`;
      throw new HttpException(message, res.status);
    }
    return data as T;
  }
}
