# Plan — Codex reset credits parity (sửa lại W1/W2)

Viết 2026-07-26. Nguồn sự thật: `router-for-me/Cli-Proxy-API-Management-Center`
(clone tại scratchpad `panel/`), file `src/components/quota/quotaConfigs.ts`,
`src/utils/quota/resetCredits.ts`, `src/utils/quota/constants.ts`.

## Vấn đề

W1/W2 gắn nút reset vào endpoint backend llmhub (`/reset-quota` xoá cooldown local,
`/reset-usage` xoá counter thống kê). Upstream làm hoàn toàn khác:

- Reset là **consume 1 "manual reset credit"** của tài khoản ChatGPT, gọi thẳng
  provider qua proxy `POST /v0/management/api-call`. Không có endpoint backend riêng.
- Chỉ **Codex** có tính năng này. Provider khác upstream không có nút reset.
- Nút chỉ hiện khi `rateLimitResetCreditsAvailableCount > 0`.
- Có hiển thị **số credit còn lại** + **danh sách hạn dùng từng credit** (GMT+8).

## Quyết định đã chốt với ní

1. Thay nút reset hiện tại bằng flow upstream. Provider khác: bỏ nút reset khỏi panel.
   Endpoint Go `/reset-quota` giữ nguyên, panel không gọi nữa.
2. Gỡ nút "reset usage statistics" (W2) khỏi panel. Handler Go `/reset-usage` + test giữ lại.

## Hợp đồng API upstream (đã đọc code, không đoán)

```
GET  https://chatgpt.com/backend-api/wham/usage
     → rate_limit_reset_credits.available_count
GET  https://chatgpt.com/backend-api/wham/rate-limit-reset-credits
     → { available_count, credits: [{ id, status, granted_at, expires_at }] }
POST https://chatgpt.com/backend-api/wham/rate-limit-reset-credits/consume
     body { redeem_request_id: <uuid v4> }
```

Tất cả đi qua `apiCallApi.request({ authIndex, method, url, header })`.
Header: `CODEX_REQUEST_HEADERS` + `Chatgpt-Account-Id` nếu resolve được.
Số credit lấy theo thứ tự: `credits API available_count` → `credits.length` → `usage payload available_count`.

---

## Bước thực hiện

### B1 — Types + constants
- [x] `web/src/utils/quota/constants.ts`: thêm `CODEX_RATE_LIMIT_RESET_CREDITS_URL`,
      `CODEX_RATE_LIMIT_RESET_CREDITS_CONSUME_URL`
- [x] `web/src/types/quota.ts`: thêm `CodexRateLimitResetCredits`,
      `CodexRateLimitResetCredit`; mở rộng `CodexUsagePayload` (`rate_limit_reset_credits`)
      và `CodexQuotaState` (`rateLimitResetCreditsAvailableCount`,
      `rateLimitResetCredits`, `rateLimitResetCreditsError`)
- verify: `bun run type-check`

### B2 — Parser + formatter
- [x] `web/src/utils/quota/parsers.ts`: `normalizeCodexResetCreditsPayload(payload)`
      → `{ availableCount, credits, invalidPayload }`, chấp nhận cả snake_case và camelCase
- [x] `web/src/utils/quota/formatters.ts`: `formatShanghaiDateTime(value)` (Asia/Shanghai)
- [x] export qua `web/src/utils/quota/index.ts`
- verify: unit test nếu B7 dựng được infra test; nếu không thì `bun run type-check`

### B3 — Quota config
- [x] `quotaConfigs.ts`: `QuotaConfig` thêm 2 field optional
      `resetQuota?`, `canResetQuota?`
- [x] `fetchCodexQuota` fetch thêm reset credits, trả về count + list + error string
      (lỗi credits **không** làm hỏng cả quota card — upstream nuốt lỗi vào field riêng)
- [x] `resetCodexQuota` = consume credit → refetch quota
- [x] `renderCodexItems`: thêm dòng "Manual resets: N" vào cụm plan, thêm block
      list hạn dùng từng credit, và dòng lỗi nếu fetch credits fail
- [x] `CODEX_CONFIG`: wire `resetQuota` + `canResetQuota: (q) => (q.rateLimitResetCreditsAvailableCount ?? 0) > 0`
      và mở rộng `buildLoadingState`/`buildSuccessState`/`buildErrorState`
- [x] `quotaStyles.ts`: thêm class Tailwind cho `codexResetCredits*` (upstream dùng SCSS module, llmhub dùng styleMap)
- verify: `bun run type-check`

### B4 — QuotaCard đổi contract nút reset
- [x] Bỏ props `canResetQuota` / `resettingQuota` / `onResetQuota`,
      thay bằng `resetQuotaAction?: ReactNode` (đúng shape upstream)
- verify: `bun run type-check` bắt hết call site

### B5 — Gỡ đường reset cũ ở 3 section
- [x] `QuotaSection.tsx`, `AllQuotaSection.tsx`: bỏ `quotaApi.resetQuota`,
      bỏ `resolveResetErrorMessageKey`, dựng `resetQuotaAction` từ config, gate bằng `canResetQuota`
- [x] `features/authFiles/components/AuthFileQuotaSection.tsx`: thêm cùng logic
      (upstream có nút này ở cả trang Auth Files)
- [x] `web/src/services/api/quota.ts`: bỏ `resetQuota` (W1 thêm, giờ thành dead code)
- verify: `bun run type-check && bun run lint`

### B6 — Gỡ UI reset usage statistics (W2)
- [x] `features/authFiles/components/AuthFileCard.tsx`: bỏ nút + props `usageResetting`/`onResetUsage`
- [x] `features/authFiles/hooks/useAuthFilesData.ts`: bỏ state + `handleResetUsage` + `resolveResetUsageErrorMessageKey`
- [x] `pages/AuthFilesPage.tsx`: bỏ props truyền xuống
- [x] `web/src/services/api/usage.ts`: bỏ `resetUsage` + type `ResetUsageResponse`
- [x] Go giữ nguyên: `internal/api/handlers/management/usage.go`, route, `Manager.ResetUsage`, test
- verify: `bun run type-check && bun run lint`

### B7 — i18n
- [x] Thêm vào `codex_quota` (cả `en.json` và `vi.json`): `reset_credits_label`,
      `reset_credits_expiry_label`, `reset_credit_number`, `reset_credits_expiry_failed`,
      `reset_credits_invalid_payload`, `reset_button`, `reset_confirm_title`,
      `reset_confirm_message`, `reset_confirm_button`, `reset_success`, `reset_failed`
- [x] Xoá key không còn dùng: `quota_management.reset_*` (W1), `auth_files.reset_usage_*` (W2)
- verify: grep không còn key mồ côi; key nào cũng có ở **cả hai** locale

### B8 — Verify tổng
- [x] `cd web && bun run type-check && bun run lint`
- [x] `bun run test:run` nếu script tồn tại
- [x] `make build-web && make build`
- [x] `go build ./...` (không đổi Go nhưng chạy cho chắc)

---

## Blocker vẫn còn

**Không smoke test được trên browser thật**: server chỉ chạy Postgres-only, máy không có
Postgres local, Docker daemon down. Nghĩa là **không xác minh được** flow consume credit
thật với tài khoản ChatGPT. Cái verify được: type-check, lint, build, và logic gate
(count = 0 → không render nút). Sẽ ghi rõ giới hạn này khi báo cáo, không claim đã test end-to-end.

## Blast radius

Chỉ frontend. Không đổi Go, không đổi API contract. Rủi ro cao nhất là B4 đổi props
`QuotaCard` — type-check sẽ bắt hết call site nên không âm thầm hỏng.

## Kết quả — 2026-07-26

- `bun run type-check`: pass
- `bun run test:run`: 5 files, 64 tests pass
- `bun run lint`: 0 errors, 8 pre-existing warnings
- locale parity: 35 `codex_quota` keys match giữa en/vi
- `git diff --check`: pass
- `make build-web`: pass (`dist/index.html`, 1,923.52 kB)
- `make build`: pass, Go binary compiled
- Browser/provider E2E chưa chạy vì môi trường không có Postgres/Docker và cần tài khoản
  ChatGPT thật có reset credit. Không claim E2E.
