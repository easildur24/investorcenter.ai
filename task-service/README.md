# Task Service

Standalone Go microservice for managing worker tasks (ClawdBot task queue). Runs on port 8001 and is proxied by the main backend via `/api/v1/admin/workers/*` and `/api/v1/worker/*`.

## Architecture

```
Frontend (Next.js)  →  Backend (Go, :8080)  →  Task Service (Go, :8001)
                         proxy via                    ↕
                     task_service_proxy.go       PostgreSQL + S3
```

Workers (ClawdBots) pull tasks from the queue, execute SOPs, store structured data in PostgreSQL (`worker_task_data`), upload result files directly to S3 (`claw-treasure` bucket), and register file metadata via the API (`worker_task_files`).

## Quick Start

```bash
# Run locally (requires PostgreSQL + .env)
cd task-service && go run .

# Run tests
cd task-service && go test ./...

# Build binary
cd task-service && go build -o task-service .

# Docker
docker build --platform linux/amd64 -t task-service .
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8001` | HTTP server port |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `investorcenter` | PostgreSQL user |
| `DB_PASSWORD` | (required) | PostgreSQL password |
| `DB_NAME` | `investorcenter_db` | PostgreSQL database |
| `DB_SSLMODE` | `require` | PostgreSQL SSL mode |
| `JWT_SECRET` | (required) | Shared JWT signing secret (same as backend) |
| `S3_BUCKET` | `claw-treasure` | S3 bucket for result files |
| `AWS_REGION` | `us-east-1` | AWS region |

## API Endpoints

### Admin Routes (`/admin/workers/*`)
Require JWT with `is_admin=true`.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/admin/workers` | List all workers |
| `POST` | `/admin/workers` | Register a new worker |
| `DELETE` | `/admin/workers/:id` | Remove a worker |
| `GET` | `/admin/workers/task-types` | List task types |
| `POST` | `/admin/workers/task-types` | Create task type |
| `PUT` | `/admin/workers/task-types/:id` | Update task type |
| `DELETE` | `/admin/workers/task-types/:id` | Delete task type |
| `GET` | `/admin/workers/tasks` | List tasks (filterable by status, assigned_to, task_type) |
| `POST` | `/admin/workers/tasks` | Create a task |
| `GET` | `/admin/workers/tasks/:id` | Get task details |
| `PUT` | `/admin/workers/tasks/:id` | Update a task |
| `DELETE` | `/admin/workers/tasks/:id` | Delete a task |
| `GET` | `/admin/workers/tasks/:id/updates` | List task updates/comments |
| `POST` | `/admin/workers/tasks/:id/updates` | Add a task update |
| `GET` | `/admin/workers/tasks/:id/data` | List collected data rows |
| `GET` | `/admin/workers/tasks/:id/files` | List result files |
| `GET` | `/admin/workers/tasks/:id/files/:fileId/download` | Download a result file from S3 |

### Worker Routes (`/worker/*`)
Require valid JWT (worker account).

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/worker/task-types` | List available task types |
| `GET` | `/worker/task-types/:id` | Get task type with SOP |
| `GET` | `/worker/tasks` | Get my assigned tasks |
| `GET` | `/worker/tasks/:id` | Get task details |
| `PUT` | `/worker/tasks/:id/status` | Update task status |
| `POST` | `/worker/tasks/:id/result` | Post task result JSON |
| `GET` | `/worker/tasks/:id/updates` | Get task updates |
| `POST` | `/worker/tasks/:id/updates` | Post a task update |
| `POST` | `/worker/tasks/:id/data` | Bulk insert collected data |
| `POST` | `/worker/tasks/:id/files` | Register an S3 file |
| `POST` | `/worker/heartbeat` | Worker heartbeat |

## Database Tables

- **`workers`** - Worker accounts (email, password hash, activity tracking)
- **`task_types`** - Task type definitions with SOPs and parameter schemas
- **`worker_tasks`** - Task queue (status, priority, assignment, results)
- **`worker_task_updates`** - Task comments/progress updates
- **`worker_task_data`** - Structured data collected by workers (dedup by data_type + external_id)
- **`worker_task_files`** - Metadata for S3 result files

## Project Structure

```
task-service/
├── main.go                 # Entry point, route registration
├── auth/                   # JWT validation and middleware
├── database/               # PostgreSQL queries
│   ├── db.go               # Connection management
│   ├── worker_task_data.go # Structured data CRUD
│   └── worker_task_files.go# File metadata CRUD
├── handlers/               # HTTP handlers
│   ├── workers.go          # Admin endpoints
│   ├── worker_api.go       # Worker endpoints
│   └── task_types.go       # Task type endpoints
└── storage/                # S3 client (read-only, for admin downloads)
    └── s3.go
```

## Worker File Upload Flow

1. Worker claims a task and reads the SOP
2. Worker does the work (crawl, analyze, etc.)
3. Worker uploads result file directly to S3: `aws s3 cp report.txt s3://claw-treasure/worker-results/{task_id}/report.txt`
4. Worker registers the file: `POST /worker/tasks/:id/files` with `{"filename": "report.txt", "s3_key": "worker-results/{task_id}/report.txt", "content_type": "text/plain", "size_bytes": 4096}`
5. Worker marks task completed: `PUT /worker/tasks/:id/status` with `{"status": "completed"}`
6. Admin downloads via UI which calls `GET /admin/workers/tasks/:id/files/:fileId/download`

S3 keys are validated to start with `worker-results/{task_id}/` to prevent path traversal.
