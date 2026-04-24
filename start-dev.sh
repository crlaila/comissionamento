#!/bin/bash

# Script para iniciar o projeto em desenvolvimento
# Use em 3 terminais diferentes:
# Terminal 1: ./start-dev.sh db
# Terminal 2: ./start-dev.sh api
# Terminal 3: ./start-dev.sh web

set -e

# Carregar variáveis de ambiente
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
else
    echo "❌ Arquivo .env não encontrado!"
    exit 1
fi

case "$1" in
    db)
        echo "🐘 Iniciando PostgreSQL..."
        docker compose up -d postgres
        echo "⏳ Aguardando PostgreSQL ficar pronto..."
        until docker compose exec -T postgres pg_isready -U $DB_USER > /dev/null 2>&1; do
            sleep 1
        done
        echo "✅ PostgreSQL pronto!"
        echo ""
        echo "📦 Aplicando migrations..."
        make migrate
        echo "✅ Banco de dados pronto para usar!"
        ;;

    api)
        echo "🚀 Iniciando API Go (Backend)..."
        echo "🔗 Acesso em: http://localhost:$PORT"
        echo "🔐 JWT Secret: $JWT_SECRET"
        echo "📡 Database: $DATABASE_URL"
        echo ""
        make dev-api
        ;;

    web)
        echo "⚛️  Iniciando React (Frontend)..."
        echo "🔗 Acesso em: http://localhost:5173"
        echo ""
        make dev-web
        ;;

    all)
        echo "🎯 Iniciando projeto completo em background..."
        # Subir banco
        make migrate > /tmp/db.log 2>&1 &
        sleep 5
        # Subir API
        make dev-api > /tmp/api.log 2>&1 &
        sleep 2
        # Subir Web
        make dev-web > /tmp/web.log 2>&1 &

        echo ""
        echo "✅ Projeto iniciando! Verifique:"
        echo "   📡 API:     http://localhost:8080"
        echo "   🌐 Frontend: http://localhost:5173"
        echo ""
        echo "📋 Logs:"
        echo "   DB:  tail -f /tmp/db.log"
        echo "   API: tail -f /tmp/api.log"
        echo "   Web: tail -f /tmp/web.log"
        echo ""
        echo "Pressione Ctrl+C para parar..."
        wait
        ;;

    *)
        echo "❌ Uso: ./start-dev.sh [db|api|web|all]"
        echo ""
        echo "Exemplos:"
        echo "  Terminal 1: ./start-dev.sh db   # Suba o banco"
        echo "  Terminal 2: ./start-dev.sh api  # Inicie a API"
        echo "  Terminal 3: ./start-dev.sh web  # Inicie o React"
        echo ""
        echo "Ou em background:"
        echo "  ./start-dev.sh all"
        exit 1
        ;;
esac
