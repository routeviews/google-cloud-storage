This document contains examples of accessing Route Views data that is available
in Google BigQuery.

# Prerequisites

There are some prerequisites to accessing this data.
You will need to:

1. Have a [Google Cloud account to access BigQuery](https://cloud.google.com/bigquery).
2. Request access to the data set (process TBD, contact "CODEOWNERS")
3. Establish the data set in your BigQuery project, discussed below.

## Establish BigQuery data set 

To access the Route Views data in BigQuery, it needs to be added to your project.

1. Open your project in the [BigQuery Console](https://console.cloud.google.com/bigquery)
2. "ADD DATA" -> "Pin a project" -> "Enter project Name" 
    * *Project name:* "public-routing-data-backup"

# Examples

Below are some practical examples of accessing Route Views data in Google BigQuery.

## Find updates whose AS Paths include both 15169 and 36040

    SELECT Announced,
           Attributes,
           Collector,
           ARRAY(SELECT asn from UNNEST(JSON_QUERY_ARRAY(attr.Payload, "$.as_paths")) as segment,
           UNNEST(JSON_VALUE_ARRAY(segment, "$.asns")) as asn)
    FROM `public-routing-data-backup.historical_routing_data.updates`,
     UNNEST(Attributes) as attr WHERE DATE(SeenAt) = "2021-11-02"
     AND attr.AttrType = 2
     AND EXISTS(
         SELECT *
            from UNNEST(JSON_QUERY_ARRAY(attr.Payload, "$.as_paths")) as segment,
            UNNEST(JSON_VALUE_ARRAY(segment, "$.asns")) as asn
         WHERE asn = "15169"
         )
     AND EXISTS(
         SELECT *
            from UNNEST(JSON_QUERY_ARRAY(attr.Payload, "$.as_paths")) as segment,
            UNNEST(JSON_VALUE_ARRAY(segment, "$.asns")) as asn
         WHERE asn = "36040"
         )
    LIMIT 10;

## Find update whose first ASN in AS Path (the peer) is 3280 in an AS_SEQ

    SELECT Announced, Attributes, Collector
    FROM `public-routing-data-backup.historical_routing_data.updates`,
     UNNEST(Attributes) as attr WHERE DATE(SeenAt) = "2021-11-02"
     AND attr.AttrType = 2
     AND ARRAY_LENGTH(JSON_QUERY_ARRAY(attr.Payload, "$.as_paths")) > 0
     AND JSON_VALUE(JSON_QUERY_ARRAY(attr.Payload, "$.as_paths")[OFFSET(0)], "$.segment_type") = "2"
     AND EXISTS (
         SELECT * from UNNEST(JSON_QUERY_ARRAY(attr.Payload, "$.as_paths")) as segment
         WHERE ARRAY_LENGTH(JSON_QUERY_ARRAY(segment, "$.asns")) > 0
         AND JSON_QUERY_ARRAY(segment, "$.asns")[OFFSET(0)] = "3280"
     )
    LIMIT 10;

## Find update whose last segment is AS_SET

    SELECT Announced, Attributes, Collector
    FROM `public-routing-data-backup.historical_routing_data.updates`,
     UNNEST(Attributes) as attr WHERE DATE(SeenAt) = "2021-11-02"
     AND attr.AttrType = 2

     AND ARRAY_LENGTH(JSON_QUERY_ARRAY(attr.Payload, "$.as_paths")) > 0
     AND JSON_VALUE(
         JSON_QUERY_ARRAY(attr.Payload, "$.as_paths")[OFFSET(
             ARRAY_LENGTH(JSON_QUERY_ARRAY(attr.Payload, "$.as_paths"))-1
         )], "$.segment_type") = "2"
    LIMIT 10;

## Exact match of 104.237.172.0/24

    CREATE TEMP FUNCTION IP(raw STRING)
      RETURNS BYTES
      AS (NET.IP_FROM_STRING(SUBSTR(raw, 0, STRPOS(raw, "/")-1)));

    CREATE TEMP FUNCTION PLEN(raw STRING)
      RETURNS INT64
      AS (CAST(SUBSTR(raw, STRPOS(raw, "/")+1) AS INT64));


    WITH input AS (
        SELECT 4 AS afi, # 16 bytes if it's IPv6
                NET.IP_FROM_STRING("104.237.172.0") AS ip,
                24 AS mask
    )
    SELECT Announced, r.Attributes
        FROM `public-routing-data-backup.historical_routing_data.updates` as r, input as i
         WHERE DATE(SeenAt) = "2021-11-02" AND EXISTS(
            SELECT * FROM UNNEST(Announced) as prefix
            WHERE BYTE_LENGTH(IP(prefix)) = i.afi
                    # Change the "=" to ">" or "<" for more-specific or less-specific match.
                    AND PLEN(prefix) = i.mask
                    AND (NET.IP_NET_MASK(i.afi, i.mask) & IP(prefix))
                    = (NET.IP_NET_MASK(i.afi, i.mask) & i.ip)
        )
    LIMIT 10;

## Find any announcements that have more-specific match of 2c0f:fb50::/32

    CREATE TEMP FUNCTION IP(raw STRING)
      RETURNS BYTES
      AS (NET.IP_FROM_STRING(SUBSTR(raw, 0, STRPOS(raw, "/")-1)));

    CREATE TEMP FUNCTION PLEN(raw STRING)
      RETURNS INT64
      AS (CAST(SUBSTR(raw, STRPOS(raw, "/")+1) AS INT64));


    WITH input AS (
        SELECT 16 AS afi,
                NET.IP_FROM_STRING("2c0f:fb50::") AS ip,
                30 AS mask
    )
    SELECT Announced, Attributes
        FROM `public-routing-data-backup.historical_routing_data.updates` as r, input as i,
        UNNEST(Attributes) as attr
        WHERE DATE(SeenAt) = "2021-11-02"
        AND attr.AttrType = 14 # MP-REACH
        AND EXISTS(
            SELECT * FROM UNNEST(JSON_QUERY_ARRAY(attr.Payload, "$.value")) as e
            WHERE BYTE_LENGTH(IP(JSON_VALUE(e, "$.prefix"))) = i.afi
                    # Change the "=" to ">" or "<" for more-specific or less-specific match.
                    AND PLEN(JSON_VALUE(e, "$.prefix")) > i.mask
                    AND (NET.IP_NET_MASK(i.afi, i.mask) & IP(JSON_VALUE(e, "$.prefix")))
                    = (NET.IP_NET_MASK(i.afi, i.mask) & i.ip)
        )
    LIMIT 10;
