#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

COMPOSE="docker compose -f docker/lx-2/docker-compose.yml"
URL="127.0.0.1:4444"

case "${1:-help}" in
     build)
          $COMPOSE build
     ;;
     up)
          $COMPOSE up -d --build
          $COMPOSE ps
     ;;
     down)
          $COMPOSE down
     ;;
     restart)
          $COMPOSE restart
     ;;
     logs)
          $COMPOSE logs -f leoxy
     ;;
     status)
          $COMPOSE ps
     ;;
     health)
          $COMPOSE exec leoxy wget -q -O - "http://${URL}/health/live"
     ;;
     clean)
          $COMPOSE down --rmi local
     ;;
     help|*)
          echo "Usage: ./script/docker.sh {build|up|down|restart|logs|status|health|clean}"
     ;;
esac
