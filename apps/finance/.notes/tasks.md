# Finance App - Development Tasks & Notes

## 1. Status Terkini (Current State)

### Fixes yang Telah Selesai:
- [x] **Workspace Isolation pada Query (`WorkspaceInterceptor`)**:
  - Memperbaiki `WorkspaceInterceptor` di `internal/config/ent.go` menggunakan type switch pada query builder (`*ent.AccountQuery`, `*ent.EntryQuery`, `*ent.ExchangeRateQuery`, `*ent.LedgerPeriodQuery`, `*ent.PostingQuery`) untuk menambahkan predicate `WorkspaceIDEQ(user.WorkspaceID)`.
  - Sebelumnya interceptor menggunakan `interface{ WhereP(...) }` yang hanya ada di mutation, sehingga `SELECT` tidak memfilter `workspace_id`.
- [x] **Struct Tag Syntax Fixes (`go vet`)**:
  - Memperbaiki struct tag yang unclosed di `ent/schema/sch_user.go` dan `ent/schema/sch_workspace.go`.
  - Menghapus tag tidak valid `StructTag("parentId")` di `ent/schema/sch_account.go`.
- [x] **Account Module**:
  - Repository, Service, Controller, Casbin RBAC, dan Integration Test Suite (`tests/account_test.go`) lulus 100%.

> **Catatan Pengujian**: Saat ini `go test` harus dijalankan serial (`go test -p 1 ./...`) karena JWKS mock bind ke port tetap (`4321/9001`).

---

## 2. Roadmap & Backlog Tugas (Next Tasks)

### Task 1: Modul `LedgerPeriod` (Periode Pembukuan) - *Prioritas Utama*
Relasi `ledger_period_id` diwajibkan oleh skema `Entry`. Diperlukan periode pembukuan aktif sebelum mencatat jurnal.

- [ ] **Model DTO (`internal/model/ledger_period_model.go`)**:
  - `CreateLedgerPeriodRequest` (`start_date`, `end_date`, `status`)
  - `UpdateLedgerPeriodRequest` (perubahan status `open`, `closed`, `locked`)
  - `LedgerPeriod` response model
- [ ] **Repository (`internal/repository/ledger_period_repository.go`)**:
  - CRUD operations
  - Query mencari periode aktif berdasarkan tanggal transaksi (`start_date <= date <= end_date`)
  - Pengecekan overlap rentang tanggal periode dalam satu workspace
- [ ] **Service (`internal/service/ledger_period_service.go`)**:
  - Validasi urutan tanggal (`end_date > start_date`)
  - Pencegahan tumpang tindih tanggal antar periode
  - Validasi alur transisi status (`open` -> `closed` -> `locked`)
- [ ] **Controller & RBAC (`internal/http/controller/ledger_period_controller.go`)**:
  - Registrasi resource `ledger_periods` di `internal/shared/authz/resource.go` & `policy.csv`
  - Endpoint RESTful: `GET /ledger-periods`, `GET /:id`, `POST /`, `PUT /:id`
- [ ] **Tests (`tests/ledger_period_test.go`)**:
  - Test CRUD, validasi status, isolasi antar workspace, dan RBAC

---

### Task 2: Modul `Entry` & `Posting` (Double-Entry Journal & Ledger Engine)
Core engine akuntansi pembukuan berpasangan.

- [ ] **Model DTO (`internal/model/entry_model.go`)**:
  - `CreatePostingRequest` (`account_id`, `currency`, `debit_amount`, `credit_amount`, `memo`)
  - `CreateEntryRequest` (`entry_date`, `entry_type`, `description`, `reference`, `postings []CreatePostingRequest`)
  - Response models untuk Entry & detail Postings
- [ ] **Service & Accounting Validation (`internal/service/entry_service.go`)**:
  - Aturan keseimbangan: `sum(debit) == sum(credit)` (zero-sum balance)
  - Validasi minimal 2 posting line per entry
  - Validasi tanggal jurnal berada dalam `LedgerPeriod` yang berstatus `open`
  - Validasi bahwa akun tujuan aktif (`status == active`), menolak posting ke akun `archived`
- [ ] **Repository (`internal/repository/entry_repository.go`)**:
  - Pembuatan Entry + Postings secara atomik dalam satu transaksi database (`WithTx`)
  - Query list jurnal dengan pagination/cursor & filter tanggal/tipe
- [ ] **Controller & RBAC (`internal/http/controller/entry_controller.go`)**:
  - Endpoint `POST /entries` (buat jurnal transaksi)
  - Endpoint `GET /entries` & `GET /entries/:id`
- [ ] **Tests (`tests/entry_test.go`)**:
  - Pengujian jurnal seimbang vs tidak seimbang
  - Penolakan posting pada periode terkunci/tutup
  - Pengujian transaksi atomik rollback jika salah satu baris posting gagal

---

### Task 3: Modul `ExchangeRate` (Mata Uang & Kurs Valas)
Mendukung transaksi multi-currency dalam pembukuan.

- [ ] **Model DTO (`internal/model/exchange_rate_model.go`)**:
  - Request/Response kurs mata uang
- [ ] **Repository & Service (`internal/repository/` & `internal/service/`)**:
  - CRUD kurs per tanggal efektif (`from_currency`, `to_currency`, `rate`, `effective_date`)
  - Helper konversi nilai valas ke `base_currency` pada baris `Posting`
- [ ] **Controller & Tests (`tests/exchange_rate_test.go`)**

---

### Task 4: Developer Experience & Test Runner Improvements
- [ ] **Dynamic Port untuk JWKS Mock**:
  - Modifikasi `testutil/jwks.go` menggunakan listener dinamis (`httptest.NewServer`) alih-alih port statis `4321/9001` agar `go test ./...` dapat dijalankan paralel tanpa bentrok port.
- [ ] **Makefile / Taskfile**:
  - Tambahkan shortcut:
    - `make test` -> `go test -count=1 -p 1 ./...`
    - `make vet` -> `go vet ./...`
    - `make gen` -> `go generate ./ent`
