# Stuffsy API

Backend API for Stuffsy - a collection of useful online tools.

## Features

- **URL Shortener** - Shorten long URLs with custom short codes
- **Cloud Storage** - Upload, download, and manage files via S3

## Tech Stack

- Go 1.25+
- Gin (HTTP framework)
- MongoDB (URL storage)
- Redis (caching)
- AWS S3 (file storage)

## Getting Started

### Prerequisites

- Go 1.25+
- MongoDB
- Redis
- AWS S3 bucket (or S3-compatible storage)

### Installation

```bash
git clone https://github.com/Nanyak/thangdq-lab.git
cd stuffsy-api
go mod download
```

### Configuration

Copy `.env.sample` to `.env` and fill in values:

```bash
cp .env.sample .env
```

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `9808` |
| `BASE_URL` | Public base URL | `http://localhost:9808` |
| `MONGO_URI` | MongoDB connection URI | `mongodb://localhost:27017` |
| `MONGO_DB` | Database name | `url_shortener` |
| `REDIS_ADDR` | Redis address | `localhost:6379` |
| `AWS_REGION` | AWS region | - |
| `S3_BUCKET` | S3 bucket name | - |
| `AWS_ACCESS_KEY_ID` | AWS access key | - |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key | - |
| `S3_ENDPOINT` | Custom S3 endpoint (optional) | - |

### Running

```bash
go run cmd/app/main.go
```

### Docker

```bash
docker-compose up -d
```

## API Endpoints

### URL Shortener

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/url` | Create short URL |
| `GET` | `/:shortUrl` | Redirect to original URL |

### Cloud Storage

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/files` | Upload file (multipart/form-data) |
| `GET` | `/files` | List files (query: `prefix`) |
| `DELETE` | `/files/:key` | Delete file |
| `GET` | `/files/:key/url` | Get presigned download URL |

## Project Structure

```
stuffsy-api/
├── cmd/app/          # Application entry point
├── internal/
│   ├── controller/   # HTTP handlers
│   ├── entity/       # Domain entities
│   ├── repository/   # Data access (MongoDB, Redis, S3)
│   └── usecase/      # Business logic
└── pkg/
    ├── config/       # Configuration
    └── errors/       # Error definitions
```

## License

MIT
