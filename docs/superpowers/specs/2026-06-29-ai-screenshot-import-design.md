# AI Screenshot Import (Gemini) — Design & Implementation Plan

**Date:** 2026-06-29
**Status:** Approved design — ready for implementation
**Scope:** Phase 5 of the Amplop v2 backend: AI-assisted import of transactions from
e-wallet / m-banking **screenshots** (GoPay, Livin' by Mandiri, …). Adds two
endpoints — `POST /scan` (extract candidates, no insert) and `POST /expenses:batch`
(bulk insert the reviewed result). This document **supersedes §9** ("Phase 2 deferred —
AI screenshot import") of `2026-06-15-amplop-v2-backend-design.md`, which was an
interface sketch only.

> **Decision change vs. the v2 spec:** §9 named multimodal **Claude**. This phase uses
> the **Gemini API** instead (user direction). Everything else in §9 (review-before-insert,
> income excluded, low-confidence flagged, `POST /expenses:batch`) is carried forward and
> made concrete here.

---

## 1. Context

Phases 0–4 of the v2 backend are complete: the envelope engine, effective-dated
config/subscriptions, expense CRUD, the `GET /month` dashboard, the single routed
`Expense` Cloud Function, migrations, and deploy. This phase adds the **AI import**
the prototype `scan-flow.jsx` designed:

1. User picks one or more screenshots of transaction history from an e-wallet / bank app.
2. Backend compresses each image and sends them, with a prompt, to Gemini.
3. Gemini returns the transactions it read (per image), already classified.
4. Backend flags income, internal transfers, and likely duplicates; returns the
   candidates **for review** — nothing is inserted yet.
5. User reviews/edits in the app, then commits the kept rows via a **bulk insert**.

The import never writes directly: the prototype's `ReviewView` lets the user toggle,
re-categorise, edit amounts/dates, and only then "Catat" (record). The backend mirrors
that: `POST /scan` is read-only extraction; `POST /expenses:batch` is the write.

### Source of truth (prototype)
`proto/scan-flow.jsx` — `SCAN_GALLERY`, `SCAN_RESULT`, `ScanFlow` (pick → process →
review). Each extracted row is `{ name, amount, date, time, dir: out|in, cat, conf }`;
income (`dir:'in'`) is auto-skipped; low confidence (`conf < 0.8`) is flagged "AI ragu";
commit emits `{ date, time, amount, cat, note }`. This design extends that shape with
**Langganan/subscription matching**, a third **`transfer`** direction, and
**duplicate** flags.

---

## 2. Decisions (locked)

| # | Decision | Choice |
|---|----------|--------|
| G1 | How to call Gemini? | **Hand-rolled REST** to `generateContent`, structured output via `responseSchema`. No SDK dependency (matches the repo's stdlib-first rule). |
| G2 | Image preprocessing? | **Downscale long edge + JPEG re-encode.** Gemini bills images by **resolution (tiles), not file bytes** — downscaling dimensions is what reduces tokens; the JPEG re-encode reduces upload size. Adds `golang.org/x/image` for quality scaling. |
| G3 | Duplicate handling? | **Flag at scan, skip at commit**, keyed on `date + time + amount` (§9). Scan marks already-recorded rows so the user sees them; the batch commit re-checks and skips them idempotently. |
| G4 | Batch failure semantics? | **All-or-nothing** for the real inserts (one DB transaction). Exact duplicates are **skipped** (re-inserting an identical existing row is a no-op by intent); any **validation / once-per-month conflict aborts the whole batch** — nothing inserted (§8). |
| G5 | Income & internal transfers? | `direction` is a **three-way enum** `out | in | transfer`. Only `out` is committable. `in` (salary/refund/cashback) and `transfer` (e-wallet top-ups / movements between the user's own accounts) are returned-but-flagged and never committed (§7). |
| G6 | Store the source images? | **No.** Discarded after the Gemini call (never persisted). |
| G7 | Subscription/Langganan detection? | The current-month resolved subscription **names** are injected into the prompt; Gemini matches a row to a known subscription; the **server** maps the matched name → `subscription_id` (never trusts an id from the model). Unmatchable "Langganan" is normalised to `Lainnya` so every `out` candidate is directly committable (§7.3). |
| G8 | Endpoints & deployment? | Two new routes on the existing `Expense` function: `POST /scan` (multipart) and `POST /expenses:batch` (JSON). No new function. |

### Carried over (assumptions)
- Go + GCP Functions Framework; CockroachDB via `sqlx`; single user, **no auth**, `CORS:*`.
- Timezone **Asia/Jakarta**; `TIME` env var pins "now" / the current month for deterministic tests.
- Money is integer Rupiah (`INT8`); the API returns integers, the client formats.
- Gemini API key and tunables come from the **environment**; nothing secret is committed.

---

## 3. Scope

### In scope
- `POST /scan`: multipart image upload → compress → Gemini structured extraction →
  classify (out/in/transfer) → match subscriptions → flag duplicates → grouped JSON.
- `POST /expenses:batch`: validate + dedup-skip + all-or-nothing bulk insert of reviewed rows.
- New platform pieces: a Gemini REST client, an image-compression utility, an `Upstream`
  (502) typed error, and Gemini/scan config.
- Tests: image util, Gemini client (httptest), scan service (faked Gemini + repos),
  batch service + repo (incl. `devdb` integration), and both handlers.

### Non-goals
- Storing/auditing source images; per-app bespoke OCR parsers (one generic vision prompt).
- Receipt/photo (non-list) parsing — the flow targets **transaction-list** screenshots.
- Auto-commit without review; multi-user; changing the `GET /month` payload.
- Detecting arbitrary self-transfers between bank accounts (only **e-wallet top-ups** are
  reliably classified as `transfer`; ordinary person-to-person transfers stay `out`).

---

## 4. Architecture

### 4.1 Package layout (additions only)

```
internal/
  platform/
    gemini/                       NEW — generateContent REST client
      client.go                     GenerateJSON(ctx, model, prompt, images, schema) -> raw JSON bytes; base URL injectable for tests
      client_test.go                httptest server: request shape + response parse + non-200 -> Upstream
    imageutil/                    NEW — image preprocessing
      compress.go                   Compress(r) -> (jpegBytes, w, h); downscale long edge, re-encode JPEG
      compress_test.go
    apierr/
      apierr.go                     + KindUpstream + Upstream(msg) -> 502
    httpx/
      httpx.go                      + map KindUpstream -> 502 in the error->status switch
    config/
      config.go                     + Gemini{APIKey, Model, MaxImageDim, JPEGQuality, MaxImages, MaxBytes}; LoadGemini()
  scan/                           NEW — the scan domain
    model.go                        Candidate, ImageResult, ScanResponse
    prompt.go                       prompt text + responseSchema (categories + subscription names injected)
    service.go                      Compress -> Gemini -> parse -> classify/normalise -> match subs -> flag dupes
    handler.go                      multipart parse -> service -> JSON
    service_test.go  handler_test.go
  expense/
    repo.go                         + ByDateRange(ctx, from, to) ; + CreateBatch(ctx, []Expense) (one tx)
    service.go                      + CreateBatch(ctx, []WriteRequest) (BatchResult)
    handler.go                      + CreateBatch
    *_test.go                       + batch cases
function.go                       + routes POST /scan, POST /expenses:batch; wire scan handler
.env.example                      + GEMINI_* / SCAN_* vars (no real key)
```

**Layering is unchanged:** `handler → service → repo`; the Gemini client and image util are
infra under `platform/` (like `database`); `scan/service.go` orchestrates them with the
expense repo (dedup lookup) and the subscription repo (`Resolve`, for the prompt + name→id
mapping). The envelope engine is untouched.

### 4.2 Endpoints

| Method | Path | Body | Returns |
|--------|------|------|---------|
| `POST` | `/scan` | `multipart/form-data`, repeated file field `images` | grouped candidates (no insert) |
| `POST` | `/expenses:batch` | JSON `{ items: [WriteRequest…] }` | insert summary |

Both parse cleanly in the existing router: `/scan` is one literal segment; `expenses:batch`
is one literal segment (no `/`), so it never collides with `POST /expenses` (create) or
`PUT/DELETE /expenses/{id}`.

---

## 5. `POST /scan` — flow

1. **Parse multipart.** Read every `images` file part (≤ `SCAN_MAX_IMAGES`, total ≤
   `SCAN_MAX_BYTES`; otherwise `400`). Reject when `GEMINI_API_KEY` is unset (`503`).
2. **Compress** each image (`imageutil.Compress`) → downscaled JPEG. Preserve input order.
3. **Resolve subscriptions** for the **current** Asia/Jakarta month (`subRepo.Resolve`)
   → names injected into the prompt; the `(name → id)` map is kept server-side.
4. **One Gemini call** (`platform/gemini`) with all images + the prompt + `responseSchema`.
5. **Parse** the structured JSON; **server assigns `index`** by input position (the model
   is told to keep image order — we never trust a model-supplied index).
6. **Normalise & classify** each row (§7): coerce category to the allowed set; map a matched
   subscription name → `subscription_id`; downgrade unmatchable `Langganan` → `Lainnya`;
   keep `direction` ∈ {out,in,transfer}.
7. **Flag duplicates** (§9): load existing expenses across the candidates' date span
   (`expense.Repo.ByDateRange`); mark any candidate whose `(date,time,amount)` matches an
   existing row (`in_database`) or an earlier kept candidate in this batch (`in_batch`).
8. Return the grouped response. **Nothing is inserted.**

### 5.1 Response shape

```jsonc
{
  "images": [
    {
      "index": 0,
      "source": "GoPay",          // model-detected app label; "" when unrecognised
      "recognized": true,
      "items": [
        { "name": "GoFood · Mie Gacoan", "amount": 28000, "date": "2026-06-16", "time": "19:20",
          "direction": "out", "category": "Makan",
          "subscription_id": null, "subscription_name": null,
          "confidence": 0.96, "duplicate": false, "duplicate_reason": null },

        { "name": "Netflix", "amount": 186000, "date": "2026-06-05", "time": "09:00",
          "direction": "out", "category": "Langganan",
          "subscription_id": "7b1c…", "subscription_name": "Netflix",
          "confidence": 0.93, "duplicate": true, "duplicate_reason": "in_database" },

        { "name": "Top Up GoPay", "amount": 200000, "date": "2026-06-15", "time": "08:00",
          "direction": "transfer", "category": "Lainnya",       // ← e-wallet top-up (GoPay side), skipped (§7)
          "subscription_id": null, "subscription_name": null,
          "confidence": 0.99, "duplicate": false, "duplicate_reason": null }
      ]
    },
    {
      "index": 1, "source": "Livin' by Mandiri", "recognized": true,
      "items": [
        { "name": "Top Up GoPay", "amount": 200000, "date": "2026-06-15", "time": "07:59",
          "direction": "transfer", "category": "Lainnya",         // ← internal transfer, skipped (§7)
          "subscription_id": null, "subscription_name": null,
          "confidence": 0.97, "duplicate": false, "duplicate_reason": null },
        { "name": "Gaji Bulanan", "amount": 6500000, "date": "2026-06-15", "time": "00:05",
          "direction": "in", "category": "Lainnya",
          "subscription_id": null, "subscription_name": null,
          "confidence": 0.99, "duplicate": false, "duplicate_reason": null }
      ]
    },
    { "index": 2, "source": "", "recognized": false, "items": [] }   // unreadable / not a tx list
  ]
}
```

- `direction:"in"` and `direction:"transfer"` rows are **returned but flagged**; the client
  shows them as skipped ("pemasukan dilewati" / "transfer dilewati") and never sends them to commit.
- `recognized:false` (empty `items`) is how the client counts "N screenshot tak dikenali".
- `confidence` is a pass-through 0..1; the client flags `< 0.8` as "AI ragu".
- `duplicate_reason ∈ { "in_database", "in_batch", null }`.

---

## 6. Gemini client (`internal/platform/gemini`)

A thin REST wrapper — no SDK.

- **Request:** `POST {baseURL}/v1beta/models/{model}:generateContent?key={APIKey}`
  (`baseURL` defaults to `https://generativelanguage.googleapis.com`, overridable for tests):

  ```jsonc
  {
    "contents": [{ "role": "user", "parts": [
      { "inline_data": { "mime_type": "image/jpeg", "data": "<base64>" } },
      { "inline_data": { "mime_type": "image/jpeg", "data": "<base64>" } },
      { "text": "<prompt>" }
    ]}],
    "generationConfig": {
      "temperature": 0,
      "responseMimeType": "application/json",
      "responseSchema": { /* §6.1 */ }
    }
  }
  ```
- **Response:** read `candidates[0].content.parts[0].text` (the structured JSON string) and
  return it as raw bytes for the scan service to unmarshal. A non-`STOP` finish reason
  (e.g. `SAFETY`, `MAX_TOKENS`) or a non-2xx HTTP status → `apierr.Upstream` (**502**).
- **Surface:** `GenerateJSON(ctx, model string, prompt string, images []Image, schema any) ([]byte, error)`
  where `Image{MIME string; Data []byte}`. The client base-64-encodes inline. `http.Client`
  with a sane timeout; the API key is read from config, never logged.

### 6.1 `responseSchema` (forces valid JSON)

```jsonc
{ "type": "object", "required": ["images"], "properties": {
  "images": { "type": "array", "items": {
    "type": "object", "required": ["source", "recognized", "items"], "properties": {
      "source":     { "type": "string" },
      "recognized": { "type": "boolean" },
      "items": { "type": "array", "items": {
        "type": "object",
        "required": ["name", "amount", "date", "direction", "category", "confidence"],
        "properties": {
          "name":              { "type": "string" },
          "amount":            { "type": "integer" },              // IDR parsed to integer Rupiah
          "date":              { "type": "string" },               // YYYY-MM-DD
          "time":              { "type": "string" },               // HH:MM ("" if absent)
          "direction":         { "type": "string", "enum": ["out","in","transfer"] },
          "category":          { "type": "string", "enum": ["Makan","Belanja","Jajan","Cash","Lainnya","Langganan"] },
          "subscription_name": { "type": "string" },               // "" unless matched
          "confidence":        { "type": "number" }
        }
      }}
    }
  }}
}}
```
`index`, `subscription_id`, `duplicate`, `duplicate_reason` are **server-assigned**, never
requested from the model.

---

## 7. Classification & normalisation (scan service)

For each model row the service produces a `Candidate`:

### 7.1 Direction (`out | in | transfer`)
- **`out`** — a real expense; committable; default kept in review.
- **`in`** — genuine income (salary, refund, cashback, incoming payment). Returned, flagged,
  never committed.
- **`transfer`** — money moving **between the user's own accounts**: e-wallet top-ups on
  either side. Returned, flagged, never committed.

### 7.2 The e-wallet top-up rule (why `transfer` exists)
The user funds **GoPay by topping up from Livin'**. That single top-up appears **twice** in
screenshots: as an **outgoing** "Top Up GoPay" in the Livin' history and as an **incoming**
"Top Up GoPay" in the GoPay history. The actual spending is itemised *inside* GoPay, so
recording the bank-side top-up as an expense would **double-count** it against those itemised
purchases. The prompt therefore classifies e-wallet top-ups (GoPay primarily; also
DANA/OVO/ShopeePay/LinkAja) and movements between the user's own accounts as `transfer` on
**either** side, and the service excludes them from commit. An ordinary person-to-person
transfer (e.g. "Transfer ke Budi") stays `out`.

### 7.3 Category & subscription
- `category` is coerced into the six allowed values; anything unexpected → `Lainnya`.
- If the row is `Langganan` and its `subscription_name` matches a resolved subscription
  (case-insensitive), set `subscription_id` + `subscription_name`. If it is `Langganan` but
  **no** known subscription matches, **downgrade** to `Lainnya` (and lower confidence) so the
  candidate stays directly committable; the user can re-categorise in review. The server
  never accepts a `subscription_id` from the model.

---

## 8. `POST /expenses:batch` — bulk insert

Request (same item shape as `POST /expenses`, spec §7.2):
```jsonc
{ "items": [
  { "date": "2026-06-16", "time": "19:20", "amount": 28000, "category": "Makan",
    "subscription_id": null, "note": "GoFood · Mie Gacoan" }
] }
```

### 8.1 Semantics (reconciling G3 "skip dupes" with G4 "all-or-nothing")
Processed in `expense.Service.CreateBatch`:

1. Reject an empty `items` (`400`).
2. Load existing expenses over the batch's date span (`ByDateRange`) → build the
   `(date,time,amount)` key set.
3. Walk items in order, tracking keys/`(subscription_id,year,month)` **seen within the batch**:
   - **Validate** (category, amount > 0, date, `time`, `Langganan ⇔ subscription_id`,
     referenced subscription exists). Any validation failure → **abort** the whole batch
     (`400`, nothing inserted), naming the offending item.
   - **Confident duplicate** (`date+time+amount` already in the existing set **or** seen
     earlier in this batch) → **skip** (counts as `skipped_duplicate`); does **not** abort.
     Rows without a time are never auto-skipped — they are inserted (§9).
   - **Once-per-month Langganan** for a *non-duplicate* row whose subscription is already paid
     this month (DB pre-check via `ExistsForSubscriptionMonth`, **or** a second Langganan for
     the same `(subscription_id, month)` within the batch) → **abort** (`409`, nothing inserted).
   - Otherwise add to the insert list; record its key and sub-month.
4. Insert the list in **one transaction** (`repo.CreateBatch`, mirroring
   `subscription.CreateWithVersion`). Any DB error (incl. the once-per-month unique index as a
   backstop → `apierr.Conflict`) rolls back the **entire** batch.

So: identical re-imports are idempotently skipped; a genuinely bad or conflicting row rejects
the whole import without partial writes.

### 8.2 Responses
- **Success** `200`:
  ```jsonc
  { "inserted": 5, "skipped_duplicate": 1, "created": [ /* full expense rows, §7.2 shape */ ] }
  ```
  (`created` lets the client refresh without a full `GET /month` refetch.)
- **Rejection**: `400` (validation) or `409` (once-per-month) with `{"error":"item 3: …"}` —
  **nothing inserted**.

---

## 9. Duplicate detection (the match key)

A candidate is a **confident** duplicate when `amount`, `occurred_date`, **and** `time`
(to the minute, both present) are equal. When either side lacks a time, a `date + amount`
match is only a **possible** duplicate. Transaction screenshots almost always carry a
timestamp, so the confident key reliably catches the *same* transaction across overlapping
screenshots and against the DB.

- **At scan (informational):** both confident and possible matches set `duplicate=true`
  (+ `duplicate_reason`); the client pre-unchecks them so the user decides.
- **At commit (enforced):** only **confident** (`date+time+amount`) matches are skipped. A
  possible (time-less) match is **not** auto-skipped — it is inserted — so two genuinely
  identical expenses at different or unknown times are never silently dropped.

`expense.Repo.ByDateRange(ctx, from, to)` backs both (range over `min..max` candidate dates),
reusing the `ForMonth` query pattern with explicit bounds.

---

## 10. Config, limits, errors

**Environment (new):**

| Var | Default | Meaning |
|-----|---------|---------|
| `GEMINI_API_KEY` | — (required for `/scan`) | API key; absent → `/scan` returns `503`, other endpoints unaffected. Never logged/committed. |
| `GEMINI_MODEL` | a Gemini **Flash** model (e.g. `gemini-2.5-flash`) | cheap multimodal model id |
| `GEMINI_MAX_IMAGE_DIM` | `1600` | long-edge px cap before upload (keeps list text legible while cutting tiles/tokens) |
| `GEMINI_JPEG_QUALITY` | `80` | re-encode quality |
| `SCAN_MAX_IMAGES` | `10` | max images per `/scan` |
| `SCAN_MAX_BYTES` | ~`20 MiB` | max total upload bytes |

**Errors:** Gemini failure / unparseable output / non-`STOP` finish → **502**
(`apierr.Upstream`, new `KindUpstream` mapped in `httpx`). Bad multipart, too many/too-large
images, empty batch → **400**. Once-per-month conflict → **409**. Unknown subscription id in a
batch item → **400**. The body stays `{"error":"message"}`.

---

## 11. Testing strategy

- **`imageutil`** — oversized image downscales below the cap and re-encodes; small image is
  preserved; output decodes; non-image input errors.
- **`gemini`** — `httptest` server asserts the request (model in URL, inline image parts,
  `responseSchema`, key in query) and parses a canned response; non-200 and non-`STOP` → `502`.
- **`scan` service** — faked Gemini client + fake expense repo + fake subscription resolver:
  income → `in`; **Livin "Top Up GoPay" → `transfer`** and the GoPay-side top-up stays excluded;
  subscription matched by name → `subscription_id` set; unmatched `Langganan` → `Lainnya`;
  duplicates flagged (`in_database` + `in_batch`); unreadable image → `recognized:false`;
  server assigns `index` by position.
- **`expense.CreateBatch`** — all-valid inserts atomically; one invalid → nothing inserted;
  exact duplicate skipped (DB + in-batch); once-per-month conflict (DB + in-batch) → reject,
  nothing inserted; `devdb` integration test exercising the real transaction + unique-index backstop.
- **Handlers** — `/scan` multipart happy path; missing images → 400; oversize/too-many → 400;
  no API key → 503; upstream failure → 502. `/expenses:batch` happy path; validation 400;
  conflict 409. `TIME` pins "now" / the current month.
- CI green per phase.

---

## 12. Implementation phases

Each phase ends green (`go build ./...`, `go test ./...`). TDD for the pure-ish pieces
(image util, classification/dedup, batch semantics). One PR per phase.

### Phase 5.1 — Platform plumbing
- [ ] `platform/imageutil`: `Compress` (decode jpeg/png → downscale long edge with
  `golang.org/x/image/draw` → JPEG re-encode) + tests; add `golang.org/x/image`.
- [ ] `platform/gemini`: `GenerateJSON` REST client (injectable base URL, structured output,
  key from config) + httptest tests.
- [ ] `apierr.Upstream`/`KindUpstream` (502) + `httpx` mapping; `config.LoadGemini` + `.env.example`.

### Phase 5.2 — Bulk insert (`POST /expenses:batch`)
- [ ] `expense.Repo.ByDateRange` + `expense.Repo.CreateBatch` (single `sqlx` tx).
- [ ] `expense.Service.CreateBatch` (validate, dedup-skip, once-per-month, all-or-nothing) + `BatchResult`.
- [ ] `expense.Handler.CreateBatch`; route `POST /expenses:batch`; unit tests + `devdb` integration.

### Phase 5.3 — Extraction (`POST /scan`)
- [ ] `scan` model + `prompt.go` (prompt text + `responseSchema`, categories + subscription names).
- [ ] `scan.Service`: compress → Gemini → parse → classify (out/in/transfer) → normalise category →
  match subscription name→id → flag duplicates.
- [ ] `scan.Handler` (multipart parse, limits) + route `POST /scan`; tests (faked Gemini + repos).

### Phase 5.4 — Wiring & docs
- [ ] `function.go` wiring (scan handler needs the Gemini client, expense repo, `subRepo.Resolve`, clock).
- [ ] Update `README` / `.env.example`; mark §9 of the v2 spec as superseded by this doc; local-run notes.
- [ ] Verify `go build ./...` + `go test ./...` green; manual `/scan` smoke against a real screenshot.

---

## 13. Open items (with defaults)
1. **Image formats** — decode JPEG/PNG to start; WebP/HEIC optional later
   (`golang.org/x/image/webp` decodes WebP if needed). *(Default: JPEG/PNG.)*
2. **`GEMINI_MAX_IMAGE_DIM`** — `1600` is a starting point; tune against real GoPay/Livin'
   screenshots if small text mis-reads. *(Default: 1600.)*
3. **Subscription matching** — current-month resolved set, case-insensitive exact/contains
   match. Fuzzy matching can come later if names drift. *(Default: current month.)*
4. **`transfer` scope** — e-wallet top-ups only; generic self-transfers between bank accounts
   stay `out`. *(Default: e-wallet top-ups.)*
5. **Batch `created` payload** — return the inserted rows so the client can merge without a
   refetch. *(Default: yes.)*

---

## 14. Prototype → backend map
| Prototype (`scan-flow.jsx`) | Backend home |
|---|---|
| `GalleryPicker` (pick screenshots) | client-only; uploads to `POST /scan` |
| `ProcessingView` ("Mengunggah… / Membaca… / Menyaring…") | `scan.Service` (compress → Gemini → classify) |
| `SCAN_RESULT` `{ source, items[] }` | `POST /scan` response `images[]` (§5.1) |
| `it.dir` (`out`/`in`) | `direction` (`out`/`in`/**`transfer`**) (§7) |
| income "dilewati" | `direction:"in"`; **e-wallet top-ups** → `direction:"transfer"` |
| `ReviewView` edit cat/amount/date, `keep` | client review of the candidates |
| `commit()` → `{ date, time, amount, cat, note }` | `POST /expenses:batch` (§8) |
| (new) duplicate awareness | `duplicate` / `duplicate_reason` (§9) |
| (new) Langganan from screenshot | `subscription_id` via name match (§7.3) |
