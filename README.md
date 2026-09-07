# YT Agent

[![CI](https://github.com/tinnt-truman/yt-agent/actions/workflows/ci.yml/badge.svg)](https://github.com/tinnt-truman/yt-agent/actions/workflows/ci.yml)

Phân tích một kênh/video YouTube, sau đó dùng AI (DeepSeek) để "biến tấu"
thành chiến lược nội dung cho một kênh mới: định vị, content pillar, ý tưởng
video, lịch đăng, mẫu tiêu đề, từ khoá SEO, bộ hashtag.

- **Backend**: Go (net/http, PostgreSQL, YouTube Data API v3, DeepSeek API)
- **Frontend**: React + TypeScript + Vite + Tailwind

## Kiến trúc

Pipeline chạy theo từng bước (step), không dùng goroutine nền — để chạy được
cả trên server thường lẫn trên nền tảng serverless (Vercel), nơi một handler
không giữ được tiến trình chạy sau khi đã trả response:

```
Nhập link (video/kênh)
   -> POST /api/analyses: tạo job, chạy NGAY bước 1 trong cùng request rồi trả kết quả
        bước 1 (pending -> generating): resolve channel, fetch tối đa 50 video
        (YouTube Data API v3), tính số liệu thuần Go (lượt xem TB, tần suất đăng,
        video nổi bật, tag phổ biến, mẫu tiêu đề...) — không gọi AI, thường vài giây
   -> FE poll POST /api/analyses/:id/step mỗi vài giây để đẩy job sang bước kế
        bước 2 (generating -> done): gọi DeepSeek API (JSON mode) sinh chiến lược
   -> step là no-op an toàn khi job đã done/failed — gọi lại bao nhiêu lần cũng được
```

Dữ liệu thô (`analysis_json`) và kết quả AI (`ai_output_json`) được lưu tách
biệt trong Postgres.

## Yêu cầu

1. **YouTube Data API key**: tạo tại [Google Cloud Console](https://console.cloud.google.com/)
   → APIs & Services → Credentials → Create API Key. Nhớ bật **YouTube Data
   API v3** cho project. Quota mặc định 10.000 unit/ngày.
2. **DeepSeek API key**: tạo tại [platform.deepseek.com](https://platform.deepseek.com/) (mục API Keys).
3. Go >= 1.22, Node >= 20, Docker (chạy Postgres).

Hai key trên **không cần điền vào `.env`** — nhập trực tiếp trên trang **Cài
đặt** (`/config`) của web app sau khi chạy xong, xem bước 3. `.env` chỉ cần
`DATABASE_URL`.

## Chạy local

### 1. Database

```bash
cd backend
docker compose up -d
```

### 2. Backend

```bash
cd backend
cp .env.example .env   # mặc định đã trỏ đúng DATABASE_URL cho docker-compose ở trên
go run ./cmd/server
```

Server chạy ở `http://localhost:8080`.

### 3. Frontend

```bash
cd frontend
npm install
npm run dev
```

Mở `http://localhost:5173`. Vite dev server đã cấu hình proxy `/api` sang
`localhost:8080`. Vì chưa có API key, app sẽ tự chuyển tới trang **Cài đặt**
(`/config`) — nhập YouTube API key + DeepSeek API key + chọn model AI, lưu
lại, app sẽ tự về trang chủ để bắt đầu phân tích.

Key được lưu trong bảng `settings` của Postgres (không mã hoá) — phù hợp cho
dùng cá nhân/local; nếu deploy công khai, cần thêm lớp xác thực trước khi cho
truy cập trang `/config`.

## Deploy lên Vercel

Vercel không giữ được goroutine chạy nền cho runtime Go sau khi handler trả
response, nên kiến trúc step-based ở trên là bắt buộc (không phải tối ưu tuỳ
chọn) khi deploy lên đây. Cần **hai project Vercel riêng** (frontend và
backend là hai ứng dụng khác nhau) cộng với một Postgres có thể truy cập từ
Internet:

### 1. Postgres — Neon (khuyến nghị, có free tier)

Tạo project tại [neon.tech](https://neon.tech/), lấy connection string dạng
pooled (`...pooler...neon.tech/...?sslmode=require`).

### 2. Backend (Go)

- Trên Vercel: **Add New Project** → import repo này → **Root Directory**
  chọn `backend/`. Vercel tự nhận diện Go qua `go.mod` + `cmd/server/main.go`.
- Biến môi trường cần set trong project settings:
  - `DATABASE_URL` = connection string Neon ở bước 1
  - `CORS_ORIGIN` (tuỳ chọn) = danh sách domain frontend được phép gọi API,
    cách nhau bằng dấu phẩy. Để trống thì cho phép mọi origin — API này
    không dùng cookie/session nên để trống vẫn an toàn, không bắt buộc phải
    điền. Một project Vercel có nhiều domain khác nhau (domain chính, domain
    `*.vercel.app` tự sinh, domain riêng mỗi preview) nên nếu điền, nhớ liệt
    kê đủ domain bạn sẽ gọi tới, ví dụ:
    `https://yt-agent-fe.vercel.app,https://yt-agent.example.com`
- Không cần cấu hình `maxDuration` thủ công: với Fluid Compute (mặc định cho
  project mới), Vercel cho **300s/function** ngay cả ở Hobby — đủ dư cho bước
  gọi DeepSeek. `maxDuration` cũng không cấu hình được qua `functions` trong
  `vercel.json` với kiểu Go Framework Preset (chạy nguyên `cmd/server/main.go`)
  — khóa đó chỉ áp dụng cho function kiểu file trong thư mục `api/`.
- Sau khi deploy, mở `/config` trên domain backend (hoặc gọi thẳng
  `PUT /api/settings`) để nhập YouTube key + DeepSeek key — settings nằm
  trong Postgres, không phải biến môi trường, nên không cấu hình lại mỗi lần
  deploy.

### 3. Frontend (React)

- **Add New Project** khác → import cùng repo → **Root Directory** chọn
  `frontend/`. Vercel tự nhận diện Vite.
- Biến môi trường: `VITE_API_BASE_URL` = URL project backend ở bước 2 (không
  có dấu `/` ở cuối), ví dụ `https://yt-agent-backend.vercel.app`.
- Nếu backend đang để `CORS_ORIGIN` trống (cho mọi origin) thì không cần làm
  gì thêm. Nếu muốn siết lại, quay lại project backend, thêm domain frontend
  vừa có vào `CORS_ORIGIN` rồi redeploy backend.

## API

| Method | Path                       | Mô tả                                   |
|--------|----------------------------|------------------------------------------|
| POST   | `/api/analyses`            | Body `{ "url": "..." }`. Chạy đồng bộ bước 1 rồi trả về `Analysis` đầy đủ; `412` nếu chưa cấu hình key |
| POST   | `/api/analyses/:id/step`   | Đẩy job tiến 1 bước, trả về `Analysis` hiện tại. No-op nếu đã done/failed |
| GET    | `/api/analyses/:id`        | Trạng thái + kết quả đầy đủ              |
| GET    | `/api/analyses`            | Danh sách 50 phân tích gần nhất          |
| GET    | `/api/settings`            | Cấu hình hiện tại (key được che, chỉ hiện 4 ký tự cuối) |
| PUT    | `/api/settings`            | Cập nhật key/model/số video mẫu. Field key để trống = giữ nguyên giá trị cũ |

`status` đi qua các bước: `pending → fetching → analyzing → generating → done`
(hoặc `failed` kèm `errorMessage`).

## Định dạng URL YouTube được hỗ trợ

`/watch?v=`, `youtu.be/`, `/shorts/`, `/channel/UC...`, `/@handle`, `/user/name`.
`/c/CustomName` (URL cũ) được xử lý qua `search.list` — có thể không chính xác
100% vì YouTube không còn API tra cứu trực tiếp cho dạng này.
