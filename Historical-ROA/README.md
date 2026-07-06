# Historical-ROA

[![Codecov](https://img.shields.io/codecov/c/github/morrowc/Historical-ROA)](https://app.codecov.io/gh/morrowc/Historical-ROA)

This repository contains a Google App Engine application designed to collect,
archive, and serve historical Route Origin Authorizations (ROAs) data.

## What the Application Does

The `historical-roa` service serves two primary functions:

1.  **Data Ingestion & Archival (`/update`)**:

    *   Periodically fetches current ROA data from an external source
        (defaulting to `https://hosted-routinator.rarc.net/json`).
    *   Compares the fetched ROAs with previously stored data in Google Cloud
        BigQuery.
    *   Inserts new records and updates observation timestamps for existing
        ROAs, effectively creating a historical timeline of ROA changes.
    *   Uses a buffer table (`historical.buf`) and a `MERGE` operation in
        BigQuery to efficiently deduplicate and append data.
    *   Includes a cooldown mechanism (50 minutes) to prevent redundant updates
        if triggered too frequently.

2.  **Data Querying / User Interface (`/`)**:

    *   Provides a web interface (`index.html`) allowing users to query the
        archived historical ROA data.
    *   Users can search by Autonomous System Number (ASN), IP Prefix, or both.
    *   Returns structured JSON (via Protocol Buffers) detailing the ROAs found,
        including their maximum length, Trust Anchor (TA), and a list of
        timestamps when this ROA state was observed.

3.  **Security (`/hsts`)**:

    *   Enforces HTTP Strict Transport Security (HSTS) and redirects HTTP
        requests to HTTPS.

### Helper: `roa_proxy`

The repository includes a sub-package `roa_proxy`, which is a standalone HTTP
proxy service. If the primary App Engine application experiences access or
reliability issues fetching directly from upstream sources (like
`https://hosted-routinator.rarc.net/json`), `roa_proxy` can be deployed on an
external cluster to reliably bridge and proxy the JSON data back to Google
Cloud.

## Architecture

*   **Runtime**: Go 1.26 (Google App Engine Standard Environment).
*   **Storage**: Google Cloud BigQuery (`historical` dataset, `roas_arr` table).
*   **Ingestion Trigger**: Can be triggered via App Engine Cron or external
    schedulers (e.g., Cloud Scheduler) authenticated with OIDC tokens.

## Prerequisites for Deployment

Before deploying, ensure you have the following configured in your Google Cloud
Platform (GCP) project:

1.  **GCP Project**: A valid GCP project with billing enabled.
2.  **App Engine**: App Engine initialized in your project.
3.  **BigQuery**:

    *   A dataset named `historical` must exist in your target location (default
        `us-central2` in code).
    *   The primary table `roas_arr` must be created. You can use the following
        schema (as derived from `main.go`):

        ```sql
        CREATE TABLE `your-project-id.historical.roas_arr` (
            asn STRING,
            prefix STRING,
            maxlen INT64,
            ta STRING,
            mask INT64,
            inserttimes ARRAY<TIMESTAMP>
        )
        CLUSTER BY prefix, mask, asn;
        ```

    *   The application will automatically create and delete a temporary `buf`
        table in the same dataset during updates.

4.  **Service Account & Permissions**:

    *   The App Engine default service account (or a custom one if configured)
        needs permissions to:
        *   Run BigQuery jobs (`roles/bigquery.jobUser`).
        *   Read and write data to the `historical` BigQuery dataset
            (`roles/bigquery.dataEditor`).

## How to Deploy

1.  **Clone the repository**:

    ```bash
    git clone <repository-url>
    cd Historical-ROA
    ```

2.  **Configure `app.yaml`** (if necessary):

    *   The default `app.yaml` is set up for Go 1.26 and `instance_class: F4`.
        Adjust as needed based on expected load.

3.  **Deploy to App Engine**:

    ```bash
    gcloud app deploy
    ```

4.  **Set up Ingestion Scheduling**: You must configure a periodic job to hit
    the `https://<your-service-url>/update` endpoint (e.g., every hour).

    *   **Google Cloud Scheduler (Recommended)**: Set up a Cloud Scheduler job
        targeting your `/update` endpoint. Configure the job to use an OIDC
        token for authentication. You can set the following environment
        variables in your `app.yaml` to secure the endpoint and restrict which
        service account can trigger updates:

        *   `SCHEDULE_AUDIENCE`: The audience string (usually your full target
            URL).
        *   `SCHEDULE_SERVICE_ACCOUNT`: The exact email of the service account
            configured in Cloud Scheduler.
        *   If `SCHEDULE_SERVICE_ACCOUNT` is not set, the application will
            default to allowing any service account belonging to the same GCP
            project (verified via the `GOOGLE_CLOUD_PROJECT` environment
            variable).

    *   **App Engine Cron (Deprecated)**: *Note: App Engine Cron is currently
        being deprecated in favor of Cloud Scheduler.* If you must maintain
        legacy deployments, you can create a `cron.yaml` file in the root
        directory:

        ```yaml
        cron:
        - description: "Hourly ROA update"
          url: /update
          schedule: every 1 hours
        ```

        And deploy it: `gcloud app deploy cron.yaml`

## How to Test

### Local Unit Tests

The repository includes standard Go unit tests. Run them using:

```bash
go test -v ./...
```

*   **`main_test.go`**: Tests XSS escaping in error handlers, OIDC/Cron
    authentication rejection logic for the `/update` endpoint, and general error
    response formatting.
*   **`roa_proxy/main_test.go`**: Tests the proxy service functionality,
    including error handling, timeout handling, and successful proxying.

### Integration / BigQuery Testing

*   Currently, the unit tests do not mock BigQuery interactions.
*   To test the full ingestion flow (`/update`), it is recommended to deploy to
    a staging GCP project with a dedicated BigQuery dataset, or verify locally
    by setting the `GOOGLE_APPLICATION_CREDENTIALS` environment variable and
    temporarily adjusting the BigQuery project/dataset target in the code or via
    environment variables (if implemented in future updates).

## Management and Monitoring

*   **App Engine Logs**: Monitor the `historical-roa` service logs in the GCP
    Console (Cloud Logging) for errors during the `/update` process or high
    latency on the query interface.
*   **BigQuery Usage**:
    *   Monitor the storage size and costs of the `historical.roas_arr` table.
    *   The `MERGE` query runs periodically; keep an eye on BigQuery slot
        utilization and query costs.
    *   Ensure the clustering on `(prefix, mask, asn)` remains effective for
        your common query patterns.
*   **External Source Health**: The application relies on the configured
    `roaURL` (`https://hosted-routinator.rarc.net/json`). If this upstream
    endpoint fails or blocks Google Cloud execution, the `/update` process will
    abort and log errors. Monitor for consistent 500 errors from `/update`,
    which indicate an issue with the upstream ROA data source or a network
    egress failure.

