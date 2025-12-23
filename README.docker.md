# Array Banking API - Docker Setup Guide

This guide provides comprehensive instructions for running the Array Banking API using Docker and Docker Compose.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Environment Configuration](#environment-configuration)
- [Development Workflow](#development-workflow)
- [Production Deployment](#production-deployment)
- [Database Migrations](#database-migrations)
- [Troubleshooting](#troubleshooting)
- [Architecture](#architecture)

## Prerequisites

Before you begin, ensure you have the following installed:

- **Docker** (version 20.10 or higher)
- **Docker Compose** (version 2.0 or higher)
- **Git** (for cloning the repository)

### Verify Installation

```bash
docker --version
docker compose --version
```

## Quick Start

### Development Environment

1. **Clone the repository** (if you haven't already):
   ```bash
   git clone <repository-url>
   cd array_interview_day_2
   ```

2. **Create environment file**:
   ```bash
   cp .env.example .env
   ```

3. **Start the application**:
   ```bash
   docker compose up
   ```

4. **Access the API**:
   - API: http://localhost:8080
   - Health Check: http://localhost:8080/health
   - API Documentation: http://localhost:8080/docs

5. **Stop the application**:
   ```bash
   docker compose down
   ```

## Environment Configuration

### Development Environment Variables

The `.env.example` file contains all available configuration options. Key variables:

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `APP_ENV` | Application environment | `development` | Yes |
| `APP_PORT` | Application port | `8080` | Yes |
| `LOG_LEVEL` | Logging level | `debug` | Yes |
| `DB_HOST` | Database hostname | `postgres` | Yes |
| `DB_PORT` | Database port | `5432` | Yes |
| `DB_USER` | Database username | `arraybank` | Yes |
| `DB_PASSWORD` | Database password | `arraybank_dev_password` | Yes |
| `DB_NAME` | Database name | `arraybank_dev` | Yes |
| `DB_SSLMODE` | PostgreSQL SSL mode | `disable` | Yes |
| `AUTO_MIGRATE` | Run migrations on startup | `true` | No |
| `SEED_DATABASE` | Load seed data | `true` | No |
| `JWT_SECRET` | JWT signing secret | `dev_jwt_secret_key...` | Yes |
| `JWT_EXPIRATION` | JWT expiration (seconds) | `3600` | Yes |

### Production Environment Variables

For production, use `.env.production.example` as a template:

```bash
cp .env.production.example .env.production
```

**Important**: Update these values for production:
- `DB_PASSWORD`: Use a strong, randomly generated password
- `JWT_SECRET`: Use a long, random string (minimum 32 characters)
- `CORS_ALLOWED_ORIGINS`: Set to your actual domain(s)
- `SEED_DATABASE`: Must be `false` in production

## Development Workflow

### Hot Reload with Air

The development environment uses [Air](https://github.com/air-verse/air) for automatic code reloading:

1. **Start with hot reload**:
   ```bash
   docker compose up
   ```

2. **Make code changes**: Edit any `.go` file

3. **Watch automatic rebuild**: Air detects changes and rebuilds automatically

4. **View logs**:
   ```bash
   docker compose logs -f api
   ```

### Running Specific Services

**Start only the database**:
```bash
docker compose up postgres
```

**Start only the API**:
```bash
docker compose up api
```

### Accessing the Database

**Using psql**:
```bash
docker compose exec postgres psql -U arraybank -d arraybank_dev
```

**From host machine** (if PostgreSQL client installed):
```bash
psql -h localhost -p 5432 -U arraybank -d arraybank_dev
```

### Viewing Logs

**All services**:
```bash
docker compose logs -f
```

**Specific service**:
```bash
docker compose logs -f api
docker compose logs -f postgres
```

**Last N lines**:
```bash
docker compose logs --tail=100 api
```

## Production Deployment

### Building for Production

1. **Create production environment file**:
   ```bash
   cp .env.production.example .env.production
   # Edit .env.production with production values
   ```

2. **Build production image**:
   ```bash
   docker build -t array-banking-api:latest .
   ```

3. **Verify image size**:
   ```bash
   docker images array-banking-api:latest
   # Should be around 50MB
   ```

### Running in Production

1. **Start with production compose file**:
   ```bash
   docker compose -f docker compose.prod.yml --env-file .env.production up -d
   ```

2. **Check service health**:
   ```bash
   docker compose -f docker compose.prod.yml ps
   curl http://localhost:8080/health
   ```

3. **View logs**:
   ```bash
   docker compose -f docker compose.prod.yml logs -f
   ```

4. **Stop services**:
   ```bash
   docker compose -f docker compose.prod.yml down
   ```

### Production Best Practices

1. **Use Docker Secrets** for sensitive data:
   ```bash
   echo "your-db-password" | docker secret create db_password -
   ```

2. **Enable HTTPS** with a reverse proxy (Nginx, Traefik, or Caddy)

3. **Monitor resources**:
   ```bash
   docker stats
   ```

4. **Regular backups** of PostgreSQL data volume:
   ```bash
   docker compose -f docker compose.prod.yml exec postgres pg_dump \
     -U arraybank_prod arraybank_prod > backup_$(date +%Y%m%d).sql
   ```

5. **Update images regularly**:
   ```bash
   docker compose -f docker compose.prod.yml pull
   docker compose -f docker compose.prod.yml up -d
   ```

## Database Migrations

### Automatic Migrations

Migrations run automatically on application startup when `AUTO_MIGRATE=true`.

### Manual Migration Management

**Check migration status**:
```bash
docker compose exec api ls -la /app/db/migrations
```

**View migration logs**:
```bash
docker compose logs api | grep -i migration
```

### Migration Files Location

- **Migrations**: `db/migrations/`
- **Seed data**: `db/seeds/`

### Creating New Migrations

1. Create migration files following the naming convention:
   ```
   db/migrations/XXXXXX_description.up.sql
   db/migrations/XXXXXX_description.down.sql
   ```

2. Restart the application to apply:
   ```bash
   docker compose restart api
   ```

### Seed Data

Seed data is loaded automatically in development when `SEED_DATABASE=true`.

**Sample users**:
- john.doe@example.com (admin)
- jane.smith@example.com
- bob.johnson@example.com
- alice.williams@example.com
- charlie.brown@example.com

**Default password** (all users): `Password123!`

## Troubleshooting

### Common Issues

#### 1. Port Already in Use

**Error**: `Bind for 0.0.0.0:8080 failed: port is already allocated`

**Solution**:
```bash
# Find and stop the process using the port
lsof -ti:8080 | xargs kill -9

# Or change the port in .env
APP_PORT=8081
```

#### 2. Database Connection Failed

**Error**: `failed to connect to database`

**Solutions**:
1. Ensure PostgreSQL service is healthy:
   ```bash
   docker compose ps postgres
   ```

2. Check database logs:
   ```bash
   docker compose logs postgres
   ```

3. Verify database credentials in `.env`

4. Wait for database to be ready (health check may take 10-30 seconds)

#### 3. Migration Failed

**Error**: `migration failed` or `dirty database`

**Solutions**:
1. Check migration files syntax:
   ```bash
   ls -la db/migrations/
   ```

2. Manually fix dirty state (if needed):
   ```bash
   docker compose exec postgres psql -U arraybank -d arraybank_dev \
     -c "UPDATE schema_migrations SET dirty=false WHERE version=X;"
   ```

3. Drop and recreate database (development only):
   ```bash
   docker compose down -v
   docker compose up
   ```

#### 4. Hot Reload Not Working

**Solutions**:
1. Check Air configuration:
   ```bash
   cat .air.toml
   ```

2. Verify source code is mounted:
   ```bash
   docker compose exec api ls -la /build
   ```

3. Check Air logs:
   ```bash
   docker compose logs api | grep -i air
   ```

#### 5. Permission Denied Errors

**Solutions**:
1. Fix file permissions:
   ```bash
   sudo chown -R $USER:$USER .
   ```

2. Check Docker user mapping in compose file

### Reset Everything

To completely reset the development environment:

```bash
# Stop and remove all containers, networks, and volumes
docker compose down -v

# Remove all images
docker compose down --rmi all -v

# Start fresh
docker compose up --build
```

### Health Checks

**API Health**:
```bash
curl http://localhost:8080/health
```

**Database Health**:
```bash
docker compose exec postgres pg_isready -U arraybank -d arraybank_dev
```

### Performance Issues

**Check resource usage**:
```bash
docker stats
```

**Adjust resource limits** in `docker compose.yml`:
```yaml
deploy:
  resources:
    limits:
      cpus: '2.0'
      memory: 2G
```

## Architecture

### Multi-Stage Docker Build

The Dockerfile uses a multi-stage build for optimization:

1. **Builder Stage** (`golang:1.24-alpine`):
   - Installs dependencies
   - Builds static binary with optimization flags
   - Size: ~500MB (build stage only)

2. **Runtime Stage** (`alpine:3.19`):
   - Minimal Alpine Linux base
   - Non-root user (appuser)
   - Only binary, migrations, and essential files
   - Final size: ~50MB

### Network Architecture

```
┌─────────────────────────────────────────┐
│  Host Machine                           │
│  ┌───────────────────────────────────┐  │
│  │  Docker Network (bridge)          │  │
│  │  ┌─────────────┐  ┌────────────┐  │  │
│  │  │  API (8080) │──│ PostgreSQL │  │  │
│  │  │  appuser    │  │  (5432)    │  │  │
│  │  └─────────────┘  └────────────┘  │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

### Volume Mounts

**Development**:
- `./` → `/build` (source code for hot reload)
- `go_modules` → `/go/pkg/mod` (Go module cache)
- `go_build_cache` → `/root/.cache/go-build` (build cache)
- `postgres_data` → `/var/lib/postgresql/data` (database persistence)

**Production**:
- `postgres_data` → `/var/lib/postgresql/data` (database persistence only)

### Security Features

1. **Non-root user** (UID 1000, GID 1000)
2. **Read-only filesystem** (production)
3. **Capability dropping** (production)
4. **Resource limits** (CPU, memory)
5. **Security headers** in API
6. **Rate limiting**
7. **No new privileges** flag

## Additional Resources

- [Docker Documentation](https://docs.docker.com/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [PostgreSQL Docker Hub](https://hub.docker.com/_/postgres)
- [Air GitHub Repository](https://github.com/air-verse/air)
- [golang-migrate Documentation](https://github.com/golang-migrate/migrate)

## Support

For issues or questions:
1. Check the [Troubleshooting](#troubleshooting) section
2. Review application logs: `docker compose logs`
3. Open an issue on the repository

---

**Last Updated**: 2025-10-23
