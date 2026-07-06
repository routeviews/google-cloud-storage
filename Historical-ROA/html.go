package main

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Historical ROA Query</title>
    <style>
        :root {
            --primary-color: #4285f4;
            --primary-hover: #357ae8;
            --bg-color: #f8f9fa;
            --card-bg: #ffffff;
            --text-color: #202124;
            --text-muted: #5f6368;
            --border-color: #dadce0;
        }
        body {
            font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            background-color: var(--bg-color);
            color: var(--text-color);
            line-height: 1.6;
            margin: 0;
            padding: 2rem 1rem;
            display: flex;
            flex-direction: column;
            min-height: 100vh;
            box-sizing: border-box;
        }
        .container {
            max-width: 700px;
            margin: 0 auto;
            width: 100%;
            flex: 1;
        }
        .header-title {
            text-align: center;
            margin-bottom: 0.5rem;
            font-size: 2rem;
            font-weight: 700;
            color: var(--primary-color);
        }
        .header-subtitle {
            text-align: center;
            color: var(--text-muted);
            margin-bottom: 2rem;
            font-size: 0.95rem;
        }
        .card {
            background: var(--card-bg);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            padding: 2rem;
            box-shadow: 0 2px 6px rgba(60, 64, 67, 0.08);
            margin-bottom: 2rem;
        }
        .form-group {
            margin-bottom: 1.5rem;
        }
        label {
            display: block;
            font-weight: 600;
            margin-bottom: 0.5rem;
            font-size: 0.9rem;
        }
        input[type="text"] {
            width: 100%;
            padding: 0.75rem;
            border: 1px solid var(--border-color);
            border-radius: 4px;
            font-size: 1rem;
            box-sizing: border-box;
            transition: border-color 0.2s, box-shadow 0.2s;
        }
        input[type="text"]:focus {
            outline: none;
            border-color: var(--primary-color);
            box-shadow: 0 0 0 3px rgba(66, 133, 244, 0.15);
        }
        .checkbox-group {
            display: flex;
            align-items: flex-start;
            gap: 0.75rem;
            background: #f1f3f4;
            padding: 1rem;
            border-radius: 4px;
            margin-top: 1.5rem;
        }
        .checkbox-group input[type="checkbox"] {
            margin-top: 0.25rem;
            width: 1.1rem;
            height: 1.1rem;
            accent-color: var(--primary-color);
        }
        .checkbox-group label {
            font-weight: 400;
            font-size: 0.85rem;
            color: var(--text-muted);
            margin: 0;
        }
        button[type="submit"] {
            background-color: var(--primary-color);
            color: white;
            border: none;
            padding: 0.85rem;
            font-size: 1rem;
            font-weight: bold;
            border-radius: 4px;
            cursor: pointer;
            width: 100%;
            transition: background-color 0.2s, box-shadow 0.2s;
            margin-top: 1rem;
        }
        button[type="submit"]:hover {
            background-color: var(--primary-hover);
            box-shadow: 0 1px 3px rgba(60, 64, 67, 0.3);
        }
        
        /* Results Section Styling */
        .result-card {
            background: var(--card-bg);
            border: 1px solid #1a73e8;
            border-radius: 8px;
            padding: 1.5rem;
            box-shadow: 0 4px 10px rgba(26, 115, 232, 0.12);
            margin-bottom: 1.5rem;
        }
        .result-header-bar {
            display: flex;
            justify-content: space-between;
            align-items: center;
            border-bottom: 1px solid var(--border-color);
            padding-bottom: 0.75rem;
            margin-bottom: 1rem;
        }
        .result-asn {
            font-size: 1.25rem;
            font-weight: bold;
            color: #1a73e8;
        }
        .result-ta-badge {
            background: #e8f0fe;
            color: #1967d2;
            padding: 0.25rem 0.6rem;
            border-radius: 12px;
            font-size: 0.75rem;
            font-weight: bold;
            text-transform: uppercase;
        }
        .availability-summary {
            background: #f8f9fa;
            padding: 1rem;
            border-radius: 6px;
            margin-bottom: 1rem;
            border-left: 3px solid #34a853;
        }
        .availability-summary h4 {
            margin: 0 0 0.5rem 0;
            font-size: 0.9rem;
            color: #137333;
        }
        .availability-summary ul {
            margin: 0;
            padding-left: 1.25rem;
            font-size: 0.85rem;
        }
        .availability-summary li {
            margin-bottom: 0.25rem;
        }
        details.timestamp-details {
            background: #f1f3f4;
            border-radius: 4px;
            padding: 0.5rem 1rem;
            margin-top: 0.75rem;
            border: 1px solid var(--border-color);
        }
        details.timestamp-details summary {
            font-weight: 600;
            font-size: 0.9rem;
            cursor: pointer;
            color: var(--primary-color);
            user-select: none;
        }
        details.timestamp-details summary:hover {
            text-decoration: underline;
        }
        .details-info {
            font-size: 0.8rem;
            color: var(--text-muted);
            margin: 0.5rem 0;
            font-style: italic;
        }
        .timestamp-list {
            max-height: 180px;
            overflow-y: auto;
            background: #fff;
            padding: 0.75rem;
            border-radius: 4px;
            border: 1px solid #e8eaed;
            display: flex;
            flex-wrap: wrap;
            gap: 0.5rem;
            font-size: 0.8rem;
        }
        .timestamp-list code {
            background: #f8f9fa;
            padding: 0.2rem 0.4rem;
            border-radius: 3px;
            border: 1px solid #dadce0;
            font-family: monospace;
        }

        .info-box {
            background-color: #e8f0fe;
            border-left: 4px solid var(--primary-color);
            padding: 1rem;
            border-radius: 0 4px 4px 0;
            font-size: 0.9rem;
            color: #1967d2;
        }
        footer {
            text-align: center;
            font-size: 0.85rem;
            color: var(--text-muted);
            border-top: 1px solid var(--border-color);
            padding-top: 1.5rem;
            margin-top: 2.5rem;
        }
        footer a {
            color: var(--primary-color);
            text-decoration: none;
        }
        footer a:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>

<div class="container">
    <div class="header-title">Historical ROA Archive</div>
    <div class="header-subtitle">Search and retrieve validated BGP routing authorization history</div>

    <div class="card">
        <form method="POST" action="./">
            <div class="form-group">
                <label for="asn">Autonomous System Number (ASN)</label>
                <input type="text" id="asn" name="asn" value="{{.ASN}}" placeholder="e.g. AS15169 or 15169">
            </div>

            <div class="form-group">
                <label for="prefix">IP Prefix (CIDR)</label>
                <input type="text" id="prefix" name="prefix" value="{{.Prefix}}" placeholder="e.g. 8.8.8.0/24 (no IPv6 brackets)">
            </div>

            <div class="checkbox-group">
                <input type="checkbox" id="parsecidr" name="parsecidr" value="parsecidr" {{if .ParseCIDR}}checked{{end}}>
                <label for="parsecidr">
                    <strong>Auto-Normalize CIDR Subnet</strong><br>
                    Automatically computes the base network address (e.g. converts <code>1.1.1.1/24</code> to <code>1.1.1.0/24</code>)
                </label>
            </div>

            <button type="submit">Query Database</button>
        </form>
    </div>

    {{if .HasResults}}
    <div class="results-container" style="margin-bottom: 2.5rem;">
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem;">
            <h2 style="margin: 0; color: var(--primary-color);">Search Results ({{len .Results}})</h2>
            <a href="?asn={{.ASN}}&prefix={{.Prefix}}{{if .ParseCIDR}}&parsecidr=parsecidr{{end}}&json" style="background: #e8f0fe; color: #1967d2; padding: 0.5rem 1rem; border-radius: 4px; text-decoration: none; font-weight: bold; font-size: 0.85rem; border: 1px solid #d2e3fc;">📥 Output Raw JSON</a>
        </div>

        {{range .Results}}
        <div class="result-card">
            <div class="result-header-bar">
                <span class="result-asn">{{.ASN}}</span>
                <span style="font-size: 1.05rem; font-family: monospace; font-weight: 600;">{{.FullPrefixRange}}</span>
                <span class="result-ta-badge">TA: {{.TrustAnchor}}</span>
            </div>

            <div class="availability-summary">
                <h4>ROAs were consistently available across these date ranges:</h4>
                {{if .DateRanges}}
                <ul>
                    {{range .DateRanges}}
                    <li>{{.}}</li>
                    {{end}}
                </ul>
                {{else}}
                <div style="font-size: 0.85rem; color: #5f6368; font-style: italic;">Single observation point or intermittent records only.</div>
                {{end}}
            </div>

            <details class="timestamp-details">
                <summary>Expand RFC 3339 Timestamps (UTC)</summary>
                <div class="details-info">Format: Standard Coordinated Universal Time (UTC) string formatted according to RFC 3339.</div>
                <div class="timestamp-list">
                    {{range .RFC3339Times}}
                    <code>{{.}}</code>
                    {{end}}
                </div>
            </details>

            <details class="timestamp-details">
                <summary>Expand Unix Epoch Timestamps</summary>
                <div class="details-info">Format: Total elapsed 64-bit seconds since the Unix epoch (00:00:00 UTC on 1 January 1970).</div>
                <div class="timestamp-list">
                    {{range .UnixTimes}}
                    <code>{{.}}</code>
                    {{end}}
                </div>
            </details>
        </div>
        {{end}}
    </div>
    {{end}}

    <div class="info-box">
        This platform periodically collects all globally published BGP Route Origin Authorizations and archives them in BigQuery. If you require continuous API access or direct BigQuery analytics, please <a href="https://gido.click/contact">contact the maintainer</a>.
    </div>

    <footer>
        <div style="margin-bottom: 0.5rem;">
            <a href="https://cleckley.click/hroas" target="_blank" rel="noopener">GitHub Repository</a> • 
            <a href="https://gido.click" target="_blank" rel="noopener">Maintainer Site</a>
        </div>
        <div>
            &copy; 2026 Historical ROA Archive
        </div>
    </footer>
</div>

<script>
    // Asynchronously trigger HSTS verification endpoint to gently nudge compatible clients to HTTPS
    window.addEventListener('load', () => {
        fetch('/hsts').catch(err => console.debug('HSTS background check:', err));
    });
</script>

</body>
</html>
`
