# Development Plan: Invest Web System

This plan tracks the progress of the transformation of the ML-board into a full-featured web system for securities market analysis.

## Status Legend
- [ ] Not Started
- [/] In Progress
- [x] Completed
- [!] Blocked / Issue

---

## Phase 1: Infrastructure & Backend Preparation
- [x] Fix Go and Dockerfile versions
- [x] Refactor `router.go` into multiple handlers (v1 prefix)
- [x] Add Request Logging & Error Handling Middleware
- [x] Standardize API Error Responses

## Phase 2: Authentication & Authorization
- [x] Database Schema: Users table
- [x] Password Hashing (bcrypt/argon2)
- [x] Implement Register / Login / Logout
- [x] JWT Middleware (Access & Refresh tokens)
- [x] Endpoint: `GET /api/v1/me`
- [x] Unit & Integration Tests for Auth

## Phase 3: Tinkoff API Credentials Management
- [x] Database Schema: `user_tinkoff_credentials` table
- [x] Implement AES-GCM encryption for tokens
- [x] Endpoints: Save / Test / Delete Token
- [x] Refactor `TinkoffAdapter` to be user-scoped
- [x] Masked Token Hint implementation

## Phase 4: User-Scoped Watchlist
- [x] Database Schema: Link watchlist to `user_id`
- [x] Update Repository & Service layers
- [x] Endpoints: `GET /api/v1/watchlist`, `POST`, `DELETE`
- [x] Data Isolation Validation

## Phase 5: Market Data & Analysis
- [x] Implement Data Refresh using user's token
- [x] Endpoint: `POST /api/v1/assets/{id}/refresh`
- [x] Endpoint: `GET /api/v1/assets/{id}/indicators`
- [x] Endpoint: `GET /api/v1/assets/{id}/summary`
- [x] Update Signal Endpoints

## Phase 6: Frontend Development
- [x] Setup Routing (React Router)
- [x] Login / Register Pages
- [x] Settings (Tinkoff Token) Page
- [x] Watchlist Page
- [x] Instrument Detail Page (Charts, Indicators)
- [x] Alerts Page
- [x] Admin ML Dashboard (Refactored from old ML-board)

## Phase 7: Notifications & Alerts
- [x] User-scoped Alert Rules (Schema & Service)
- [x] Frontend for Alert Rules management
- [x] Alert Events History
- [x] Alert Processing Logic

## Phase 8: Finalization & Documentation
- [x] Update README.md
- [ ] Architecture Diagrams
- [x] User Manual / Scenarios
- [ ] Final Smoke Tests & Acceptance
