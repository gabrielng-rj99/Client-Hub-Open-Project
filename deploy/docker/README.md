# Client Hub - Docker Deployment

Deploy Client Hub using Docker Compose for a consistent, containerized environment. This mode represents the recommended way to run the application in production.

## 📋 Overview

**Docker mode** runs:

- ✅ Backend API
- ✅ Frontend (static build served by Nginx)
- ✅ PostgreSQL
- ✅ Built-in backup utilities

All orchestrated using the unified `Makefile`, which replaces direct interactions with raw `docker-compose` commands.

---

## 🚀 Quick Start

Ensure you have Docker and Docker Compose installed.

### 1. Configure

Initialize and configure your environment variables:

```bash
make env-check
```

*(Review and edit the generated `.env` file before proceeding).*

### 2. Build Images

Build the container images locally:

```bash
make build
```

### 3. Start

Start the stack detached:

```bash
make up
```

### 4. Access

- **Frontend**: <http://localhost> (or via configured port)
- **Backend API**: <http://localhost:3000>

---

## 📝 Commands Reference

All commands should be executed from within `deploy/docker`.

### Management Commands

- `make start` / `make up`: Start all containers
- `make stop` / `make down`: Stop and remove containers
- `make restart`: Restart the stack
- `make build`: Build Docker images
- `make rebuild`: Build Docker images bypassing cache
- `make status`: View the status of containers and local images

### Diagnostics

- `make logs`: View combined logs
- `make logs-backend`: View backend logs
- `make logs-frontend`: View frontend logs
- `make logs-db`: View database logs
- `make health`: Ping internal health checks for the various services

### Access & Maintenance

- `make shell`: Open an interactive bash shell in the backend container
- `make shell-db`: Open an interactive Postgres session
- `make clean`: Bring down the stack and clean any hanging resources

---

## 🗄️ Database Management

The Docker deployment bundles a pg_dump-powered backup script configured via `.env`.

- `make backup`: Manually trigger a database backup
- `make backup-rotate`: Run backups via the dedicated backup service with retention rules
- `make restore`: Restore the database from the backup artifacts in your `backups` directory

## 🔄 Updates (Registry-based)

If you're utilizing CI/CD generated images (via a container registry like GitHub Container Registry), you can configure `.env.registry` and deploy via the registry update wrapper script instead of doing local builds.

- `make registry-pull`: Pull new images from the remote registry
- `make registry-migrate`: Perform schema migrations
- `make registry-update`: Pull images, migrate schema, and restart containers all at once. Or utilize `./auto-update-registry.sh` for an even more hands-off automation, including automatic rollback options on failure.
