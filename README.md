# YT Agent

Phân tích một kênh/video YouTube, sau đó dùng AI (Claude) để "biến tấu" thành
chiến lược nội dung cho một kênh mới: định vị, content pillar, ý tưởng video,
lịch đăng, mẫu tiêu đề, từ khoá SEO, bộ hashtag.

- **Backend**: Go (net/http, PostgreSQL, YouTube Data API v3, Claude API)
- **Frontend**: React + TypeScript + Vite + Tailwind

## Kiến trúc

```
Nhập link (video/kênh)
   -> POST /api/analyses (trả về job id ngay, xử lý nền)
   -> resolve channel -> fetch tối đa 50 video gần nhất (YouTube Data API v3)
   -> tính toán số liệu thuần Go (không cần AI): lượt xem TB, tần suất đăng,
      video nổi bật, tag phổ biến, mẫu tiêu đề...
   -> gọi Claude API (structured output) để sinh chiến lược cho kênh mới
   -> lưu vào Postgres, FE poll GET /api/analyses/:id để cập nhật tiến trình
```

Dữ liệu thô (`analysis_json`) và kết quả AI (`ai_output_json`) được lưu tách
biệt — nếu muốn cải thiện prompt sau này, có thể re-run bước AI mà không tốn
lại quota YouTube.

## Yêu cầu

1. **YouTube Data API key**: tạo tại [Google Cloud Console](https://console.cloud.google.com/)
   → APIs & Services → Credentials → Create API Key. Nhớ bật **YouTube Data
   API v3** cho project. Quota mặc định 10.000 unit/ngày.
2. **Anthropic API key**: tạo tại [console.anthropic.com](https://console.anthropic.com/).
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
(`/config`) — nhập YouTube API key + Anthropic API key + chọn model AI, lưu
lại, app sẽ tự về trang chủ để bắt đầu phân tích.

Key được lưu trong bảng `settings` của Postgres (không mã hoá) — phù hợp cho
dùng cá nhân/local; nếu deploy công khai, cần thêm lớp xác thực trước khi cho
truy cập trang `/config`.

## API

| Method | Path                    | Mô tả                                   |
|--------|-------------------------|------------------------------------------|
| POST   | `/api/analyses`         | Body `{ "url": "..." }` → `202 { id, status }`, `412` nếu chưa cấu hình key |
| GET    | `/api/analyses/:id`     | Trạng thái + kết quả đầy đủ              |
| GET    | `/api/analyses`         | Danh sách 50 phân tích gần nhất          |
| GET    | `/api/settings`         | Cấu hình hiện tại (key được che, chỉ hiện 4 ký tự cuối) |
| PUT    | `/api/settings`         | Cập nhật key/model/số video mẫu. Field key để trống = giữ nguyên giá trị cũ |

`status` đi qua các bước: `pending → fetching → analyzing → generating → done`
(hoặc `failed` kèm `errorMessage`).

## Định dạng URL YouTube được hỗ trợ

`/watch?v=`, `youtu.be/`, `/shorts/`, `/channel/UC...`, `/@handle`, `/user/name`.
`/c/CustomName` (URL cũ) được xử lý qua `search.list` — có thể không chính xác
100% vì YouTube không còn API tra cứu trực tiếp cho dạng này.
