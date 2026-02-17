# CORS Configuration for RAD Gateway

## Overview

RAD Gateway includes built-in CORS middleware to support browser-based clients, including the React Admin UI.

## Configuration

The default CORS configuration allows:

- **Origins**: `http://localhost:3000`, `http://localhost:5173`, `http://localhost:8080`
- **Methods**: GET, POST, PUT, PATCH, DELETE, OPTIONS
- **Headers**: Content-Type, Accept, Authorization, X-Requested-With, X-Request-Id, X-Trace-Id, X-API-Key
- **Credentials**: Enabled (cookies/auth headers allowed)
- **Max Age**: 24 hours (preflight caching)

## Backend Setup

The CORS middleware is automatically enabled in `cmd/rad-gateway/main.go`:

```go
handler := middleware.WithRequestContext(protectedMux)
handler = middleware.WithCORS(handler)  // CORS support added
```

## Frontend Setup

### Development Proxy

The Next.js dev server includes a proxy configuration in `web/next.config.js`:

```javascript
async rewrites() {
  return [
    {
      source: '/api/proxy/:path*',
      destination: 'http://172.16.30.45:8090/:path*',
    },
  ];
}
```

This allows the frontend to make requests to `/api/proxy/health` which proxies to `http://172.16.30.45:8090/health`.

### API Client Configuration

The API client (`web/src/api/client.ts`) automatically uses the proxy in development:

```typescript
const isDevelopment = process.env.NODE_ENV === 'development';
const API_BASE_URL = isDevelopment
  ? '/api/proxy'  // Uses Next.js rewrites
  : (process.env.NEXT_PUBLIC_API_URL || 'http://172.16.30.45:8090');
```

## Preflight Requests

The CORS middleware handles OPTIONS preflight requests automatically:

1. Browser sends OPTIONS request with `Origin` and `Access-Control-Request-Method` headers
2. Server responds with `204 No Content` and appropriate CORS headers
3. Browser caches the preflight response for the configured `MaxAge` duration

## Custom CORS Configuration

To customize CORS settings, modify the configuration in `main.go`:

```go
config := middleware.CORSConfig{
    AllowedOrigins:   []string{"https://mydomain.com"},
    AllowedMethods:   []string{http.MethodGet, http.MethodPost},
    AllowedHeaders:   []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
    MaxAge:           3600,
}
cors := middleware.NewCORS(config)
handler = cors.Handler(handler)
```

## Security Considerations

1. **Never use wildcard origins (`*`) with credentials enabled** - browsers reject this
2. **Keep allowed origins minimal** - only add origins you control
3. **Limit allowed methods** - only expose methods your API actually supports
4. **Be selective with headers** - only allow headers your API needs

## Troubleshooting

### CORS errors in browser console

Check that:
- The backend is running and accessible
- The origin is in the allowed origins list
- Credentials setting matches (if using auth, `AllowCredentials` must be true)

### Preflight failures

Ensure the backend handles OPTIONS requests properly. The CORS middleware returns `204 No Content` for valid preflight requests.

### Proxy not working in development

Verify:
- `web/next.config.js` has the rewrites configuration
- API URLs start with `/api/proxy/` not the full backend URL
- Backend is running at the configured proxy destination
