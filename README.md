# YT Agent

[![CI](https://github.com/tinnt-truman/yt-agent/actions/workflows/ci.yml/badge.svg)](https://github.com/tinnt-truman/yt-agent/actions/workflows/ci.yml)

Phân tích một kênh/video YouTube, sau đó dùng AI (DeepSeek) để "biến tấu"
thành chiến lược nội dung cho một kênh mới: định vị, content pillar, ý tưởng
video, lịch đăng, mẫu tiêu đề, từ khoá SEO, bộ hashtag.

- **Backend**: Go (net/http, PostgreSQL, YouTube Data API v3, DeepSeek API / OpenRouter / 9Router)
- **Frontend**: React + TypeScript + Vite + Tailwind
- **Auth**: một mật khẩu chung (`APP_PASSWORD`) bảo vệ toàn bộ API — xem
  phần [Deploy lên Vercel](#deploy-lên-vercel)

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
2. **AI key** — chọn 1 trong 3 trên trang **Cài đặt** (`/config`):
   - **DeepSeek API key**: tạo tại [platform.deepseek.com](https://platform.deepseek.com/) (mục API Keys).
   - **OpenRouter API key** (có model miễn phí): tạo tại
     [openrouter.ai/keys](https://openrouter.ai/keys) (không cần thẻ cho model
     `:free`). Model free gợi ý: `openrouter/free` (auto-router),
     `nvidia/nemotron-3-ultra-550b-a55b:free` (ctx ~1M),
     `nvidia/nemotron-3.5-lightning:free` (ctx ~262K),
     `inclusionai/ling-3.0-flash-fin:free`, `minimax/m3|m2.7:free`,
     `z-ai/glm-5.2:free` — giới hạn free ~20 req/phút, ~200 req/ngày. Danh
     sách free xoay vòng, xem tại
     [openrouter.ai/collections/free-models](https://openrouter.ai/collections/free-models)
     rồi dán model ID vào ô "Custom" trên `/config`.
   - **9Router** ([github.com/decolua/9router](https://github.com/decolua/9router)):
     router AI tự host, chạy local (`npm install -g 9router && 9router`), gộp
     40+ provider (Claude Code, Kiro, GLM, Copilot...) sau một endpoint
     OpenAI-compatible duy nhất. Mặc định gọi vào `http://localhost:20128/v1`
     (cùng máy với backend) — nếu 9Router chạy máy khác, đổi "Base URL"
     trên `/config` (hoặc env `NINEROUTER_BASE_URL`) sang địa chỉ đó. Lấy
     API key và danh sách model đã kết nối (dạng `provider/model`, vd
     `cc/claude-opus-4-7`) từ dashboard 9Router, rồi dán vào `/config`.
   - Không dùng OpenCode Zen free qua API: Zen chặn gọi ngoài OpenCode client
     (`OpenCode's free tier can only be used in OpenCode`).
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

## Kênh của tôi (Google OAuth, tuỳ chọn)

Ngoài phân tích kênh công khai (YouTube Data API key), app còn cho phép
**kết nối chính kênh YouTube của bạn** qua Google để xem dữ liệu riêng tư mà
API key thường không bao giờ thấy được: giờ xem, nguồn traffic, video xem
nhiều nhất, doanh thu ước tính và trạng thái kiếm tiền — vào mục **"Kênh của
tôi"** trên nav.

Tính năng này hoàn toàn tuỳ chọn — không cấu hình thì phần còn lại của app
vẫn chạy bình thường, chỉ riêng "Kênh của tôi" báo lỗi rõ ràng khi bấm vào.

**Thiết lập trên Google Cloud Console** (project có thể dùng chung với
project đã tạo YouTube API key ở trên):

1. **APIs & Services → Library** → bật thêm **"YouTube Analytics API"**
   (khác với "YouTube Data API v3" đã bật trước đó).
2. **APIs & Services → OAuth consent screen** → chọn **External** → điền tên
   app, email hỗ trợ → ở mục **Test users**, thêm chính email Google của
   bạn. Giữ app ở chế độ **Testing** (không cần nộp Google verify) — đủ dùng
   cho 1 người vận hành; verify chỉ cần khi mở cho người ngoài dùng.
3. **APIs & Services → Credentials → Create Credentials → OAuth client ID**
   → Application type **Web application** → ở **Authorized redirect URIs**,
   thêm đúng URL callback của backend, ví dụ cả hai:
   - `http://localhost:8080/api/oauth/google/callback` (chạy local)
   - `https://yt-agent-backend.vercel.app/api/oauth/google/callback` (khi
     deploy — thay bằng domain backend thật của bạn)
4. Copy **Client ID** và **Client secret**, set vào biến môi trường backend:
   `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_OAUTH_REDIRECT_URL`
   (đúng bằng 1 trong các URI đã đăng ký ở bước 3), `FRONTEND_URL` (domain
   frontend, để backend biết redirect trình duyệt về đâu sau khi xong).

Doanh thu (`estimatedRevenue`) chỉ trả về nếu kênh đã bật kiếm tiền **và**
tài khoản Google kết nối có quyền xem doanh thu của kênh đó — nếu không, app
tự hiển thị "Chưa bật kiếm tiền" thay vì báo lỗi. Token OAuth (access +
refresh) được lưu trong bảng `connected_channels` của Postgres, không mã
hoá — cùng mức bảo mật với các API key khác trong app này.

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
  - `APP_PASSWORD` = **bắt buộc** một khi đã deploy công khai. Toàn bộ
    `/api/*` (trừ `/api/auth/login`, `/healthz`) yêu cầu đăng nhập bằng mật
    khẩu này — không set thì ai có URL cũng gọi được API, tốn quota
    YouTube/DeepSeek và đọc/đổi được key trong `/config` của bạn.
  - `CORS_ORIGIN` (tuỳ chọn) = danh sách domain frontend được phép gọi API,
    cách nhau bằng dấu phẩy. Để trống thì cho phép mọi origin — API này
    không dùng cookie/session nên để trống vẫn an toàn, không bắt buộc phải
    điền. Một project Vercel có nhiều domain khác nhau (domain chính, domain
    `*.vercel.app` tự sinh, domain riêng mỗi preview) nên nếu điền, nhớ liệt
    kê đủ domain bạn sẽ gọi tới, ví dụ:
    `https://yt-agent-fe.vercel.app,https://yt-agent.example.com`
  - `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_OAUTH_REDIRECT_URL`,
    `FRONTEND_URL` (tuỳ chọn) — chỉ cần nếu dùng "Kênh của tôi", xem phần
    [Kênh của tôi](#kênh-của-tôi-google-oauth-tuỳ-chọn) ở trên.
    `GOOGLE_OAUTH_REDIRECT_URL` phải là URL backend thật sau khi deploy
    (`https://<domain-backend>/api/oauth/google/callback`), và URL này phải
    được thêm vào Authorized redirect URIs của OAuth client trên Google
    Cloud Console.
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
- `frontend/vercel.json` đã cấu hình rewrite toàn bộ path về `index.html` —
  cần thiết vì đây là SPA dùng React Router: reload thẳng vào một route như
  `/trending` sẽ bị 404 trên static hosting nếu thiếu file này (client-side
  routing chỉ hoạt động sau khi `index.html` đã tải).

## API

| Method | Path                       | Mô tả                                   |
|--------|----------------------------|------------------------------------------|
| POST   | `/api/auth/login`          | Body `{ "password": "..." }`. Không cần token — dùng để lấy xác nhận trước khi FE lưu mật khẩu |
| POST   | `/api/analyses`            | Body `{ "url": "..." }`. Chạy đồng bộ bước 1 rồi trả về `Analysis` đầy đủ; `412` nếu chưa cấu hình key |
| POST   | `/api/analyses/:id/step`   | Đẩy job tiến 1 bước, trả về `Analysis` hiện tại. No-op nếu đã done/failed |
| GET    | `/api/analyses/:id`        | Trạng thái + kết quả đầy đủ              |
| GET    | `/api/analyses`            | Danh sách 50 phân tích gần nhất          |
| GET    | `/api/settings`            | Cấu hình hiện tại (key được che, chỉ hiện 4 ký tự cuối) |
| PUT    | `/api/settings`            | Cập nhật key/model/số video mẫu. Field key để trống = giữ nguyên giá trị cũ |
| GET    | `/api/trending?region=VN&category=20&max=25` | Báo cáo kênh đang trending theo khu vực, tuỳ chọn lọc theo `category` (id danh mục video, lấy từ endpoint dưới) — không lưu DB, live mỗi lần gọi |
| GET    | `/api/trending/categories?region=VN` | Danh sách danh mục video khả dụng theo khu vực (tên được localize theo `region`) |
| POST   | `/api/trending/insight`    | Body là `TrendingReport` (lấy từ `GET /api/trending`) → DeepSeek tóm tắt xu hướng + gợi ý cơ hội nội dung |
| GET    | `/api/oauth/google/url`    | Trả về URL đăng nhập Google (state đã ký) để FE điều hướng cả tab sang |
| GET    | `/api/oauth/google/callback` | Google redirect về đây sau khi người dùng đồng ý cấp quyền — không cần token, xác thực qua `state` |
| GET    | `/api/channels`            | Danh sách kênh đã kết nối qua Google OAuth |
| DELETE | `/api/channels/:id`        | Ngắt kết nối 1 kênh (thu hồi token ở Google + xoá khỏi DB) |
| GET    | `/api/channels/:id/analytics` | Snapshot YouTube Analytics riêng tư 28 ngày gần nhất: lượt xem, giờ xem, nguồn traffic, video xem nhiều nhất, doanh thu ước tính, trạng thái kiếm tiền |
| POST   | `/api/videos/prompt`       | Body `{ title, description?, tags? }` (lấy từ 1 video trong kết quả phân tích/trending) → DeepSeek viết prompt tiếng Anh cho công cụ AI tạo video (Kling/Runway/Sora), lấy cảm hứng cùng chủ đề chứ không sao chép |

Tất cả endpoint trên (trừ `/api/auth/login`, `/api/oauth/google/callback`,
`/healthz`) yêu cầu header `Authorization: Bearer <APP_PASSWORD>`.

`status` đi qua các bước: `pending → fetching → analyzing → generating → done`
(hoặc `failed` kèm `errorMessage`).

## Định dạng URL YouTube được hỗ trợ

`/watch?v=`, `youtu.be/`, `/shorts/`, `/channel/UC...`, `/@handle`, `/user/name`.
`/c/CustomName` (URL cũ) được xử lý qua `search.list` — có thể không chính xác
100% vì YouTube không còn API tra cứu trực tiếp cho dạng này.
