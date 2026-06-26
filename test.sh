set -e
docker compose down
docker build -t test . --no-cache
docker compose up -d
