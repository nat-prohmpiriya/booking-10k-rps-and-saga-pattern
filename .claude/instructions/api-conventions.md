# API Conventions

## URL Structure
```
/{version}/{resource}/{id}/{sub-resource}
```
Examples:
- `GET /v1/events`
- `GET /v1/events/:id`
- `GET /v1/events/:id/shows`
- `POST /v1/bookings/reserve`

## Response Format

### Success
```json
{
    "success": true,
    "data": { ... },
    "meta": {
        "request_id": "uuid",
        "timestamp": "ISO8601"
    }
}
```

### Error
```json
{
    "success": false,
    "error": {
        "code": "SEATS_UNAVAILABLE",
        "message": "Not enough seats available"
    },
    "meta": {
        "request_id": "uuid",
        "timestamp": "ISO8601"
    }
}
```

## HTTP Status Codes

| Code | Use Case |
|------|----------|
| 200 | Success (GET, PUT) |
| 201 | Created (POST) |
| 204 | No Content (DELETE) |
| 400 | Validation error |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not found |
| 409 | Conflict (duplicate, race condition) |
| 429 | Rate limited |
| 500 | Internal error |
