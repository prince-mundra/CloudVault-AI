# CloudVault AI — Frontend UI/UX Design & Architecture Specification

**Tagline:** *Intelligent Cloud Storage. Powered by AI.*
**Target Stack:** React, Vite, TypeScript, Tailwind CSS, shadcn/ui, Lucide React, React Router, Recharts, Framer Motion
**Consumed Services:** Go + Gin Core API (Auth, Files, Folders, Sharing, Analytics) & Python + FastAPI AI Service (PostgreSQL + `pgvector`, Ollama `llama3.2:3b` & `nomic-embed-text`)

---

## Part 1: Complete Product Design Specification

### 1.1 Product & Frontend Architecture Overview

CloudVault AI combines the spatial predictability of traditional cloud storage with verifiable, source-grounded AI document intelligence. The frontend is a single-page React + TypeScript application that consumes two existing backend services without introducing any additional backend layer:

| Service | Base Prefix (via Vite Proxy / API Gateway) | Responsibilities |
| :--- | :--- | :--- |
| **Go + Gin Core API** | `/api/v1/*` | JWT/session authentication, user profile & settings, folder hierarchy, multipart file upload/download, storage quotas, share links & RBAC permissions, audit/activity logs, storage analytics. |
| **Python + FastAPI AI Service** | `/ai/v1/*` | Async PDF/text extraction & chunking, `nomic-embed-text` vector indexing in `pgvector`, `llama3.2:3b` document summarization & key-point extraction, RAG chat (SSE streaming with page-level citations), semantic vector search. |

> **Note on Sample Data:** All filenames, metrics, emails, and document excerpts in this specification are **demonstration data** for UI scaffolding, Storybook stories, and initial empty/mock states prior to live API hydration.

---

### 1.2 Design System & Tokens

#### Color Palette (Strict Dark Minimalist SaaS)
| Token | Hex | Usage |
| :--- | :--- | :--- |
| `--bg-canvas` | `#101114` | Primary application & landing page background. |
| `--bg-surface` | `#15171D` | Left sidebar, top header, secondary side panels, table headers. |
| `--bg-card` | `#191B22` | Elevated cards, modals, dropdowns, chat bubbles, input surfaces. |
| `--bg-card-hover` | `#1F222B` | Hover state for interactive cards and table rows. |
| `--border-hairline` | `#242731` | Default `1px` borders, dividers, and input outlines. |
| `--border-hover` | `#323644` | Focused/hovered card and input borders. |
| `--accent-primary` | `#7667F5` | Electric Indigo — primary CTAs, active nav indicators, selected checkboxes, focus rings. |
| `--accent-primary-hover` | `#8679F7` | Hover state for primary buttons. |
| `--accent-secondary` | `#36C8D8` | Cyan — AI intelligence badges, semantic match pills, encryption status, chart series. |
| `--text-primary` | `#F3F4F6` | Headings, primary body copy, active labels. |
| `--text-muted` | `#9296A3` | Metadata, timestamps, secondary descriptions, inactive nav items. |
| `--status-success` | `#22C55E` | Indexed/completed status, positive growth deltas. |
| `--status-warning` | `#F59E0B` | Expiring links, approaching storage quota (>85%). |
| `--status-danger` | `#EF4444` | Revoke access, delete file, failed indexing, form validation errors. |

#### Typography (Inter / Geist)
* **Font Family:** `'Inter', 'Geist', -apple-system, BlinkMacSystemFont, sans-serif` (with `font-feature-settings: "cv02", "cv03", "cv04", "cv11"`).
* **Tabular Numerals:** Apply `tabular-nums` (`font-variant-numeric: tabular-nums`) on all storage sizes, percentages, timestamps, and chart axes.
* **Type Scale:**
  * Hero Display (Landing): `clamp(2.5rem, 5vw, 4rem)`, weight `700`, tracking `-0.03em`, line-height `1.1`.
  * Page Title (`h1`): `24px` (`1.5rem`), weight `600`, tracking `-0.02em`.
  * Section / Card Title (`h2`): `16px` (`1rem`), weight `600`, tracking `-0.01em`.
  * Body Default: `14px` (`0.875rem`), weight `400`, line-height `1.5`.
  * Metadata / Table Header: `12px` (`0.75rem`), weight `500`, color `#9296A3`.

#### Spacing, Radii & Motion
* **Spacing Scale:** `4px` base grid (`4, 8, 12, 16, 20, 24, 32, 48, 64`). Card internal padding: `20px` (`p-5`) or `24px` (`p-6`).
* **Border Radius:** `12px` (`rounded-xl`) for cards/modals, `8px` (`rounded-lg`) for buttons/inputs/nav pills, `9999px` (`rounded-full`) for status badges and avatars.
* **Shadows:** Restrained dark elevation (`0 8px 24px rgba(0, 0, 0, 0.35)`) reserved strictly for floating dropdowns, context menus, and modals.
* **Motion (Framer Motion):**
  * Page/tab transitions: `opacity: 0 -> 1`, `y: 6 -> 0`, duration `180ms`, ease `[0.16, 1, 0.3, 1]`.
  * Hover transitions: `150ms ease-out` on border color and background tint.
  * Respect `prefers-reduced-motion: reduce`.

---

### 1.3 Global App Shell & Navigation Architecture

All authenticated screens (Screens 3–10) share a persistent `<AppShell />` layout wrapper:

1. **Fixed Left Sidebar (`240px` desktop, `72px` collapsed tablet optional, slide-over `Sheet` on mobile `<768px`)**:
   * **Top Brand Header:** CloudVault AI geometric vault-shield icon + wordmark (`#F3F4F6` + `#36C8D8` "AI").
   * **Primary Nav Items:**
     * `Overview` (`/dashboard`) — `LayoutDashboard` icon
     * `My Files` (`/files`) — `Folder` icon
     * `Shared Files` (`/shared`) — `Users` icon
     * `AI Assistant` (`/assistant`) — `Sparkles` icon
     * `Smart Search` (`/search`) — `Search` icon
     * `Analytics` (`/analytics`) — `BarChart3` icon
     * `Settings` (`/settings`) — `Settings` icon
   * **Active State:** `#191B22` fill, `1px` `#242731` border, `3px` left Electric Indigo (`#7667F5`) indicator or subtle pill highlight, `#F3F4F6` text.
   * **Bottom Storage Meter:** Compact card showing `64.2 GB / 100 GB` (`64%`) with thin `#7667F5` → `#36C8D8` gradient progress bar. Clicking navigates to `/analytics`.
2. **Sticky Top Header (`64px` height, `#15171D` / `#101114` translucent backdrop)**:
   * **Left (Mobile only):** Hamburger button (`Menu` icon) opening sidebar drawer.
   * **Center/Left:** Global Command Input (`Search files, folders, or ask AI...` + `⌘K` badge). Clicking or pressing `⌘K` / `Ctrl+K` opens the global **Command Palette Modal** (`shadcn/ui Command`).
   * **Right:**
     * Indexing Queue Indicator (visible only when files are actively embedding via Python FastAPI, showing a subtle Cyan pulse + count).
     * Notification Bell (`Bell` icon with unread dot) -> opens popover with recent sharing & AI indexing events.
     * User Avatar Dropdown (`Alex Morgan`, `alex@cloudvault.io`) -> `Settings`, `Storage & Plan`, `Sign Out`.

---

### 1.4 End-to-End Product Flows

1. **Onboarding Flow:** User visits `/` → clicks `Get Started` → registers on `/register` (`POST /api/v1/auth/register`) → redirected to `/dashboard` with empty-state onboarding checklist or populated workspace.
2. **Upload & AI Indexing Flow:** User clicks `Upload File` (from Dashboard or My Files) → drops `Annual_Business_Strategy_2026.pdf` into Upload Modal → `POST /api/v1/files/upload` streams file to Go backend → Go triggers Python AI service (`POST /ai/v1/ingest`) → UI shows real-time status badge transition (`Uploading 78%` → `Extracting Text` → `Embedding (nomic-embed-text)` → `Processed & Indexed` in Cyan/Green).
3. **Document Intelligence Flow:** User clicks `Annual_Business_Strategy_2026.pdf` → navigates to `/intelligence/:fileId` → frontend fetches `GET /api/v1/files/:id/preview` and `GET /ai/v1/documents/:id/summary` (powered by `llama3.2:3b`) → displays PDF beside structured Executive Summary, Key Points, and Important Insights.
4. **RAG Chat Flow:** User clicks `Ask AI` on the document or navigates to `/assistant` → selects active documents in right-hand context panel → submits question → `POST /ai/v1/chat/stream` streams tokens via SSE → AI response renders with clickable inline `[1]`, `[2]` citations and source pills (`Annual_Report.pdf · Page 12`). Clicking a citation highlights the exact chunk in the right-hand Document Context panel.
5. **Semantic Search Flow:** User navigates to `/search` → enters natural-language query → `POST /ai/v1/search/semantic` embeds query with `nomic-embed-text` and queries `pgvector` → renders ranked documents with cosine-similarity match badges (`98% Match`) and highlighted snippets.
6. **Secure Sharing Flow:** User opens context menu on a file → clicks `Share` → Share Modal opens → user enters `sarah@acme.io`, selects `Editor` or `Viewer`, sets expiration (`Dec 15, 2026`), toggles password/E2E link → `POST /api/v1/shares` → appears in `/shared` management table with `Copy Link` and `Revoke` actions.

---

### 1.5 Responsive Breakpoints

| Breakpoint | Width | Layout Adaptation |
| :--- | :--- | :--- |
| **Desktop** | `>= 1280px` | Full 240px sidebar + multi-column grids (3–4 KPI cards, 2–3 column splits). |
| **Tablet** | `768px – 1279px` | Sidebar collapses to `72px` icon-only rail (expandable on hover/click) or drawer; 3-column RAG chat stacks Document Context into a toggleable right `Sheet`; 4-col KPI grids wrap to 2x2. |
| **Mobile** | `< 768px` | Sidebar hidden behind top-left hamburger `Sheet`; tables convert to stacked cards; split-pane Document Intelligence uses top `Summary | Preview` segmented tabs; upload/action bars collapse secondary actions into `...` menu. |

---

## Part 2: Screen-by-Screen UI Breakdown

---

### Screen 1: Public Landing Page (`/`)

* **Purpose:** Communicate CloudVault AI's value proposition, build security/AI trust, and convert visitors to `/register` or `/login`.
* **Layout & Component Hierarchy:**
  * **Top Sticky Navbar (`64px`):** Left: CloudVault AI logo. Center: `Features`, `Intelligence`, `Security`, `Pricing` anchor links. Right: `Sign In` (ghost button) + `Get Started` (`#7667F5` pill button).
  * **Hero Section:** Centered layout with generous vertical padding (`96px` top/bottom) and subtle radial `#7667F5` / `#36C8D8` ambient gradient (`opacity: 0.14`, blur `120px`). Headline: `"Your Files. Your Intelligence."` Subtitle: `"AI-powered cloud storage that organizes, summarizes, and answers questions from your documents."` Dual CTAs: `Get Started` (primary `#7667F5`) and `Explore Features` (secondary bordered dark button).
  * **Interactive Product Preview:** Framed `#191B22` window with `1px` `#242731` border showcasing a clean static/interactive preview of the CloudVault AI workspace.
  * **Feature Bento Grid (4 Cards):**
    1. `AI Document Intelligence` (`BrainCircuit` icon) — Instant summaries & key insights via `llama3.2:3b`.
    2. `RAG Document Chat` (`Sparkles` icon) — Grounded answers with clickable page-level citations.
    3. `Smart Semantic Search` (`Search` icon) — Meaning-based vector search powered by `pgvector`.
    4. `Zero-Knowledge Security` (`ShieldCheck` icon) — Encrypted storage with expiring, revocable share links.
  * **Minimal Footer:** Copyright (`CloudVault AI © 2026`), `Privacy`, `Terms`, `Security`, `GitHub` links.
* **Interactions & States:** Smooth scroll to anchor sections; subtle Framer Motion `whileInView` fade-up (`y: 16 -> 0`, `300ms`) on bento cards; navbar gains `backdrop-blur-md` and `#242731` bottom border on scroll.
* **Mobile Responsiveness:** Nav links collapse into a clean mobile menu drawer; bento grid stacks `4 -> 2 -> 1` columns; hero font scales via `clamp()`.

---

### Screen 2: Authentication — Login & Registration (`/login`, `/register`)

* **Purpose:** Frictionless account sign-in and onboarding into the Go + Gin JWT/session auth system.
* **Layout & Component Hierarchy:**
  * Centered single-column card (`max-width: 420px`, `#191B22`, `1px` `#242731` border, `p-8`, `rounded-2xl`) over `#101114` canvas with a subtle top indigo radial glow.
  * **Header:** CloudVault AI logo + title (`"Welcome back"` on `/login`, `"Create your vault"` on `/register`).
  * **Form Fields (`shadcn/ui Input` + `Label`):**
    * Registration adds `Full Name` (`Alex Morgan`).
    * `Email Address` (`alex@cloudvault.io`).
    * `Password` with show/hide eye toggle + password strength indicator bar on `/register`.
  * **Actions:** Full-width `#7667F5` submit button (`"Sign In"` / `"Create Account"`), divider (`"or"`), optional SSO/OAuth button (`Continue with Google`), and bottom toggle link (`"Don't have an account? Sign up"`).
* **States:**
  * **Loading:** Button shows `Loader2` spinner + `"Signing in..."` and disables inputs.
  * **Error:** Inline `#EF4444` alert banner above submit button (`"Invalid email or password"`) and red border on invalid fields.
* **API:** `POST /api/v1/auth/login`, `POST /api/v1/auth/register`.
* **Mobile Responsiveness:** Full-width card with `16px` horizontal viewport padding.

---

### Screen 3: Main Dashboard (`/dashboard`)

* **Purpose:** Orient the user within 3 seconds—showing storage health, recent files, AI insights, and immediate actions.
* **Layout & Component Hierarchy:**
  * **Header Row:** Left: `"Good morning, Alex."` + subtitle `"Your workspace is indexed and encrypted."` Right: 3 action buttons — `Upload File` (`#7667F5` primary), `New Folder` (secondary `#191B22`), `Ask AI` (secondary with `#36C8D8` sparkle icon).
  * **KPI Strip (3 Equal Cards):**
    1. `Storage Used`: `64.2 GB` (`of 100 GB`) with thin `#7667F5` → `#36C8D8` progress bar.
    2. `Total Files`: `1,284` (`+12% this month` in `#22C55E`).
    3. `Shared Links`: `48` (`active links`).
  * **Main 2-Column Split (55% / 45%):**
    * **Left Card (`Recent Documents`):** 4 crisp rows (`Project_Report.pdf · 4.2 MB`, `Financial_Analysis.xlsx · 2.8 MB`, `Annual_Strategy_2026.pdf · 8.4 MB`, `Product_Roadmap.docx · 1.9 MB`) with file-type icons, timestamps, and click-to-open `/intelligence/:id`.
    * **Right Column:**
      * **Top Card (`AI Insights`):** Subtle `#7667F5`/`#36C8D8` glass-tinted header highlighting 2 actionable insights extracted from recently indexed docs (e.g., `"24% YoY ARR target identified in Annual_Strategy_2026.pdf"`).
      * **Bottom Card (`Recent Activity`):** Compact 3-item timeline (`Alex uploaded Annual_Strategy_2026.pdf · 10m ago`, `Sarah viewed Financial_Analysis.xlsx · 1h ago`).
* **States:**
  * **Empty State (New User):** Illustration + `"Your vault is empty"` + prominent `Upload Your First PDF` CTA.
  * **Loading State:** `shadcn/ui Skeleton` placeholders matching exact KPI card and list row heights.
  * **Error State:** Non-blocking inline retry banner if `/api/v1/dashboard` fails.
* **API:** `GET /api/v1/analytics/summary`, `GET /api/v1/files?limit=4&sort=recent`, `GET /ai/v1/insights/recent`.
* **Mobile Responsiveness:** Action buttons wrap cleanly; KPI strip becomes 1-column stack; 2-column split stacks vertically.

---

### Screen 4: My Files / File Manager (`/files`, `/files/:folderId`)

* **Purpose:** Intuitive Dropbox-style file and folder management with integrated AI actions.
* **Layout & Component Hierarchy:**
  * **Action Toolbar:** Left: Interactive Breadcrumb (`My Files > Strategy & Reports`). Right: `Sort: Modified` dropdown, `Filter` dropdown (All, PDF, Spreadsheet, Doc), Grid/List view toggle, `New Folder` button, and `Upload` (`#7667F5`) button.
  * **Folders Section:** 4-column grid of minimal folder cards (`Financials 2026 · 354 items`, `Product Specs · 48 items`, `Legal & Compliance · 71 items`, `Board Decks · 16 items`). Double-click or click opens `/files/:folderId`.
  * **Files Section (List or Grid View):**
    * **Demo Rows:** `Project_Report.pdf` (`4.2 MB · Nov 20`), `Financial_Analysis.xlsx` (`2.8 MB · Nov 19`), `Research_Paper.pdf` (`6.1 MB · Nov 18`), `Business_Plan.docx` (`1.9 MB · Nov 15`), `Product_Strategy.pdf` (`3.5 MB · Nov 12`).
    * **Status Pill:** Subtle `Indexed` (Cyan) or `Processing...` (animated spinner) badge per file.
    * **Context Menu (`...` button & right-click):** `Summarize with AI` (opens `/intelligence/:id`), `Chat with File` (opens `/assistant?file=:id`), `Share` (opens Share Modal), `Download`, `Rename`, `Delete` (`#EF4444`).
* **Modals:**
  * **Upload Modal:** Drag-and-drop zone + multi-file progress bars + auto-index toggle.
  * **New Folder Modal:** Simple input + `Create` button.
* **States:** Empty folder state (`"No files in this folder yet"` + Upload dropzone); skeleton grid while fetching; toast notification on upload/delete errors.
* **API:** `GET /api/v1/folders`, `GET /api/v1/files`, `POST /api/v1/files/upload`, `PATCH /api/v1/files/:id`, `DELETE /api/v1/files/:id`.
* **Mobile Responsiveness:** Grid collapses `4 -> 2 -> 1`; toolbar secondary controls collapse into a `Filter & Sort` sheet.

---

### Screen 5: AI Document Intelligence (`/intelligence/:fileId`)

* **Purpose:** Inspect a document side-by-side with its structured AI-generated summary, key points, and insights.
* **Layout & Component Hierarchy:**
  * **Subheader:** Breadcrumb (`My Files / Annual Business Strategy 2026.pdf`), Cyan/Green `Processed & Indexed` status pill, `Ask AI` button (navigates to `/assistant?file=:id`), and `Share` button.
  * **Split-Pane Workspace (42% Left / 58% Right):**
    * **Left Pane:** Clean dark PDF/document viewer toolbar (page indicator `1 / 42`, zoom controls, download icon) + rendered page preview + compact **Document Metadata** strip (`42 Pages · 8.4 MB · Alex Morgan · Nov 20, 2026`).
    * **Right Pane (`AI Intelligence` Card):**
      * Minimal underline tabs: `Summary` | `Key Points` | `Insights` (+ `Regenerate` icon button).
      * **Executive Summary:** Concise 2-3 sentence overview generated by `llama3.2:3b`.
      * **Key Points:** Bulleted list with Cyan (`#36C8D8`) dot markers.
      * **Important Insights Callout:** Nested card with `3px` Electric Indigo (`#7667F5`) left border highlighting strategic takeaways (`24% YoY ARR Growth Target`, `Q2 APAC Market Entry`).
* **States:**
  * **Processing / Loading:** If the file is still indexing, right pane shows animated step tracker (`Extracting text -> Chunking -> Generating summary with llama3.2:3b`) with skeleton lines.
  * **Error:** `"Could not generate AI summary"` with `Retry Analysis` button.
* **API:** `GET /api/v1/files/:id/content`, `GET /ai/v1/documents/:id/intelligence`, `POST /ai/v1/documents/:id/summarize`.
* **Mobile Responsiveness:** Split-pane converts into top segmented control (`AI Intelligence` | `Document Preview`) so neither pane is squished.

---

### Screen 6: Chat With Your Documents — RAG AI Assistant (`/assistant`)

* **Purpose:** Multi-document conversational Q&A grounded strictly in uploaded files with verifiable page-level citations.
* **Layout & Component Hierarchy (3-Column Workspace):**
  * **Left Inner Column (`240px`, `#15171D`):** `+ New Chat` (`#7667F5`) button + scrollable `Conversations` list (`Q4 Financial Risk Analysis`, `2026 Strategy Summary`, `Vendor Contract Review`).
  * **Center Chat Area (`flex-1`, `#101114`):**
    * Top title bar (`Q4 Financial Risk Analysis`) + mobile toggle button for Document Context.
    * Message thread: User bubble right-aligned (`#191B22`, `1px` `#242731` border); AI response card left-aligned with Cyan sparkle icon, structured response (`"The report identifies three major financial risks:"` + numbered list with clickable inline `[1]`, `[2]`, `[3]` citation chips), and a `Sources` footer row (`Annual_Report.pdf · Page 12`, `Financial_Analysis.pdf · Page 8`).
    * Suggested follow-up chips (`"Compare this to the previous quarter"`, `"List actions for supply chain risk"`) above a sleek bottom input bar (`"Ask a question about your documents..."` + Send button).
  * **Right Context Panel (`300px`, `#191B22`):**
    * `Document Context` header + `2 Active Documents` checklist (`Annual_Report.pdf`, `Financial_Analysis.pdf` with Indigo checkboxes to scope RAG retrieval).
    * **Source Excerpt Viewer:** Displays the highlighted text chunk and page number when a user clicks any `[1]` citation or source pill.
* **States:**
  * **Empty State:** Centered welcome prompt + 3 starter cards (`Summarize Q4 Risks`, `Compare Revenue Projections`, `Extract Key Deadlines`).
  * **Streaming / Loading:** Animated Cyan `"Analyzing referenced pages..."` indicator + real-time token streaming.
  * **Error:** Inline message retry button if Ollama/FastAPI stream disconnects.
* **API:** `GET /ai/v1/conversations`, `POST /ai/v1/chat/stream` (SSE), `GET /ai/v1/documents/:id/chunks`.
* **Mobile Responsiveness:** Left Conversations and Right Document Context panels move into slide-over `Sheet` drawers accessible from the chat header.

---

### Screen 7: Smart Semantic Search (`/search`)

* **Purpose:** Meaning-based vector search across all indexed documents using `nomic-embed-text` + `pgvector`.
* **Layout & Component Hierarchy:**
  * **Prominent Search Bar:** Full-width input (`56px` height, `#191B22`, subtle `#7667F5` focus border) with query `"Find documents discussing financial growth strategies."` and a Cyan `Semantic` mode badge.
  * **Filter Strip:** File-type pills (`All`, `PDF`, `Spreadsheet`, `Doc`), Date Range dropdown (`Last 90 Days`, `Last 30 Days`, `All Time`), and Sort (`Relevance` | `Newest`).
  * **Results List (Single-Column Spacious Cards):**
    * **Demo Results:**
      1. `Annual_Business_Strategy_2026.pdf` — `98% Match` (Cyan badge) — snippet highlighting `financial growth` in soft `#7667F5/20` pill.
      2. `Financial_Analysis.xlsx` — `94% Match` — snippet highlighting `ARR expansion`.
      3. `Product_Strategy.pdf` — `89% Match` — snippet highlighting `revenue strategy`.
    * Each card shows page number, last modified date, and quick actions (`Open Intelligence`, `Ask AI`).
* **States:**
  * **Initial / Empty Query:** Recent semantic queries + `"Search by concept, question, or phrase across your entire vault."`
  * **No Results:** `"No matching passages found"` + suggestion to broaden filters.
  * **Loading:** 3 pulsing skeleton result cards.
* **API:** `POST /ai/v1/search/semantic`.
* **Mobile Responsiveness:** Filter pills scroll horizontally (`overflow-x-auto`); result card metadata wraps cleanly.

---

### Screen 8: Analytics Dashboard (`/analytics`)

* **Purpose:** Provide clear visibility into storage consumption, file distribution, and AI processing performance.
* **Layout & Component Hierarchy:**
  * **Header:** `"Analytics"` + Date Range selector (`Last 30 Days`, `Last 90 Days`, `This Year`).
  * **Top KPI Row (4 Cards):**
    1. `Storage Used`: `64.2 GB`
    2. `Available`: `35.8 GB`
    3. `Total Files`: `1,284`
    4. `AI Summaries`: `342`
  * **Middle Chart Row (60% / 40% Split using Recharts):**
    * **Left (`File Uploads Over Time`):** Smooth `AreaChart` with `#7667F5` → `#36C8D8` gradient fill, clean gridlines (`#242731`), and dark custom tooltip.
    * **Right (`Storage by File Type`):** `PieChart` (donut) showing `PDFs 48%`, `Spreadsheets 24%`, `Docs 18%`, `Media 10%` with aligned legend.
  * **Bottom Row (50% / 50% Split):**
    * **Left (`Processing Performance`):** Slim progress bars for `Indexed: 99.2%` and `Avg Speed: 1.4s`.
    * **Right (`Activity Trends`):** Minimalist `BarChart` of daily uploads & AI queries.
* **States:** Skeleton chart containers during fetch; empty-state `"Upload documents to view usage trends"` if 0 files; retry card on API failure.
* **API:** `GET /api/v1/analytics/storage`, `GET /api/v1/analytics/activity`, `GET /ai/v1/analytics/processing`.
* **Mobile Responsiveness:** KPIs stack `4 -> 2 -> 1`; charts stack vertically with responsive `ResponsiveContainer width="100%"`.

---

### Screen 9: Secure File Sharing (`/shared`)

* **Purpose:** Centralized access control, permission management, and audit visibility for shared documents.
* **Layout & Component Hierarchy:**
  * **Header:** Shield icon + `"Secure File Sharing"` + Cyan `E2E Encrypted` badge + `+ Share File` (`#7667F5`) button.
  * **Summary Strip (3 Compact Cards):** `18 Active Links`, `24 Recipients`, `3 Expiring Soon`.
  * **Permissions Table (`#191B22`):**
    * Columns: `Document` | `Recipient` | `Access` | `Expires` | `Actions`.
    * **Demo Rows:**
      * `Annual_Business_Strategy_2026.pdf` | `sarah@acme.io` | `Editor` (Indigo dropdown pill) | `Dec 15, 2026` | `Copy` + `Revoke Access`
      * `Financial_Analysis.xlsx` | `david@acme.io` | `Viewer` (Slate dropdown pill) | `Nov 30, 2026` | `Copy` + `Revoke Access`
      * `Project_Report.pdf` | `john@cloudvault.io` | `Editor` | `Never` | `Copy` + `Revoke Access`
      * `Product_Strategy.pdf` | `chloe@acme.io` | `Viewer` | `Mar 01, 2027` | `Copy` + `Revoke Access`
  * **Recent Sharing Activity Card:** 3-line audit log (`Alex shared 'Financial_Analysis.xlsx' with david@acme.io · 10m ago`).
* **Modals & Interactions:**
  * **Share File Modal:** File picker, recipient email input, role selector (`Viewer` / `Editor`), expiration date picker, optional password toggle, `Generate Secure Link` button.
  * **Revoke Confirmation Dialog:** Prevents accidental link revocation.
  * **Copy Link Toast:** `"Encrypted link copied to clipboard"`.
* **API:** `GET /api/v1/shares`, `POST /api/v1/shares`, `PATCH /api/v1/shares/:id`, `DELETE /api/v1/shares/:id`.
* **Mobile Responsiveness:** Table converts to stacked permission cards showing document name, recipient badge, role selector, and action buttons.

---

### Screen 10: User Settings (`/settings`)

* **Purpose:** Manage profile details, security credentials, AI model preferences, and storage plan.
* **Layout & Component Hierarchy:**
  * **Header:** `"Settings"` + subtitle `"Manage your account, security, and AI workspace preferences."`
  * **Vertical Section Tabs (Left `200px` on desktop, horizontal scroll tabs on mobile):** `Profile` | `Security` | `AI Preferences` | `Storage & Plan`.
  * **Settings Content Card (`#191B22`, `max-width: 720px`):**
    * **Profile:** Avatar upload, `Full Name` (`Alex Morgan`), `Email` (`alex@cloudvault.io`), `Save Changes` (`#7667F5`).
    * **Security:** Change Password form, Two-Factor Authentication (2FA) toggle switch, Active Sessions list with `Revoke Session` action.
    * **AI Preferences:** Toggle `Auto-index uploaded PDFs`, toggle `Auto-generate summary on upload`, default citation display (`Inline + Page Footers`), active model indicator (`Ollama · llama3.2:3b & nomic-embed-text`).
    * **Storage & Plan:** Visual breakdown of `64.2 GB / 100 GB` used, danger zone (`Delete Account` / `Purge Vector Index`).
* **States:** Toast confirmation on save (`"Preferences updated"`); inline validation errors on password mismatch; loading spinner on save button.
* **API:** `GET /api/v1/users/me`, `PATCH /api/v1/users/me`, `PATCH /api/v1/users/me/preferences`, `POST /api/v1/auth/password`.
* **Mobile Responsiveness:** Left section tabs become a horizontal scrollable pill bar above the form card.
