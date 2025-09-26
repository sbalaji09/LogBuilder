#!/bin/bash

# Development helper script for log analytics system

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is running
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker Desktop."
        exit 1
    fi
}

# Start the development environment
start() {
    print_status "Starting log analytics development environment..."
    check_docker
    
    # Start core services
    docker-compose up -d postgres redis
    
    print_status "Waiting for services to be ready..."
    sleep 10
    
    # Check service health
    if docker-compose ps postgres | grep -q "healthy\|Up"; then
        print_status "PostgreSQL is ready"
    else
        print_warning "PostgreSQL might not be fully ready yet"
    fi
    
    if docker-compose ps redis | grep -q "healthy\|Up"; then
        print_status "Redis is ready"
    else
        print_warning "Redis might not be fully ready yet"
    fi
    
    print_status "Development environment is ready!"
    print_status "PostgreSQL: localhost:5432 (user: loguser, db: logs)"
    print_status "Redis: localhost:6379"
    
    # Start optional dev tools if requested
    if [[ "$1" == "--with-tools" ]]; then
        print_status "Starting development tools..."
        docker-compose --profile dev-tools up -d adminer redis-commander
        print_status "Adminer (DB): http://localhost:8090"
        print_status "Redis Commander: http://localhost:8091"
    fi
}

# Stop the development environment
stop() {
    print_status "Stopping log analytics development environment..."
    docker-compose down
    print_status "Development environment stopped"
}

# Reset the development environment (removes all data)
reset() {
    print_warning "This will destroy all data in your development environment!"
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_status "Resetting development environment..."
        docker-compose down -v
        docker-compose up -d postgres redis
        print_status "Environment reset complete"
    else
        print_status "Reset cancelled"
    fi
}

# Show logs from services
logs() {
    if [[ -z "$1" ]]; then
        docker-compose logs -f
    else
        docker-compose logs -f "$1"
    fi
}

# Generate test data
generate_test_data() {
    print_status "Generating test data..."
    
    # Check if PostgreSQL is ready
    if ! docker-compose exec postgres pg_isready -U loguser -d logs > /dev/null 2>&1; then
        print_error "PostgreSQL is not ready. Please run './dev.sh start' first."
        exit 1
    fi
    
    # Insert sample log data
    docker-compose exec postgres psql -U loguser -d logs -c "
        INSERT INTO logs (timestamp, source, level, message, service, fields) VALUES 
        (NOW() - INTERVAL '1 hour', 'web-server-01', 'INFO', 'User login successful', 'auth', '{\"user_id\": \"12345\", \"ip\": \"192.168.1.100\"}'),
        (NOW() - INTERVAL '45 minutes', 'web-server-01', 'ERROR', 'Database connection timeout', 'auth', '{\"timeout\": \"5s\", \"retry_count\": 3}'),
        (NOW() - INTERVAL '30 minutes', 'web-server-02', 'WARN', 'High memory usage detected', 'monitoring', '{\"memory_percent\": 85}'),
        (NOW() - INTERVAL '15 minutes', 'payment-service', 'ERROR', 'Payment processing failed', 'payments', '{\"transaction_id\": \"tx_abc123\", \"amount\": 99.99}'),
        (NOW() - INTERVAL '10 minutes', 'web-server-01', 'INFO', 'Health check passed', 'health', '{}'),
        (NOW() - INTERVAL '5 minutes', 'database', 'FATAL', 'Connection pool exhausted', 'database', '{\"pool_size\": 10, \"active_connections\": 10}'),
        (NOW(), 'web-server-02', 'INFO', 'Request processed successfully', 'api', '{\"endpoint\": \"/api/users\", \"response_time\": \"120ms\"}');
    "
    
    print_status "Test data generated successfully!"
    print_status "You can now query the logs table to see sample data."
}

# Show status of services
status() {
    print_status "Development environment status:"
    docker-compose ps
}

# Connect to PostgreSQL
psql() {
    docker-compose exec postgres psql -U loguser -d logs
}

# Connect to Redis CLI
redis_cli() {
    docker-compose exec redis redis-cli
}

# Show help
help() {
    echo "Log Analytics Development Helper"
    echo
    echo "Usage: $0 [COMMAND]"
    echo
    echo "Commands:"
    echo "  start [--with-tools]  Start development environment (optionally with dev tools)"
    echo "  stop                  Stop development environment"
    echo "  reset                 Reset environment and remove all data"
    echo "  status                Show status of all services"
    echo "  logs [service]        Show logs (all services or specific service)"
    echo "  generate-test-data    Insert sample log data for testing"
    echo "  psql                  Connect to PostgreSQL"
    echo "  redis-cli             Connect to Redis CLI"
    echo "  help                  Show this help message"
    echo
    echo "Examples:"
    echo "  $0 start --with-tools   # Start with Adminer and Redis Commander"
    echo "  $0 logs postgres        # Show PostgreSQL logs"
    echo "  $0 generate-test-data   # Add sample data for testing"
}

# Main script logic
case "$1" in
    start)
        start "$2"
        ;;
    stop)
        stop
        ;;
    reset)
        reset
        ;;
    status)
        status
        ;;
    logs)
        logs "$2"
        ;;
    generate-test-data)
        generate_test_data
        ;;
    psql)
        psql
        ;;
    redis-cli)
        redis_cli
        ;;
    help|--help|-h)
        help
        ;;
    *)
        print_error "Unknown command: $1"
        help
        exit 1
        ;;
esac