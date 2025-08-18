# 🖼️ Professional Image Resize API

A high-performance, production-ready Go API for image processing with resize, crop, watermark, and batch processing capabilities.

## ✨ Features

- **🔄 Image Resizing**: High-quality image resizing with multiple algorithms
- **✂️ Cropping**: Precise image cropping with coordinate-based positioning
- **💧 Watermarking**: Text and image watermarks with opacity control
- **📦 Batch Processing: Concurrent processing of multiple images
- **⚡ Caching**: Redis-based result caching for improved performance
- **☁️ Cloud Storage**: Supabase integration for scalable storage
- **🔄 Queue System**: RabbitMQ-based job queue for async processing
- **📊 Monitoring**: Health checks and performance metrics
- **🔒 Security**: Input validation, and security headers
- **🐳 Docker Ready**: Complete containerization setup

## 🚀 Quick Start

### Prerequisites

- Go 1.21+
- Redis (for caching)
- RabbitMQ (for queue processing)
- Supabase (for cloud storage)

### Installation

1. **Clone the repository**
```bash
git clone https://github.com/phambaophuc/image-resize.git
cd image-resize
```

2. **Install dependencies**

3. **Set up environment variables**
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. **Run the application**
```bash
go run server/main.go
```

The API will be available at `http://localhost:8080`

### Docker Setup

**Using Docker Compose (Recommended)**
```bash
docker compose up -d --build
```

This starts the API with Redis and RabbitMQ services.

**Using Docker only**
```bash
docker build -t image-resize-api .
docker run -p 8080:8080 image-resize-api
```

## 📖 API Documentation

### Base URL
```bash
http://localhost:8080/api/v1
```

### Endpoints

#### Health Check
```http
GET /health
```

#### Single Image Resize
```http
POST /images/resize
Content-Type: multipart/form-data

Parameters:
- image: Image file (required)
- width: Target width in pixels (required)
- height: Target height in pixels (required)
- quality: JPEG quality 1-100 (optional, default: 85)
- format: Output format jpeg|png|webp (optional)
- return_url: Return Storage URL instead of binary (optional)
```

**Example using curl:**
```bash
curl -X POST http://localhost:8080/api/v1/images/resize \
  -F "image=@photo.jpg" \
  -F "width=800" \
  -F "height=600" \
  -F "quality=90" \
  -F "format=jpeg"
```

#### Advanced Processing
```bash
curl -X POST http://localhost:8080/api/v1/images/process \
  -H "Content-Type: multipart/form-data" \
  -F "image=@photo.jpg" \
  -F 'payload={
    "resize": { "width": 800, "height": 600, "quality": 90 },
    "crop": { "x": 0, "y": 0, "width": 400, "height": 300 },
    "watermark": { "text": "© Your Company", "position": "bottom-right", "opacity": 0.7 },
    "compress": true
  }'
```

#### Statistics
```http
GET /stats
```

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `SUPABASE_URL` | Supabase url | - |
| `SUPABASE_KEY` | Supabase secret key | - |
| `SUPABASE_BUCKET` | Supabase bucket name | - |
| `REDIS_ADDR` | Redis address | `localhost:6379` |
| `RABBITMQ_URL` | RabbitMQ connection URL | `amqp://guest:guest@localhost:5672/` |
| `MAX_FILE_SIZE` | Maximum file size in bytes | `10485760` (10MB) |
| `CACHE_DURATION` | Cache duration | `24h` |

### Supported Image Formats

**Input**: JPEG, PNG, WebP, GIF
**Output**: JPEG, PNG, WebP

## 📊 Performance

### Benchmarks

- **Single image resize (1920x1080 → 800x600)**: ~50ms
- **Batch processing (10 images)**: ~200ms with 5 workers
- **Cache hit response**: ~5ms
- **Memory usage**: ~20MB baseline + ~2MB per concurrent request

### Scaling

- **Horizontal**: Multiple API instances behind load balancer
- **Vertical**: Increase worker count for queue processing
- **Caching**: Redis cluster for distributed caching
- **Storage**: Supabase for unlimited storage capacity

## 🔒 Security Features

- **Input validation**: File type and size validation
- **Security headers**: OWASP recommended headers
- **CORS**: Configurable cross-origin resource sharing
- **Error handling**: Secure error responses without sensitive information

## 🚦 Monitoring

### Health Check
```bash
curl http://localhost:8080/api/v1/health
```

### Metrics
```bash
curl http://localhost:8080/api/v1/stats
```

### Docker Health Check
The Docker container includes automatic health checks that monitor:
- API responsiveness
- Redis connectivity (if configured)
- RabbitMQ connectivity (if configured)

## 📈 Production Deployment

**Development**
```bash
export GIN_MODE=debug
export LOG_LEVEL=debug
```

**Production**
```bash
export GIN_MODE=release
export LOG_LEVEL=info
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

⭐ **Star this repo if it helped you build something awesome!** ⭐
