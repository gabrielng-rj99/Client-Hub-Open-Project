# Query Scalability Architecture — Client Hub Open Project

## Overview

This document describes the query architecture and scalability patterns used in the Client Hub project, with emphasis on financial queries. It serves as a blueprint for maintaining and extending the system while preventing performance degradation.

---

## The Problem We Solved

### Initial Issue: "Query Bomb" in Financial Page

The original `GetFinancialDetailedSummary()` function was generating **15,000–20,000 SQL queries per request**, causing the Financial dashboard to freeze for **15–30 seconds**. This happened because:

1. **N+1 Query Problem**: For every financial record, the code fetched related installments individually
2. **Nested Loops**: Multiple layers of iteration without batch loading
3. **No Aggregation**: All calculations done in application memory instead of at the database level
4. **Inefficient Date Filtering**: Time-based queries without proper indexing

**After fixes**: Same page now loads in **0.72 seconds** with proper batching, aggregation, and indexing.

---

## Core Scalability Principles

### 1. **Batch Queries Over Individual Fetches**

❌ **Anti-pattern (N+1)**:

```go
// For each of 100 contracts...
for _, contract := range contracts {
    // ...fetch its 5 installments individually
    installments, err := repo.GetInstallmentsByContractID(contract.ID)
}
// Result: 100 queries + 500 installment queries = 600 queries
```

✅ **Correct Pattern (Batch)**:

```go
// Fetch all 500 installments in ONE query
installments, err := repo.GetInstallmentsByContractIDsBatch(contractIDs)
// Result: 1 query
```

**When to use**:

- Loading related data for multiple parent records
- Processing collections of IDs
- Joining across multiple entities

---

### 2. **Aggregate at Database Level, Not Application**

❌ **Anti-pattern (App-level aggregation)**:

```go
var totalPaid float64
for _, installment := range installments {
    if installment.Status == "pago" {
        totalPaid += installment.Value
    }
}
// Loops through thousands of rows in memory
```

✅ **Correct Pattern (SQL aggregation)**:

```sql
SELECT SUM(value) as total_paid
FROM financial_installments
WHERE status = 'pago' AND contract_id = ANY($1)
-- Database engine optimizes this, returns 1 row
```

**When to use**:

- Sums, counts, averages
- Grouping across large datasets
- Time-range queries

---

### 3. **Enrich API Responses with JOINed Data**

❌ **Anti-pattern (Multiple API Calls)**:

```javascript
// Frontend makes separate calls to build a table
const contracts = await fetch('/api/contracts');        // 1 query
const clients = await fetch('/api/clients');            // 1 query  
const categories = await fetch('/api/categories');      // 1 query
// Browser must wait for ALL 3 to finish, then cross-reference with Maps
// Total time: MAX(3 calls) + Map building + client-side filtering/sorting
```

✅ **Correct Pattern (Enriched Response)**:

```go
// Backend JOINs related tables and returns everything inline
// SELECT c.*, cl.name as client_name, cat.name as category_name, ...
// FROM contracts c
// LEFT JOIN clients cl ON ...
// LEFT JOIN categories cat ON ...
contracts, err := repo.GetAllContractsWithEnrichedData()
// Result: 1 query, response already has client_name, category_name
// Frontend renders immediately, no waiting, no cross-reference Maps needed
```

**Applied To:**

- `GET /contracts` — Now includes `client_name`, `client_nickname`, `category_name`, `subcategory_name` inline
- `GET /financial` — Now includes `contract_model`, `client_name`, `category_name`, `subcategory_name`, `contract_start`, `contract_end` inline

**Why This Matters:**

- **Contracts page**: Was making 3 parallel API calls (`/contracts` + `/clients` + `/categories`). Now makes 1 call with enriched data. Renders immediately instead of waiting for all 3 responses.
- **Financial page**: Was making 6 parallel API calls (`/financial` + `/contracts` + `/clients` + `/categories` + `/upcoming` + `/overdue`). Primary table now loads from 1 enriched call. Auxiliary data loads in background for modals/filters.
- **Real-world impact**: Page becomes interactive 3-6x faster depending on network latency.

**Implementation**:

```sql
-- Backend query (one example)
SELECT
    c.id, c.model, c.item_key, c.start_date, c.end_date,
    c.subcategory_id, c.client_id, c.affiliate_id, c.archived_at,
    COALESCE(cl.name, '') AS client_name,
    cl.nickname AS client_nickname,
    COALESCE(cat.name, '') AS category_name,
    COALESCE(s.name, '') AS subcategory_name
FROM contracts c
LEFT JOIN clients cl ON cl.id = c.client_id
LEFT JOIN subcategories s ON s.id = c.subcategory_id
LEFT JOIN categories cat ON cat.id = s.category_id
```

```javascript
// Frontend: use enriched fields directly
const contractClientName = contract.client_name;  // No map lookup needed
const contractCategory = contract.category_name;   // Data already present
// If auxiliary data (for modals) hasn't loaded yet, that's OK —
// Table renders with main data, dropdowns populate when background calls finish
```

---

### 4. **Use Covering Indexes for Large Tables**

❌ **Anti-pattern (No index)**:

```sql
SELECT COUNT(*) FROM financial_installments WHERE status = 'pago';
-- Full table scan (10,000+ rows)
```

✅ **Correct Pattern (Covering index)**:

```sql
-- Create composite covering index
CREATE INDEX idx_installments_status_value_contract 
ON financial_installments (status, contract_id) 
INCLUDE (value, due_date);

-- Query uses index, no table access
SELECT COUNT(*) FROM financial_installments WHERE status = 'pago';
```

**Current indexes** (in schema):

- `idx_financial_installments_contract_status`: Status filtering by contract
- `idx_financial_installments_due_date`: Date range queries
- `idx_financial_installments_status_value`: Status + aggregations

---

### 5. **Pagination for Large Result Sets & Frontend Syncing**

❌ **Anti-pattern (No limit)**:

```go
// Load ALL 5,000 contracts at once
contracts, err := repo.GetAllContracts()
// Memory spike, network bloat, slow rendering
```

✅ **Correct Pattern (Pagination & Scroll sync)**:

```go
// Load 20 per page
contracts, total, err := repo.GetContractsPaginated(limit: 20, offset: 0)
// Next page: offset: 20, etc.
```

**Implementation & Pitfalls**:

- All list endpoints now support `limit` (default 20, max 500) and `offset`
- Backend returns `{data, total, limit, offset}`
- Frontend implements infinite scroll or page navigation
- **Gotcha (React-Select)**: When implementing `onMenuScrollToBottom` to trigger pagination offsets, avoid passing `isLoading=true` while keeping previous options mounted. `react-select` recalculates its scroll height when `isLoading` changes, tricking it into firing a cascade of `offset` requests (infinite scroll loop). Handle pagination loading state internally instead of using the library's built-in `isLoading` prop if you want seamless background scrolling.
- **Gotcha (Backend SQL logic)**: Always verify that `limit` parameters parsed by handlers propagate down to the actual data repository (`category_store.go`). HTTP query limits are purely cosmetic if the database driver doesn't physically append `LIMIT $X` to the executed SQL string.

#### Example 3: Simplifying Soft-Delete States (Schema Cleanup)

Tracking redundant soft-delete logic (e.g., using both `archived_at` and a custom feature like `cancelled_at`) severely bloats execution plans because every SELECT query requires multifaceted `NULL` checks.

```sql
-- BEFORE: Bloated query execution plan
SELECT * FROM contracts 
WHERE client_id = $1 
AND archived_at IS NULL 
AND cancelled_at IS NULL; 

-- AFTER: Schema simplification
-- Dropped cancelled_at entirely. Consolidated logic to archived_at.
SELECT * FROM contracts 
WHERE client_id = $1 
AND archived_at IS NULL;
```

*Takeaway: The best query optimization is often deleting unnecessary schema columns. Fewer conditions require fewer composite indexes and result in highly predictable explain plans.*

---

### 6. **Frontend Request Patterns: Load Critical Data First, Auxiliary Data in Background**

❌ **Anti-pattern (Blocking All Requests)**:

```javascript
// Frontend waits for Promise.all() to complete before rendering
const [financial, contracts, clients, categories, upcoming, overdue] = 
    await Promise.all([
        loadFinancial(),      // 1s
        loadContracts(),      // 0.8s
        loadClients(),        // 0.6s
        loadCategories(),     // 0.7s
        loadUpcoming(),       // 0.5s
        loadOverdue(),        // 0.4s
    ]);
// User sees loading spinner for MAX(all) = ~1s, even though financial alone renders fine
```

✅ **Correct Pattern (Progressive Loading)**:

```javascript
// 1. Load critical data FIRST — render immediately
try {
    const financialData = await loadFinancial();  // Enriched, renders table immediately
    setFinancialRecords(financialData);           // UI interactive NOW
} catch (err) {
    setError(err.message);
} finally {
    setLoading(false);  // Stop spinner
}

// 2. Load auxiliary data in BACKGROUND (needed for dropdowns/modals only)
Promise.all([
    loadContracts(),
    loadClients(),
    loadCategories(),
    loadUpcoming(),
    loadOverdue(),
]).then(([contracts, clients, categories, upcoming, overdue]) => {
    setContracts(contracts);
    setClients(clients);
    setCategories(categories);
    setUpcomingFinancials(upcoming);
    setOverdueFinancials(overdue);
}).catch(err => console.warn("Background load:", err));
// User sees table data in ~0.2s, dropdowns populate in next ~1s (no blocking)
```

**Applied To:**

- **Contracts page**: Loads enriched contracts first (instant), then clients/categories in background
- **Financial page**: Loads enriched financial records first (instant), then contracts/clients/categories/upcoming/overdue in background

**Why This Matters:**

- Page becomes interactive 3-6x faster
- User can start reading/scrolling while dropdowns populate
- Network latency becomes invisible to user perception
- Graceful degradation: if background call fails, page still works (just without filters temporarily)

**Implementation**:

```javascript
const loadData = async (silent = false) => {
    if (!silent) setLoading(true);
    
    try {
        // PRIMARY: Critical path data — table content
        const financialData = await financialApi.loadFinancial(apiUrl, token, onTokenExpired);
        setFinancialRecords(Array.isArray(financialData) ? financialData : financialData.data || []);
    } catch (err) {
        setError(err.message);
    } finally {
        setLoading(false);  // Unblock UI immediately
    }
    
    // SECONDARY: Background refresh of auxiliary data (non-blocking)
    Promise.all([
        fetchContracts({}, false).then(res => setContracts(res?.data || [])).catch(() => {}),
        fetchClients({}, false).then(res => setClients(res?.data || [])).catch(() => {}),
        fetchCategories(false).then(res => setCategories(res?.data || [])).catch(() => {}),
    ]).catch(err => console.warn("Background load:", err));
};
```

> **⚠️ Gotcha**: `fetchCategories` accepts `(forceRefresh)` only, while `fetchContracts`/`fetchClients` accept `(params, forceRefresh)`. Calling `fetchCategories({}, false)` passes `{}` as `forceRefresh` (truthy in JavaScript), bypassing the cache entirely. Always verify call signatures match function definitions.

---

### 7. **Frontend DataContext Cache with TTL**

The `DataContext` provides centralized fetch functions with in-memory caching via `cacheManager`.

✅ **Correct Pattern (Use cache for auxiliary data)**:

```javascript
// Background auxiliary data — use cache (forceRefresh=false)
fetchCategories(false)   // Uses cached data if < 5 min old
fetchContracts({}, false) // Same — cache hit eliminates the request
fetchClients({}, false)   // Same
```

❌ **Anti-pattern (Bypass cache unintentionally)**:

```javascript
// WRONG: {} is truthy, treated as forceRefresh=true
fetchCategories({}, false)  // Always bypasses cache!
// WRONG: true forces refetch even if data is fresh
fetchCategories(true)       // Only use when data was just mutated
```

**How the cache works** (`cacheManager.js`):

- Singleton `CacheManager` with `Map` for data and `Map` for timestamps
- `get(key, ttl)` returns cached data if `(Date.now() - timestamp) < ttl`, else `null`
- Default TTL: 5 minutes
- Mutation operations (`createContract`, `deleteClient`, etc.) call `cacheManager.invalidateResource()` to bust the cache
- `requestInProgressRef` deduplicates concurrent requests for the same key

**When to use `forceRefresh=true`**:

- After a mutation (create/update/delete) — cache is stale
- Never for read-only page loads that share data with other pages

---

### 8. **Backend HTTP Caching with Cache-Control**

For endpoints that return stable, rarely-changing configuration data, the backend sends `Cache-Control` headers:

```go
// Stable config endpoints — 5 min browser cache
w.Header().Set("Cache-Control", "private, max-age=300, must-revalidate")
```

**Applied to:**

- `GET /api/settings` — System settings rarely change mid-session
- `GET /api/user/theme` — User's theme preference is stable
- `GET /api/system-config/dashboard` — Dashboard config is stable

**Result**: Revisiting `/settings` or `/appearance` pages produces **0 API requests** — the browser serves from disk cache.

**When NOT to use**:

- List endpoints (`/contracts`, `/clients`) — data changes frequently
- Write endpoints — never cache POST/PUT/DELETE responses
- Auth endpoints — tokens must always be validated fresh

---

### 9. **Perceived Performance: Non-Blocking Transitions**

Even with highly optimized backend queries and an in-memory `DataContext` cache, the *perception* of speed can be ruined by rigid front-end loading states.

❌ **Anti-pattern (Blocking Page Loaders)**:

```javascript
// A component that replaces the entire page content with a spinner if ANY data is loading.
// This was present in Financial, Dashboard, Contracts, etc.
if (loading) {
    return <div className="loading-container"><Spinner /></div>;
}
return <div className="page-content">{/*...*/}</div>;
```

*Impact*: The sidebar and header remain, but the main content area completely flashes to a loading screen. This is visually jarring (Layout Jitter) and makes the app feel heavier than it actually is, even if the load takes only 100ms.

✅ **Correct Pattern (Global Progress Bar + Instant Render)**:

```javascript
// Page structure renders immediately. 
// Data tables handle their own empty/loading states gracefully, 
// OR the data is already in cache and renders instantly.
return <div className="page-content">{/*... table rendering data */}</div>;
```

*Combined with*:
A global `TopProgressBar` component (mounted inside `App.jsx`) that intercepts route clicks natively at the DOM level and tracks `location.pathname` changes.

**Why This Matters:**

- When navigating to a cached page, the structural DOM never unmounts unnecessarily. The new page content replaces the old content instantly.
- The top progress bar gives immediate tactical feedback ("your click registered and we are working on it") without destroying the user's visual context.
- It simulates the "premium feel" of modern enterprise applications, closing the gap between actual network speed and human perception.

---

## Case Study: Contracts & Financial Pages — From 6 API Calls to 1 Enriched Response

### The Problem: "Slow Despite Earlier Fixes"

Even after the initial Financial Query Bomb optimization (documented above), the **Contracts** and **Financial** pages remained slow in production. Root cause analysis revealed:

**Frontend Pattern**:

- **Contracts page**: Made 3 parallel API calls
  1. `GET /contracts` (returns all contracts, no enrichment)
  2. `GET /clients` (all clients, for mapping)
  3. `GET /categories` (all categories, for mapping)
  - Result: 3 requests blocking → table renders only after **all 3 responses** arrive

- **Financial page**: Made up to 6 parallel API calls
  1. `GET /financial` (all financials, no enrichment)
  2. `GET /contracts` (all contracts, for mapping)
  3. `GET /clients` (all clients, for mapping)
  4. `GET /categories` (all categories, for mapping)
  5. `GET /financial/upcoming` (separate endpoint)
  6. `GET /financial/overdue` (separate endpoint)
  - Result: 6 requests → **3-6x slower** than needed due to waterfalls and parallel overhead

**Backend Pattern**:

- `GET /contracts` used `GetAllContractsIncludingArchived()` → returns all contracts, no JOINs
- `GET /financial` used `GetAllFinancials()` → no enrichment with contract/client/category names
- Frontend had to manually build Maps and cross-reference by ID for every rendered row

**Client-Side Impact**:

- ContractsTable.jsx reconstructed Maps on every render
- For each table row, performed ID-based lookups: `clientMap[contract.client_id]`
- No caching of enriched data → expensive recalculations on sort/filter
- Auxiliary data (modals, dropdowns) blocking main table render

### The Solution: Enriched Responses + Progressive Loading

#### 1. Backend: Enrich API Responses with JOINs

**Before** (`GetAllContractsIncludingArchived`):

```sql
SELECT c.* FROM contracts c
```

Result: Contract with no client/category names; frontend must make 2 more calls

**After** (same endpoint, optimized):

```sql
SELECT
    c.id, c.model, c.item_key, c.start_date, c.end_date,
    c.subcategory_id, c.client_id, c.affiliate_id, c.archived_at,
    COALESCE(cl.name, '') AS client_name,
    cl.nickname AS client_nickname,
    COALESCE(cat.name, '') AS category_name,
    COALESCE(s.name, '') AS subcategory_name
FROM contracts c
LEFT JOIN clients cl ON cl.id = c.client_id
LEFT JOIN subcategories s ON s.id = c.subcategory_id
LEFT JOIN categories cat ON cat.id = s.category_id
```

**Modified Files**:

- `backend/domain/models.go` — Added enriched fields to `Contract` struct:
  - `ClientName string`
  - `ClientNickname string`
  - `CategoryName string`
  - `SubcategoryName string`

- `backend/repository/contract/contract_store.go` — Updated `GetAllContractsIncludingArchived()` to perform JOINs

- `backend/repository/contract/financial_store.go` — Updated `GetAllFinancials()` and `GetAllFinancialsPaged()` to enrich with:
  - `ContractModel string`
  - `ClientName string`
  - `CategoryName string`
  - `SubcategoryName string`
  - `ContractStart *time.Time`
  - `ContractEnd *time.Time`

#### 2. Frontend: Progressive Loading Pattern

**Before**:

```javascript
// Wait for all 3 to finish
const [contracts, clients, categories] = await Promise.all([
  fetchContracts(),  // no enrichment
  fetchClients(),
  fetchCategories(),
]);
// Then render table (3-6 seconds later)
```

**After**:

```javascript
// 1. Load enriched data immediately (blocking)
const contractsData = await fetchContracts(); // NOW returns client_name, category_name inline
setContracts(contractsData);  // Render table immediately with enriched names

// 2. Load auxiliary data in background (non-blocking)
Promise.allSettled([
  fetchClients(),   // For modals, dropdowns (not critical)
  fetchCategories(), // For filters, searches
]).then(([clientRes, catRes]) => {
  if (clientRes.status === 'fulfilled') setClients(clientRes.value);
  if (catRes.status === 'fulfilled') setCategories(catRes.value);
});
```

**Modified Files**:

- `frontend/src/pages/Contracts.jsx` — Refactored to:
  1. Load enriched contracts first
  2. Use `contract.client_name`, `contract.category_name` directly
  3. Load clients/categories in background for dropdowns
  4. Filters/sorting use enriched fields when available, fallback to Maps

- `frontend/src/pages/Financial.jsx` — Similar pattern:
  1. Load enriched financial data first (includes contract model, client name, etc.)
  2. Render main table immediately
  3. Load contracts/clients/categories in background for modals/filters

- `frontend/src/components/contracts/ContractsTable.jsx` — Updated to use enriched fields:

  ```javascript
  // Old: Look up client name from separate Map
  const clientName = clientMap[contract.client_id]?.name || 'Unknown';
  
  // New: Use enriched field directly
  const clientName = contract.client_name || clientMap[contract.client_id]?.name || 'Unknown';
  ```

- `frontend/src/utils/contractHelpers.js` — Adapted filters to use enriched fields:
  - `getClientName()` now accepts enriched contract object and returns `contract.client_name` directly
  - `getCategoryName()` similarly adapted

### Performance Impact

| Metric | Before | After | Improvement |
| -------- | -------- | ------- | ------------- |
| **Contracts Page Load (FCP)** | 4–6 sec | 0.8–1.2 sec | **4-5x faster** |
| **Financial Page Load (FCP)** | 6–8 sec | 1.0–1.5 sec | **4-8x faster** |
| **API Calls (Critical Path)** | 3–6 calls | 1 call | **3-6x fewer** |
| **Client-Side Map Building** | Every render | On demand | **Eliminated** |
| **Browser Jank on Sort/Filter** | High (recalc maps) | None | **Smooth** |

**Why This Works**:

- **Contracts table now renders from a single enriched API call** — 1 request instead of 3, arrives in ~200-400ms instead of waiting for 3 parallel responses (800ms+ typical).
- **Auxiliary data (for modals/dropdowns) loads in background** — Non-blocking. If it arrives, great. If it hasn't, table still shows main data with fallback lookups.
- **Frontend rendering is instant** — No Maps to build, no ID lookups per row. Just render `contract.client_name` as a string.
- **Filters and sorts are faster** — Operating on enriched strings, not requiring Map lookups or cross-references.

### Trade-offs & Considerations

| Aspect | Trade-off | Mitigation |
| -------- | ----------- | ----------- |
| **Payload size** | Enriched responses are slightly larger (client_name + category_name per row) | Still much smaller than 3 separate API calls; names are strings, ~50 bytes per record |
| **Query complexity** | Slightly more complex SQL (LEFT JOINs) | Queries still execute in <100ms with proper indexes; no performance regression |
| **Cache invalidation** | If client/category names change, enriched responses may be stale | Acceptable; names rarely change; can add cache-busting header if needed |
| **Pagination** | If `/contracts` grows to 100k+, returning all might be slow | Already addressed: can implement pagination (`limit`/`offset`) on backend and frontend will paginate the enriched response |

### Lessons Learned

1. **Don't assume frontend is the bottleneck**: The problem wasn't React rendering; it was 6 API calls blocking the table render.
2. **Enrich at the database level**: JOINs are cheaper than round-trips. One enriched 1MB response is faster than 3 separate 300KB responses.
3. **Progressive loading is not just for UX**: It's for performance. Load critical data first; load auxiliary data in background without blocking.
4. **Eliminate client-side Maps**: If data is simple enough to JOIN at the DB, do it. Saves browser memory and render cycles.

---

## Financial Query Architecture

### GetFinancialDetailedSummary() — Before vs. After

#### Before: 15,000–20,000 queries, 15–30 seconds

```text
1. GetContracts() [1 query]
   ↓
2. For each contract:
   - GetFinancialByContractID() [~2,500 queries]
   - For each financial:
     - GetInstallmentsByFinancialID() [~15,000 queries]
     - For each installment:
       - Calculate status, aggregate [in-memory]
3. SUM in application loop [slow]
```

#### After: 4–5 queries, ~0.72 seconds

```text
1. GetContracts() [1 query]
2. GetFinancialsByContractIDBatch(contractIDs) [1 query, JOIN optimized]
3. GetInstallmentsByContractIDBatch(contractIDs) [1 query, aggregate in SQL]
4. GetInstallmentsByDueDateRanges(dateRanges) [1-2 queries for range filtering]
```

### Key Changes

| Aspect | Before | After |
| :--- | :--- | :--- |
| **Queries** | 15,000–20,000 | 4–5 |
| **Load Time** | 15–30 sec | 0.72 sec |
| **Approach** | N+1, app aggregation | Batch, SQL aggregation |
| **Indexing** | Minimal | Covering indexes on status, date ranges |
| **Memory** | Thousands of objects | Only results needed |

---

## Repository Layer: Batch Query Methods

### Naming Convention

```go
// Single entity
GetFinancialByID(id string) (*Financial, error)

// Multiple entities (batch)
GetFinancialsByIDBatch(ids []string) ([]*Financial, error)

// All (use with caution, only for small tables)
GetAllFinancials() ([]*Financial, error)

// Paginated (preferred for large lists)
GetFinancialsPaginated(limit, offset int) ([]*Financial, int, error)

// Aggregated (count, sum, group by)
GetInstallmentsSummaryByStatus(contractIDs []string) (map[string]int, error)
```

### Example: Financial Installments Batch Query

```go
// GetInstallmentsByContractIDBatch fetches all installments for multiple contracts
// in a single query, using SQL ANY() for efficient filtering.
func (r *FinancialStore) GetInstallmentsByContractIDBatch(
    ctx context.Context,
    contractIDs []string,
) ([]*Installment, error) {
    query := `
        SELECT id, financial_id, contract_id, due_date, value, status, paid_date
        FROM financial_installments
        WHERE contract_id = ANY($1)
        ORDER BY contract_id, due_date
    `
    rows, err := r.db.QueryContext(ctx, query, pq.Array(contractIDs))
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var installments []*Installment
    for rows.Next() {
        var inst Installment
        if err := rows.Scan(&inst.ID, &inst.FinancialID, &inst.ContractID, 
            &inst.DueDate, &inst.Value, &inst.Status, &inst.PaidDate); err != nil {
            return nil, err
        }
        installments = append(installments, &inst)
    }
    return installments, rows.Err()
}
```

---

## SQL Aggregation Patterns

### Pattern 1: Status-Based Aggregations

```sql
-- Get count and sum by status
SELECT 
    status,
    COUNT(*) as count,
    SUM(value) as total_value
FROM financial_installments
WHERE contract_id = ANY($1)
GROUP BY status;
```

### Pattern 2: Date Range Grouping

```sql
-- Monthly summary: past, current, next
WITH monthly_data AS (
    SELECT 
        DATE_TRUNC('month', due_date) as month,
        SUM(value) as total,
        COUNT(*) as count
    FROM financial_installments
    WHERE contract_id = ANY($1)
      AND due_date >= CURRENT_DATE - INTERVAL '30 days'
      AND due_date <= CURRENT_DATE + INTERVAL '90 days'
    GROUP BY DATE_TRUNC('month', due_date)
)
SELECT * FROM monthly_data ORDER BY month;
```

### Pattern 3: Covering Index for Multi-Column Filters

```sql
-- Index definition (covers: status, contract_id, value, due_date)
CREATE INDEX idx_installments_status_value_contract 
ON financial_installments (status, contract_id) 
INCLUDE (value, due_date);

-- Query benefits from covering index (no table access)
SELECT SUM(value) as total
FROM financial_installments
WHERE status = 'pago' AND contract_id = ANY($1);
```

---

## Frontend Query Patterns

### Don't: Fire Multiple Requests in Parallel

```javascript
// ❌ Anti-pattern: 4 concurrent requests, rate-limited
const financial = await financialApi.loadFinancial(apiUrl, token);
const summary = await financialApi.getFinancialSummary(apiUrl, token);
const monthly = await financialApi.getMonthlySummary(apiUrl, token, year, month);
const upcoming = await financialApi.getUpcomingFinancial(apiUrl, token);
```

### Do: Batch Frontend Requests or Use Endpoints That Return Multiple Data

```javascript
// ✅ Better: Single endpoint returns dashboard data
const dashboard = await financialApi.getDashboard(apiUrl, token);
// Returns: { financial: [...], summary: {...}, monthly: {...} }
```

Or, if multiple endpoints are needed:

```javascript
// ✅ Acceptable: Sequential loading with proper error handling
try {
    const financial = await financialApi.loadFinancial(apiUrl, token);
    const summary = await financialApi.getFinancialSummary(apiUrl, token);
} catch (err) {
    if (err.message.includes("Excesso de requisições")) {
        // Handle rate limit: show message, wait, retry
        showRateLimitMessage();
    }
}
```

---

## Scalability Projections

### Current Performance Baseline (as of 2026-02-22)

| Page | Query Count | Load Time | API Requests | Status |
| ------ | ------------ | ----------- | ------------- | -------- |
| Contracts List | 2–3 | <500ms | **0** (cache hit) | ✅ DataContext cache |
| Financial Dashboard | 4–5 | 0.72s | **4** (was 5) | ✅ Categories cache fix |
| Clients List | 1–2 | <300ms | 2 | ✅ Optimized |
| Categories + Subcategories | 2 | <400ms | 1 | ✅ Optimized |
| Settings | 1 | <100ms | **0** (Cache-Control) | ✅ Browser cache |
| Appearance | 1 | <100ms | **0** (Cache-Control) | ✅ Browser cache |
| Audit Logs | 1 | <50ms | 1 (2 in dev) | ⚠️ StrictMode dev-only |

### Projection: 10x Data Growth (250,000 contracts)

| Scenario | Query Count | Estimated Time | Mitigation |
| ---------- | ------------ | ----------------- | ----------- |
| Financial Dashboard (no changes) | 4–5 | 0.9–1.2s | ✅ Still acceptable |
| With new indexes | 4–5 | 0.7–0.9s | ✅ Indexes scale well |
| Without pagination | 4–5 | 5–10s | ⚠️ Pagination required |

**Recommendation**: If dataset grows beyond 100,000 records, add incremental aggregation tables (materialized views) for frequently-queried date ranges.

### Projection: 100x Data Growth (2.5M+ contracts)

| Action | Impact |
| -------- | -------- |
| **Materialized Views** | Precompute monthly/annual summaries, update hourly |
| **Partitioning** | Split `financial_installments` by year, status |
| **Caching Layer** | Redis for dashboard summaries (5-min TTL) |
| **Async Aggregation** | Background job computes slow queries, caches result |

---

## How to Extend This Architecture

### Adding a New Large-Dataset Endpoint

1. **Define the schema**: How many rows? What indexes?

   ```go
   // Example: expenses table with 500k rows
   type Expense struct {
       ID         string
       ContractID string
       Date       time.Time
       Amount     float64
       Category   string
   }
   ```

2. **Design the index**:

   ```sql
   -- For common queries (by contract + date)
   CREATE INDEX idx_expenses_contract_date 
   ON expenses (contract_id, date DESC) 
   INCLUDE (amount, category);
   ```

3. **Implement batch retrieval**:

   ```go
   func (r *ExpenseStore) GetExpensesByContractIDBatch(
       ctx context.Context,
       contractIDs []string,
   ) ([]*Expense, error) { ... }
   ```

4. **Use pagination for lists**:

   ```go
   func (r *ExpenseStore) GetExpensesPaginated(
       limit, offset int,
   ) ([]*Expense, int, error) { ... }
   ```

5. **Test query performance**:

   ```bash
   EXPLAIN ANALYZE SELECT ... FROM expenses WHERE ...;
   ```

---

## Monitoring & Observability

### Backend Log Format (Fixed-Width Columnar)

All API requests are logged in a single-line, fixed-width columnar format for easy scanning:

```text
PAGE(14)       | METHOD(6) | API(35)                             | STATUS | DURATION     | user, role
/dashboard     | GET       | /api/dashboard/counts               | 200    |      6.47ms  | root, root
/financial     | GET       | /api/financial/detailed-summary     | 200    |     34.671ms | root, root
/login         | POST      | /api/login                          | 200    |     70.157ms | -, -
-              | POST      | /api/logout                         | 200    |        540µs | root, root
```

**Page source** is extracted from the `Referer` header, making it trivial to identify which frontend page triggered each request.

### Authentication Audit Trail

Login and logout events are recorded in the `audit_logs` table:

| Event | Operation | Resource | Status | Details |
| ------- | ----------- | ---------- | -------- | --------- |
| Successful login | `login` | `auth` | `success` | method=password |
| Failed login | `login` | `auth` | `failed` | reason=invalid credentials |
| Manual logout | `logout` | `auth` | `success` | method=manual |
| Token refresh failure | `login` | `auth` | `failed` | method=token |

Each entry includes: `admin_id`, `admin_username`, `ip_address`, `user_agent`, `request_method`, `request_path`.

### Slow Query Detection

```sql
-- PostgreSQL: Enable slow query logging
ALTER SYSTEM SET log_min_duration_statement = 500; -- Log queries > 500ms
SELECT pg_reload_conf();

-- View slow queries
SELECT query, mean_time, calls FROM pg_stat_statements 
WHERE mean_time > 500 
ORDER BY mean_time DESC 
LIMIT 10;
```

### Query Metrics to Track

| Metric | Target | Action if Exceeded |
| -------- | -------- | ------------------- |
| Financial dashboard query count | < 10 | Review batch loading |
| Average response time | < 1s | Check indexes, add caching |
| Slow query frequency | < 5% | Analyze query plans |
| API 429 rate-limit hits | < 1% | Increase burst or cache results |
| Pages with 0 API requests | ≥3 | DataContext + Cache-Control working |
| Redundant API calls per page | 0 | Check call signatures vs function defs |

---

## Anti-Patterns to Avoid

| Anti-Pattern | Problem | Solution |
| ------------- | --------- | ---------- |
| **SELECT * without limit** | Loads all rows, slow network | Add `LIMIT`, use pagination |
| **N+1 queries** | Exponential query count | Use batch methods |
| **App-level aggregation** | Slow, memory-heavy | Aggregate in SQL |
| **Nested loops with queries** | Query explosion | Refactor to joins or batch |
| **No indexes on filter columns** | Full table scans | Create covering indexes |
| **Hardcoded magic numbers** | Inflexible, unmaintainable | Use config constants |

---

## Summary: The Scalability Checklist

- ✅ Use batch queries (`*Batch` methods) for related data
- ✅ Aggregate in SQL (SUM, COUNT, GROUP BY) not application code
- ✅ Create covering indexes for frequently-filtered columns
- ✅ Implement pagination for any list > 100 rows
- ✅ Monitor slow queries (>500ms) and investigate
- ✅ Test query performance with `EXPLAIN ANALYZE`
- ✅ Document query assumptions (expected row count, typical filters)
- ✅ Plan for 10x growth: can current indexes handle it?
- ✅ Use `Cache-Control` headers for stable config endpoints
- ✅ Use `forceRefresh=false` for auxiliary data to leverage DataContext TTL cache
- ✅ Verify call signatures — `fn({}, false)` ≠ `fn(false)` when first param is `forceRefresh`
- ✅ Log authentication events (login/logout) in audit-logs for compliance

---

## Related Documentation

- `backend/BACKEND_ARCHITECTURE.md` — Service and handler layer
- `backend/database/QUERIES.md` — SQL query reference
- `PERFORMANCE_JOURNEY.md` — Complete performance optimization timeline
- `backend/server/middleware_logging.go` — Columnar log format implementation
- `backend/server/auth_handlers.go` — Login/logout handlers with audit logging
- `frontend/src/contexts/DataContext.jsx` — DataContext cache with TTL
- `frontend/src/utils/cacheManager.js` — CacheManager singleton

---

**Last Updated**: 2026-02-22  
**Status**: Complete, ready for extension  
**Maintainer**: DevSecOps Team
