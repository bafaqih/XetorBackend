# Troubleshooting: Gagal Memuat Data di Android App

## Kemungkinan Penyebab

### 1. Backend di VPS Belum Di-update
**Gejala**: App gagal memuat data setelah push kode ke GitHub

**Solusi**: Update backend di VPS dengan kode terbaru

```bash
# Di VPS (SSH)
cd ~/XetorBackend
git pull origin main

# Di laptop Windows (PowerShell)
cd E:\BAFCORP\TechnologyInnovation\Digibaf\Projects\Kotlin\Xetor\XetorBackend
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o api-linux ./cmd/api
scp "api-linux" bafagih@103.197.188.41:/home/bafagih/XetorBackend/api
Remove-Item Env:\GOOS
Remove-Item Env:\GOARCH

# Kembali ke VPS
sudo systemctl restart xetor-api
sudo systemctl status xetor-api
```

### 2. Backend Crash atau Error
**Gejala**: Service tidak jalan atau error di log

**Cek Status**:
```bash
# Di VPS
sudo systemctl status xetor-api
sudo journalctl -u xetor-api -n 50 -f
```

**Cek API Endpoint**:
```bash
curl https://api.xetor.bafagih.my.id/
```

### 3. Token Expired atau Invalid
**Gejala**: Error 401 Unauthorized

**Solusi**: 
- Logout dan login ulang di app
- Cek token di DataStore masih valid

### 4. Network/SSL Issue di Emulator
**Gejala**: Connection timeout atau SSL error

**Solusi**:
- Pastikan emulator punya akses internet
- Cek `network_security_config.xml` sudah benar
- Coba akses API dari browser di emulator

### 5. Environment Variables Tidak Lengkap
**Gejala**: Backend crash saat startup

**Cek di VPS**:
```bash
cat ~/XetorBackend/.env | grep -E "CDN_BASE_URL|DEFAULT_PHOTO_URL|MEDIA_BASE_PATH"
```

Pastikan semua variabel ada:
- `CDN_BASE_URL=https://cdn.xetor.bafagih.my.id`
- `DEFAULT_PHOTO_URL=https://cdn.xetor.bafagih.my.id/profile/default.jpg`
- `MEDIA_BASE_PATH=/var/www/xetor/images`

## Debugging Steps

### Step 1: Cek Backend Status
```bash
# Di VPS
./check-backend-status.sh
```

### Step 2: Cek Log Android App
Di Android Studio:
- Buka Logcat
- Filter: `UserRepository` atau `HomeViewModel`
- Cari error message

### Step 3: Test API Manual
```bash
# Test endpoint public
curl https://api.xetor.bafagih.my.id/

# Test endpoint dengan auth (ganti TOKEN)
curl -H "Authorization: Bearer TOKEN" https://api.xetor.bafagih.my.id/user/profile
```

### Step 4: Cek Database Connection
```bash
# Di VPS
sudo systemctl status postgresql
sudo -u postgres psql -d xetor_db -c "SELECT COUNT(*) FROM users;"
```

## Quick Fix Checklist

- [ ] Backend service jalan: `sudo systemctl status xetor-api`
- [ ] API respond: `curl https://api.xetor.bafagih.my.id/`
- [ ] Environment variables lengkap di `.env`
- [ ] Database connection OK
- [ ] Token masih valid (coba login ulang)
- [ ] Network security config OK
- [ ] Kode terbaru sudah di-deploy ke VPS

