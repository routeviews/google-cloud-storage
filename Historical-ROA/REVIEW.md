# Historical-ROA Codebase Analysis

## Intent and Purpose

The `Historical-ROA` service is an App Engine application built in Go, purposed with tracking continuous routing security data—specifically, Resource Public Key Infrastructure (RPKI) Route Origin Authorizations (ROAs). 

**Key Goals Accompanied by the Source:**
1. **Background Update/Pull Job:** A scheduled job via App Engine cron (`/update` via `cron.yaml`) downloads hourly full ROA datasets from a centralized repository (`https://hosted-routinator.rarc.net/json`).
2. **Diff & Tracking Mechanism:** Rather than maintaining independent full tables for every hourly observation, the system maintains a consolidated dataset in BigQuery (`roas_arr`). It stores an array of observation timestamps against discrete combinations of `(ASN, Prefix, Mask, MaxLength, TA)`.
3. **Data Availability:** It provides an interactive form and endpoint (`/` with POST inputs) for researchers or operators to query historical data, receiving parsed Protobuf output mapping the results.

---

## Overall Efficacy

The core logic theoretically achieves the goal of reducing long-term duplicate object retention (merging discrete state variables in BigQuery via array tracking). However, it presents significant risks to production operation (breaking Google App Engine standards, brittle HTTP transport tricks, zero authentication on mutating handlers, and aggressive use of panics).

### Architectural and Defect Summary

#### 1. Inappropriate Asynchronous Transport Management
- **Line References**: `main.go` (around L306-L309)
- To unblock the cron request timer and force execution in the background without task queues, the author manually closes the TCP socket behind the router:
  ```go
  http.Error(w, "can't hijack rw", 200)
  hj, _ := w.(http.Hijacker)
  conn, _, _ := hj.Hijack()
  conn.Close()
  ```
  **Vulnerability:** Behind Google App Engine (or any reverse-proxy / GFE configuration), manually hijacking the connection breaks protocol streams (like HTTP/2 frames) and is explicitly not supported on many managed runtime instances. Proper asynchronous execution should leverage Cloud Tasks or Pub/Sub triggers.

#### 2. Flawed TLS/HSTS Imposition
- **Line References**: `main.go` and `index.html`
- Instead of deploying global TLS intercepting middleware across the router, the author injects an unusual client-side loop via `onload`, launching an async GET request to a standalone `/hsts` handler that redirects.
  **Remediation:** Inject modern middleware directly over `http.HandleFunc` invocations to enforce `http.Redirect` and inject strict-transport header signatures consistently.

#### 3. Inefficient Schema Migration and BigQuery Overhead
- **BigQuery Scratch Volatility:** The update cycle unconditionally drops and reinstantiates temporary synchronization tables (`buf`) directly in BigQuery. Frequent internal schema adjustments over large BigQuery analytics projects can trip metadata quotas and slow down insert triggers.
- **Concurrency Violations:** Without transactional isolation, multiple updates running at exactly the same hour (due to manual HTTP triggering by a malicious or errant user hitting `/update`) would destructively replace each other's `buf` tables causing severe state corruption.

#### 4. Insecure Exposure and Denial of Service Vulnerability
- **Open HTTP Controller (`/update`):** The trigger is publicly reachable. The only defensive check is a BigQuery poll to assert if the last modification time is under 50 minutes. Continuous brute forcing of this routine exhausts CPU capacity parsing massive memory blocks from remote JSON arrays.

#### 5. Error Management Practices
- **Non-Professional Commit Content:** Code documentation lacks professional standards, containing negative editorializations and unchecked panics.
  ```go
  Mask:   int32(row[2].(int64)), // stupid
  Maxlen: int32(row[3].(int64)), // I hate you,
  Ta:     row[4].(string),       // Google
  ```
- Panics via `log.Fatal`/`log.Fatalln` are widely unmitigated throughout local operations; missing schema rows, cast failures, or network faults will forcibly terminate the container process under live handling conditions.

---

## Recommendation

If long-term sustainability of this tool is required:
1. **Decouple Scheduling:** Trigger the task asynchronously using GCP Cloud Scheduler emitting into GCP Cloud Pub/Sub.
2. **Transport Redesign:** Refactor ingestion away from synchronous App Engine workers toward a standalone batch utility, saving API cost overhead.
3. **Hardening:** Enforce structured IAM validations upon invocation paths, eliminating raw exposure to publicly facing mutation requests.
