# Trustroots KPI Dashboard

A metrics visualization system for Trustroots and Nostroots. Automated data collection and a web dashboard.


## Overview

The Trustroots KPI Dashboard provides key performance indicators for both the Trustroots community platform and its Nostr-based Nostroots extension. The system consists of a Go-based data collection service that regularly generates metrics and a static web dashboard that visualizes the data through interactive charts.

### Architecture

```
MongoDB → Go Service → JSON File → Static Dashboard
    ↓         ↓           ↓            ↓
Trustroots  Data      kpi.json    Web Browser
Database  Collection   Output     Visualization
```

## Features

### Trustroots Metrics
- **Messages**: Daily message counts and trends
- **Reviews**: Positive and negative review tracking
- **Thread Votes**: Upvote and downvote analytics
- **Reply Times**: Average time to first reply analysis

### Nostroots Metrics
- **User Adoption**: Users with Nostr public keys (npubs)
- **Activity**: Active poster counts and engagement rates
- **Content Types**: Notes categorized by kind (0, 1, 4, 30023, etc.)

### Dashboard Features
- Real-time data visualization with Chart.js
- Responsive design for mobile and desktop
- Auto-refresh every 5 minutes
- Historical trend analysis (7-day charts)
- Error handling and retry mechanisms

## Quick Start

### Option 1: Docker Compose (Recommended)

**Prerequisites:**
- Docker and Docker Compose
- Access to Trustroots MongoDB data

1. **Clone and start the services:**
   ```bash
   # Start all services
   docker-compose up --build
   
   # Or start in background
   docker-compose up --build -d
   ```

2. **Access the dashboard:**
   - Use the nginx configuration file for production
   - For development, serve the public directory with any web server

3. **Useful commands:**
   ```bash
   docker-compose logs -f      # View logs
   docker-compose restart      # Restart services
   docker-compose down         # Stop services
   docker-compose down -v      # Stop and remove volumes
   ```

### Option 2: Manual Setup

**Prerequisites:**
- Go 1.21 or later
- MongoDB instance with Trustroots data
- Web server (Nginx, Apache, or simple HTTP server)
- Access to Nostr relays (optional, uses mock data if unavailable)

1. **Clone and build the service:**
   ```bash
   go build -o kpi-service .
   ```

2. **Configure the service:**
   ```bash
   # Copy the example configuration
   cp config.example config
   
   # Edit the configuration file
   nano config
   
   # Or set environment variables directly
   export MONGO_URI="mongodb://localhost:27017"
   export MONGO_DB="trustroots"
   export OUTPUT_PATH="public/kpi.json"
   export NOSTR_RELAYS="wss://relay.trustroots.org,wss://relay.nomadwiki.org"
   export UPDATE_INTERVAL_MINUTES="60"
   ```

3. **Run the service:**
   ```bash
   # Run once and exit
   ./kpi-service --once
   
   # Run continuously with regular updates
   ./kpi-service
   ```

4. **Testing the dashboard:**
   ```bash
   # Start a local web server
   python3 -m http.server 8765 --directory public
   
   # Open in browser
   open http://localhost:8765/
   ```

## Project Structure

```
kpi.trustroots.org/
├── main.go                  # Application entry point
├── collectors/              # Data collection modules
├── models/                  # Data structures
├── go.mod                   # Go dependencies
├── go.sum                   # Go dependencies lock
├── Dockerfile               # Container configuration
├── public/                  # Static dashboard frontend
│   ├── index.html           # Interactive dashboard
│   └── kpi.json             # Generated metrics data
├── docker-compose.yml       # Docker Compose configuration
├── kpi.trustroots.org.nginx.conf # Nginx configuration
├── config.example           # Configuration template
├── .cursorrules             # Cursor IDE rules
├── LICENSE                  # Unlicense
└── README.md               # This file
```

## Development

### Local Development

1. **Backend Development:**
   - See [kpi-service/README.md](kpi-service/README.md) for detailed setup
   - Test with specific dates: `./kpi-service --once --date 2025-01-15`
   - Use test output: `OUTPUT_PATH=./test-kpi.json ./kpi-service --once`

2. **Frontend Development:**
   - Edit `public/index.html` for dashboard changes
   - Test with local data by serving the file directly
   - Charts automatically adapt to available data

### Testing

```bash
# Test data collection
MONGO_URI=mongodb://localhost:27017 \
OUTPUT_PATH=./test-kpi.json \
./kpi-service --once

# Test with specific date
./kpi-service --once --date 2025-01-15

# Verify output file
cat test-kpi.json | jq .
```

## Deployment

### Docker Compose (Recommended)

The easiest way to deploy the KPI Dashboard is using Docker Compose:

```bash
# Start services
docker-compose up -d
```

This will start:
- **MongoDB**: Database with persistent storage
- **KPI Service**: Data collection service that generates `public/kpi.json`

### Web Server Configuration

The project includes `kpi.trustroots.org.nginx.conf` for production nginx setup:

```nginx
events {
    worker_connections 1024;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    server {
        listen 80;
        root /path/to/trustroots/public;
        index index.html;

        # Cache JSON data for 5 minutes
        location /kpi.json {
            add_header Cache-Control "public, max-age=300";
        }

        # Serve static files
        location / {
            try_files $uri $uri/ =404;
        }
    }
}
```

**Production Setup:**
1. Copy `kpi.trustroots.org.nginx.conf` to your nginx configuration directory
2. Update the `root` path to point to your `public/` directory
3. Configure your domain and SSL as needed
4. The KPI service will generate `public/kpi.json` which nginx will serve

### Manual Deployment

For production without Docker, see the detailed setup in [kpi-service/README.md](kpi-service/README.md).



### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MONGO_URI` | `mongodb://localhost:27017` | MongoDB connection string (read-only) |
| `MONGO_DB` | `trustroots` | Database name |
| `NOSTR_RELAYS` | `wss://relay.trustroots.org,wss://relay.nomadwiki.org` | Comma-separated Nostr relay URLs |
| `OUTPUT_PATH` | `public/kpi.json` | Path for JSON output file |
| `UPDATE_INTERVAL_MINUTES` | `60` | Update frequency in minutes |

**Configuration File**: Copy `config.example` to `config` and modify as needed.

## Data Flow

1. **Collection**: Go service queries MongoDB for Trustroots data and Nostr relays for activity
2. **Processing**: Data is aggregated and formatted into structured JSON
3. **Output**: JSON file is written to the specified path
4. **Visualization**: Static dashboard loads JSON data and renders interactive charts
5. **Updates**: Service runs regularly to refresh data automatically

## Monitoring

### Service Health

```bash
# Check service status
systemctl status kpi-service

# View logs
journalctl -u kpi-service -f

# Verify output file
ls -la /var/www/trustroots/public/kpi.json
```

### Troubleshooting

- **MongoDB Connection**: Verify database is running and accessible
- **Output File**: Check write permissions and disk space
- **Nostr Queries**: Service falls back to mock data if relays are unavailable
- **Dashboard Loading**: Ensure JSON file is accessible via web server

## Output Format

The service generates a JSON file with the following structure:

```json
{
  "generated": "2025-01-15T12:00:00Z",
  "trustroots": {
    "messagesPerDay": [{"date": "2025-01-14", "count": 150}],
    "reviewsPerDay": [{"date": "2025-01-14", "positive": 5, "negative": 1}],
    "threadVotesPerDay": [{"date": "2025-01-14", "upvotes": 10, "downvotes": 2}],
    "timeToFirstReplyPerDay": [{"date": "2025-01-14", "avgMs": 7200000}]
  },
  "nostroots": {
    "usersWithNpubs": 234,
    "activePosters": 45,
    "notesByKindPerDay": [{"date": "2025-01-14", "kind1": 20, "kind30023": 5}]
  }
}
```


## License

This project is released into the public domain under the [Unlicense](https://unlicense.org/). See [LICENSE](LICENSE) for the full license text.

This project is part of the Trustroots ecosystem.
