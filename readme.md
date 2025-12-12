# Task Manager Application with Microservices Architecture
A distributed task management system built using a scalable Go microservices architecture, featuring gRPC for fast inter-service communication, a dedicated API Gateway for HTTP routing, centralized authentication, task/user services, and full observability with OpenTelemetry and Jaeger. All services are containerized with Docker and orchestrated using Docker Compose for local development.

---

## Features and Tools

- **Microservices architecture with independently deployable services for authentication, user management, and task operations**
- **High-performance inter-service communication using [gRPC]((https://github.com/grpc/grpc-go)) with Protocol Buffers for strict type safety**
- **RESTful API Gateway built with [Gin](https://github.com/gin-gonic/gin) for request routing, middleware, and unified HTTP access**
- **JWT-based authentication using [golang-jwt](https://github.com/golang-jwt/jwt) for secure and stateless user sessions**
- **[PostgreSQL](https://github.com/postgres/postgres) as the primary database for reliable user and task persistence**
- **[Redis]((https://github.com/redis/redis)) caching support for future token/session optimization and performance improvements**
- **Centralized metrics collection and monitoring with [Prometheus](https://github.com/prometheus/prometheus)-compatible instrumentation**
- **Distributed tracing and observability powered by [OpenTelemetry](https://github.com/open-telemetry/opentelemetry-go), exported to [Jaeger](https://github.com/jaegertracing/jaeger) for full request lifecycle tracking**
- **Structured application logging using [Zap](https://github.com/uber-go/zap) for high-performance, production-grade log output**
- **Containerization of all services using [Docker](https://github.com/docker/compose) for isolated, reproducible environments**
- **Service orchestration with [Docker Compose](https://github.com/docker/compose) for simplified local development and multi-service management**
- **[Protocol Buffers](https://github.com/protocolbuffers/protobuf) for defining service contracts and generating gRPC server/client implementations**
- **[Swagger](https://github.com/swaggo/swag) documentation support within the API Gateway for interactive REST API exploration**
- **Clean architecture patterns with clear separation of handlers, business logic, repositories, and transport layers**

---

## How to run?

### Using Docker Compose

cd to the project directory and run this command:

```bash
docker-compose up --build -d
```

to stop all services:

```bash
docker-compose down
```

to stop services and remove database volumes:

```bash
docker-compose down -v
```

## API Endpoints

- [Authentication Endpoints](./endpoints/authentication/authentication.md)
- [User Management Endpoints](./endpoints/user-management/user-management.md)
- [Task Management Endpoints](./endpoints/task-management/task-management.md)
- [Health Check](./endpoints/health-check/health-check.md)

---

## Running Tests

### Run All Tests

```bash
cd todo-service
go test ./tests/... -v
```

### Run Specific Test Suites

#### Repository Tests Only

```bash
go test ./tests -run TestTaskRepositoryTestSuite -v
```

#### Service Tests Only

```bash
go test ./tests -run TestTaskServiceTestSuite -v
```

#### Handler Tests Only

```bash
go test ./tests -run TestTaskHandlerTestSuite -v
```

## Swagger

- [swagger docs](./api-gateway/internal/docs/docs.go)
- [swagger.json](./api-gateway/internal/docs/swagger.json)