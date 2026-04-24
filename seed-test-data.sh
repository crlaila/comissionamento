#!/bin/bash

# Script para popular dados de teste no banco
# Uso: ./seed-test-data.sh

API_URL="${API_URL:-http://localhost:8080}"

echo "🌱 Populando dados de teste..."
echo "📡 API URL: $API_URL"
echo ""

# Cores para output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ============================================================
# 1. Criar Admin
# ============================================================
echo -e "${BLUE}➤ Criando usuários...${NC}"

ADMIN_RESPONSE=$(curl -s -X POST $API_URL/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@test.com",
    "password": "Admin@123",
    "name": "Administrator",
    "role": "admin"
  }')

ADMIN_ID=$(echo $ADMIN_RESPONSE | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
echo -e "${GREEN}✓ Admin criado (ID: $ADMIN_ID)${NC}"

# ============================================================
# 2. Fazer login como admin para obter token
# ============================================================
LOGIN_RESPONSE=$(curl -s -X POST $API_URL/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@test.com",
    "password": "Admin@123"
  }')

ADMIN_TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)
echo -e "${GREEN}✓ Admin autenticado${NC}"

# ============================================================
# 3. Criar Manager (Gestor)
# ============================================================
MANAGER_RESPONSE=$(curl -s -X POST $API_URL/api/users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "email": "manager@test.com",
    "password": "Manager@123",
    "name": "Maria Gestora",
    "role": "manager"
  }')

MANAGER_ID=$(echo $MANAGER_RESPONSE | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
echo -e "${GREEN}✓ Manager criado (ID: $MANAGER_ID)${NC}"

# ============================================================
# 4. Criar Reps (Vendedores)
# ============================================================
for i in {1..3}; do
  REP_RESPONSE=$(curl -s -X POST $API_URL/api/users \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d "{
      \"email\": \"rep$i@test.com\",
      \"password\": \"Rep@123\",
      \"name\": \"Rep $i\",
      \"role\": \"rep\",
      \"manager_id\": $MANAGER_ID
    }")

  REP_ID=$(echo $REP_RESPONSE | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
  echo -e "${GREEN}✓ Rep $i criado (ID: $REP_ID)${NC}"

  if [ $i -eq 1 ]; then
    REP_1_ID=$REP_ID
  fi
done

# ============================================================
# 5. Criar Finance (Financeiro)
# ============================================================
FINANCE_RESPONSE=$(curl -s -X POST $API_URL/api/users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "email": "finance@test.com",
    "password": "Finance@123",
    "name": "Pedro Financeiro",
    "role": "finance"
  }')

FINANCE_ID=$(echo $FINANCE_RESPONSE | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
echo -e "${GREEN}✓ Finance criado (ID: $FINANCE_ID)${NC}"

# ============================================================
# 6. Criar um Período
# ============================================================
echo ""
echo -e "${BLUE}➤ Criando período de comissão...${NC}"

PERIOD_RESPONSE=$(curl -s -X POST $API_URL/api/periods \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name": "Abril 2026",
    "start_date": "2026-04-01",
    "end_date": "2026-04-30",
    "status": "open"
  }')

PERIOD_ID=$(echo $PERIOD_RESPONSE | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
echo -e "${GREEN}✓ Período criado (ID: $PERIOD_ID)${NC}"

# ============================================================
# 7. Criar Goals (Metas)
# ============================================================
echo ""
echo -e "${BLUE}➤ Criando metas de vendas...${NC}"

for REP_ID in $REP_1_ID $((REP_1_ID+1)) $((REP_1_ID+2)); do
  GOAL_RESPONSE=$(curl -s -X POST $API_URL/api/goals \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d "{
      \"rep_id\": $REP_ID,
      \"period_id\": $PERIOD_ID,
      \"acquisition_target\": 10,
      \"renewal_target\": 5,
      \"commission_value\": 100000
    }")

  GOAL_ID=$(echo $GOAL_RESPONSE | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
  echo -e "${GREEN}✓ Meta criada para Rep $REP_ID${NC}"
done

# ============================================================
# 8. Informações de Login
# ============================================================
echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo -e "${GREEN}✅ Dados de teste populados com sucesso!${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo ""
echo -e "🔐 ${BLUE}Credenciais de Login:${NC}"
echo ""
echo -e "   ${GREEN}Admin${NC}"
echo "   Email:    admin@test.com"
echo "   Senha:    Admin@123"
echo ""
echo -e "   ${GREEN}Manager (Gestor)${NC}"
echo "   Email:    manager@test.com"
echo "   Senha:    Manager@123"
echo ""
echo -e "   ${GREEN}Rep (Vendedor)${NC}"
echo "   Email:    rep1@test.com"
echo "   Email:    rep2@test.com"
echo "   Email:    rep3@test.com"
echo "   Senha:    Rep@123"
echo ""
echo -e "   ${GREEN}Finance (Financeiro)${NC}"
echo "   Email:    finance@test.com"
echo "   Senha:    Finance@123"
echo ""
echo -e "🌐 ${BLUE}Acessar a aplicação:${NC}"
echo "   http://localhost:5173"
echo ""
