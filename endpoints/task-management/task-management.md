# Task Management Endpoints

## Create Task

```bash
curl -X POST "http://localhost:8080/api/v1/tasks" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Complete project documentation",
    "description": "Write comprehensive documentation for the task manager system",
    "status": "TODO",
    "priority": "HIGH",
    "due_date": "2024-01-31T23:59:59Z"
  }'
```

## List My Tasks

```bash
curl -X GET "http://localhost:8080/api/v1/tasks/me?page=1&page_size=10&filter_by_status=TODO&sort_by=due_date&sort_desc=false" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## Get Task by ID

```bash
curl -X GET "http://localhost:8080/api/v1/tasks/aa1e8400-e29b-41d4-a716-446655440000" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## Update Task

```bash
curl -X PUT "http://localhost:8080/api/v1/tasks/aa1e8400-e29b-41d4-a716-446655440000" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Complete project documentation - UPDATED",
    "status": "IN_PROGRESS",
    "priority": "URGENT"
  }'
```

## Delete Task

```bash
curl -X DELETE "http://localhost:8080/api/v1/tasks/aa1e8400-e29b-41d4-a716-446655440000" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## List All Tasks (Admin Only)

```bash
curl -X GET "http://localhost:8080/api/v1/tasks?page=1&page_size=20&filter_by_priority=URGENT&sort_by=created_at&sort_desc=true" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```