#!/bin/bash
# Script untuk cek status backend di VPS
# Usage: ./check-backend-status.sh

echo "=== Cek Status Xetor Backend di VPS ==="
echo ""

# Cek service status
echo "1. Service Status:"
sudo systemctl status xetor-api --no-pager -l
echo ""

# Cek log terakhir
echo "2. Log Terakhir (50 baris):"
sudo journalctl -u xetor-api -n 50 --no-pager
echo ""

# Cek apakah API respond
echo "3. Test API Endpoint:"
curl -s https://api.xetor.bafagih.my.id/ || echo "❌ API tidak respond"
echo ""

# Cek apakah binary ada
echo "4. Cek Binary:"
if [ -f ~/XetorBackend/api ]; then
    echo "✅ Binary ada di ~/XetorBackend/api"
    ls -lh ~/XetorBackend/api
else
    echo "❌ Binary tidak ditemukan"
fi
echo ""

# Cek environment variables
echo "5. Cek Environment Variables:"
if [ -f ~/XetorBackend/.env ]; then
    echo "✅ .env file ada"
    echo "CDN_BASE_URL: $(grep CDN_BASE_URL ~/XetorBackend/.env | cut -d'=' -f2)"
    echo "DEFAULT_PHOTO_URL: $(grep DEFAULT_PHOTO_URL ~/XetorBackend/.env | cut -d'=' -f2)"
else
    echo "❌ .env file tidak ditemukan"
fi

