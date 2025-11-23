#!/bin/bash
# Script untuk update backend di VPS
# Usage: ./update-vps.sh

echo "=== Update Xetor Backend di VPS ==="
echo ""

# Step 1: Pull perubahan dari repo
echo "Step 1: Pull perubahan dari GitHub..."
cd ~/XetorBackend
git pull origin main

if [ $? -ne 0 ]; then
    echo "❌ Error: Gagal pull dari GitHub"
    exit 1
fi

echo "✅ Pull berhasil"
echo ""

# Step 2: Cek status service
echo "Step 2: Cek status service..."
sudo systemctl status xetor-api --no-pager

echo ""
echo "=== SELANJUTNYA: Stop service dan upload binary ==="
echo ""
echo "Step 3: Stop service di VPS (jalankan di terminal VPS):"
echo "sudo systemctl stop xetor-api"
echo ""
echo "Step 4: Backup dan hapus file lama (opsional, di VPS):"
echo "cd ~/XetorBackend"
echo "cp api api.backup  # Backup (opsional)"
echo "rm api  # Hapus file lama"
echo ""
echo "Step 5: Build binary di laptop Windows (PowerShell):"
echo ""
echo "cd E:\\BAFCORP\\TechnologyInnovation\\Digibaf\\Projects\\Kotlin\\Xetor\\XetorBackend"
echo "\$env:GOOS = \"linux\""
echo "\$env:GOARCH = \"amd64\""
echo "go build -o api-linux ./cmd/api"
echo "scp \"api-linux\" bafagih@103.197.188.41:/home/bafagih/XetorBackend/api"
echo "Remove-Item Env:\\GOOS"
echo "Remove-Item Env:\\GOARCH"
echo ""
echo "Step 6: Set permission dan start service (di VPS):"
echo "chmod +x ~/XetorBackend/api"
echo "sudo systemctl start xetor-api"
echo "sudo systemctl status xetor-api"
echo "sudo journalctl -u xetor-api -n 50"

