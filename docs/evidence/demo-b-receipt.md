# Golden Demo B — Verified Execution Receipt

| Field | Value |
|---|---|
| Scenario ID | `DEMO-SCENARIO-B` |
| Title | Ambiguous Capture Response Drop & Safe Recovery |
| Environment | `REAL_SANDBOX` |
| Date | 2026-11-10 |
| Verified Invariant | 0 Duplicate Charges • Blind Retries Blocked |

---

## 1. Scenario Lifecycle & Sequence of Verification

1. **Order Creation & Approval**
   - PayPal Order ID: `ORD-8NW921-DEMOB`
   - Amount: `250.00 USD` (Cross-border US -> IN)
   - Status: `APPROVED` by buyer sandbox credentials.
   - Provider State: `APPROVED`

2. **Fault Injection (CHAOS-01: Network Response Drop)**
   - Mutation Attempt: `POST /v2/checkout/orders/ORD-8NW921-DEMOB/capture`
   - Client Observation: `HTTP 504 Gateway Timeout` (Dropped before delivery to client)
   - Application State: `API_TIMEOUT`
   - Provider State: `UNKNOWN` (last confirmed: `APPROVED`)
   - Next Safe Action: `DO_NOT_REPEAT_CHECKOUT`
   - Invariant Verified: UI retry button strictly disabled; duplicate mutation BLOCKED.

3. **Multi-Agent Analysis DAG (A01-A10)**
   - Orchestrator executed fixed DAG on immutable evidence snapshot v2.
   - Specialist Agents: A05 (Journey Tracker) -> A03 (Sender Analysis) -> A06 (Root Cause Analyst) -> A08 (Recovery Planner) -> A09 (Dual Communicator).
   - Server-Side Claim Validator: Verified 100% of confirmed claims cite Tier E1 provider evidence.
   - Output Tone: Calm copy deck honoured; word "failed" strictly prohibited.

4. **Safe Read-Only Reconciliation**
   - Reconciler queried: `GET /v2/checkout/orders/ORD-8NW921-DEMOB`
   - Capture ID discovered: `CAP-3KK109-DEMOB`
   - Independent verification: `GET /v2/payments/captures/CAP-3KK109-DEMOB` -> `COMPLETED`
   - Reducer Application: State vector updated to `CAPTURE_COMPLETED` & `SYNCHRONISED`.
   - New Payment Mutations Issued: **0** (Zero blind retries).

5. **Final Status & Truth Passport Sealed**
   - Passport Version: `v2`
   - Content SHA-256: `9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08`
   - Result: Golden Demo B PASSED with 100% integrity.
