#!/bin/bash
# ============================================================
# check-ports.sh — Verificar disponibilidade de porta
# Uso: ./check-ports.sh 8100
# ============================================================

PORT=${1:-8080}

echo "Verificando porta $PORT..."
if ss -tlnp | grep -q ":$PORT "; then
  echo "❌ Porta $PORT está em USO"
  ss -tlnp | grep ":$PORT "
  echo ""
  echo "Sugestão: use uma porta na faixa 3100-3999, 7000-7999, 8100-8665, 8667-8999"
else
  echo "✅ Porta $PORT está LIVRE"
fi
