interface CachedJSONOptions {
  key?: string
  ttlMs?: number
  persist?: boolean
}

interface StoredResponse {
  body: string
  etag?: string
  expiresAt: number
  lastModified?: string
  dataVersion?: string
}

const memoryCache = new Map<string, StoredResponse>()
const inFlight = new Map<string, Promise<unknown>>()

export function apiCacheKey(method: string, path: string, body?: unknown) {
  const bodyKey = body === undefined ? '' : stableJSONStringify(body)
  return `${method.toUpperCase()} ${path} ${bodyKey}`
}

export function stableJSONStringify(value: unknown): string {
  return JSON.stringify(canonicalize(value))
}

export async function cachedJSON<T>(
  url: string,
  init: RequestInit = {},
  options: CachedJSONOptions = {},
): Promise<T> {
  const key = options.key
  const cached = key ? readCached(key, Boolean(options.persist)) : undefined
  const now = Date.now()

  if (cached && cached.expiresAt > now) {
    return parseBody<T>(cached.body)
  }

  if (key && !init.signal) {
    const active = inFlight.get(key)
    if (active) return active as Promise<T>
  }

  const request = fetchAndCache<T>(url, init, options, cached)
  if (!key || init.signal) return request

  inFlight.set(key, request)
  try {
    return await request
  } finally {
    inFlight.delete(key)
  }
}

async function fetchAndCache<T>(
  url: string,
  init: RequestInit,
  options: CachedJSONOptions,
  cached?: StoredResponse,
): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  if (cached?.etag) {
    headers.set('If-None-Match', cached.etag)
  }

  const response = await fetch(url, {
    ...init,
    headers,
  })

  if (response.status === 304 && cached) {
    const refreshed = {
      ...cached,
      expiresAt: expiresAt(options.ttlMs),
    }
    writeCached(options.key, refreshed, Boolean(options.persist))
    return parseBody<T>(cached.body)
  }

  const body = await response.text()
  if (!response.ok) {
    throw new Error(messageFromErrorBody(body, response.status))
  }

  if (options.key) {
    writeCached(
      options.key,
      {
        body,
        etag: response.headers.get('ETag') || cached?.etag,
        lastModified: response.headers.get('Last-Modified') || cached?.lastModified,
        dataVersion: response.headers.get('X-Data-Version') || cached?.dataVersion,
        expiresAt: expiresAt(options.ttlMs),
      },
      Boolean(options.persist),
    )
  }

  return parseBody<T>(body)
}

function readCached(key: string, persist: boolean): StoredResponse | undefined {
  const memoryHit = memoryCache.get(key)
  if (memoryHit) return memoryHit

  if (!persist || typeof window === 'undefined') return undefined

  try {
    const raw = window.sessionStorage.getItem(storageKey(key))
    if (!raw) return undefined

    const parsed = JSON.parse(raw) as StoredResponse
    memoryCache.set(key, parsed)
    return parsed
  } catch {
    return undefined
  }
}

function writeCached(key: string | undefined, value: StoredResponse, persist: boolean) {
  if (!key) return

  memoryCache.set(key, value)
  if (!persist || typeof window === 'undefined') return

  try {
    window.sessionStorage.setItem(storageKey(key), JSON.stringify(value))
  } catch {
    // Session storage can be unavailable or full; memory cache is still enough for this page load.
  }
}

function storageKey(key: string) {
  return `co-tuong-wiki:api:${key}`
}

function expiresAt(ttlMs = 0) {
  return Date.now() + Math.max(0, ttlMs)
}

function parseBody<T>(body: string): T {
  return (body ? JSON.parse(body) : undefined) as T
}

function canonicalize(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map(canonicalize)
  }
  if (!value || typeof value !== 'object') {
    return value
  }

  return Object.keys(value as Record<string, unknown>)
    .sort()
    .reduce<Record<string, unknown>>((result, key) => {
      const next = (value as Record<string, unknown>)[key]
      if (next !== undefined) {
        result[key] = canonicalize(next)
      }
      return result
    }, {})
}

function messageFromErrorBody(body: string, status: number) {
  if (!body) return `Request failed with ${status}`

  try {
    const payload = JSON.parse(body) as unknown

    if (payload && typeof payload === 'object') {
      const record = payload as Record<string, unknown>
      if (typeof record.error === 'string' && record.error) return record.error
      if (typeof record.message === 'string' && record.message) return record.message
    }
  } catch {
    // Non-JSON error bodies can be shown as-is.
  }

  return body
}
