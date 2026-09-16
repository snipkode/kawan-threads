# Panduan Setup Threads API

Panduan end-to-end untuk menghubungkan KAWAN Threads ke Threads API Meta —
mulai dari membuat Meta app sampai mendapatkan Client ID, Client Secret, User ID,
dan access token (long-lived, 60 hari).

Alur OAuth sudah terpasang di app ini:

- `GET /api/auth/threads` → redirect ke Threads untuk otorisasi
- `GET /api/auth/threads/callback` → callback, token disimpan otomatis ke
  datastore (tidak perlu manual)
- Scope yang dipakai (hard-coded di `internal/infrastructure/threads/adapter.go`):
  `threads_basic`, `threads_content_publish`, `threads_manage_insights`,
  `threads_read_replies`

---

## Prasyarat

- Meta developer account — https://developers.facebook.com (login pakai Facebook).
- Satu akun Threads yang akan dipakai untuk uji/publish.
- Untuk produksi: Business Portfolio dan Instagram PRO / Professional profile
  sebagai linked business asset.

## 1. Buat Meta app

1. Buka **My Apps → Create App**.
2. Pilih **type: Business**.
3. Isi nama app + email kontak, pilih Business Portfolio (buat baru jika belum ada).
4. Klik **Create App**.

## 2. Tambah produk Threads

Di halaman app: **Add product to your app → Threads → Set up**.

Jika menu tidak muncul, pastikan app dalam mode yang benar (lihat langkah 5).

## 3. Ambil Client ID & Client Secret

Menu **App settings → Basic**:

| Field Meta | Setting di UI |
|-----------|---------------|
| **App ID** | `threads_client_id` |
| **App Secret** (klik **Show**) | `threads_client_secret` |

## 4. Daftarkan Redirect URI

Menu **Threads → API setup → Valid OAuth redirect URIs**. Daftarkan (lokal &
produksi):

```
http://localhost:8090/api/auth/threads/callback
https://api.kawan.app/api/auth/threads/callback
```

Isi juga di UI Settings → `threads_redirect_uri` — ambil yang sesuai lingkungan
yang dipakai (dev = `http://localhost:8090/api/auth/threads/callback`).

> **PENTING:** redirect URI harus **100% sama** (termasuk `http` vs `https`,
> tanpa trailing slash) antara Meta app, `threads_redirect_uri` di UI, dan base
> URL API (dari `VITE_API_BASE_URL`).

## 5. Mode app & tester

- **Development Mode** (uji sendiri): tambahkan akun pengetes di
  **App roles → Users**. Email tersebut bias authorize app tanpa App Review.
- **Produksi**: perlu **App Review** untuk scope
  `threads_basic`, `threads_content_publish`, `threads_manage_insights`,
  `threads_read_replies`.

## 6. Generate access token — cara otomatis (disarankan)

1. Di UI **Settings** isi `threads_client_id`, `threads_client_secret`,
   `threads_redirect_uri`, `threads_user_id` (User ID dari langkah 7 jika belum
   tahu).
2. Buka `http://localhost:8090/api/auth/threads` di browser.
3. Login Threads → **Authorize**.
4. Callback menyimpan **long-lived token** (60 hari) ke datastore.
5. Status chip "Status koneksi" di UI berubah menjadi **Terkoneksi**.

## 7. Ambil User ID

User ID diperlukan untuk semua publish endpoint dan tidak disimpan otomatis oleh
callback. Ambil lewat Graph API setelah punya token:

```bash
curl -X GET "https://graph.threads.net/v1.0/me?fields=id,username&access_token=LONG_LIVED_TOKEN"
```

Nilai `id` dari respons = `threads_user_id`.

### Alternatif manual (tanpa UI)

```bash
# 1) Buka URL berikut di browser, izinkan akses, salin ?code=... dari redirect:
# http://localhost:8090/api/auth/threads

# 2) Tukar code → short-lived token
curl -X GET "https://graph.threads.net/oauth/access_token?client_id=CLIENT_ID&client_secret=CLIENT_SECRET&grant_type=authorization_code&redirect_uri=http://localhost:8090/api/auth/threads/callback&code=CODE"

# 3) Upgrade → long-lived token (60 hari)
curl -X GET "https://graph.threads.net/access_token?grant_type=th_exchange_token&client_secret=CLIENT_SECRET&access_token=SHORT_LIVED_TOKEN"
```

## Verifikasi

```bash
curl -s localhost:8090/api/settings/status
```

- `threads_configured: true` — client_id + user_id terisi.
- `threads_connected: true` — token tersimpan dan non-kosong.

Keduanya `true` = siap publish: buat konten → Approve → Queue → scheduler AMAB
yang menerbitkan thread pada slot terbaik.

## Ringkasan field

| UI / runtime key | Sumber |
|-----------------|--------|
| `threads_client_id` | Meta App ID |
| `threads_client_secret` | Meta App Secret |
| `threads_redirect_uri` | `http(s)://<API>/api/auth/threads/callback` |
| `threads_user_id` | `id` dari `GET /v1.0/me` |
| `threads_access_token` | Otomatis tersimpan via OAuth callback (atau manual) |