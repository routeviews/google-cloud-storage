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

    SELECT Announced, Attributes.ASPath
    FROM `public-routing-data-backup.historical_routing_data.updates`
    WHERE DATE(SeenAt) = "2021-11-02" AND (
        36040 IN (
        SELECT asn
            FROM UNNEST(Attributes.ASPath) AS segment
                CROSS JOIN UNNEST(segment.ASList) AS asn
        )
    AND 15169 IN (
        SELECT asn
            FROM UNNEST(Attributes.ASPath) AS segment 
                CROSS JOIN UNNEST(segment.ASList) AS asn
                )
        )
    LIMIT 100;

## Find update whose first ASN in AS Path (the peer) is 3280 in an AS_SEQ

    SELECT Announced, Attributes.ASPath
    FROM `public-routing-data-backup.historical_routing_data.updates` AS r
    WHERE ARRAY_LENGTH(r.Attributes.ASPath) > 0
    AND ARRAY_LENGTH(r.Attributes.ASPath[OFFSET(0)].ASList) > 0
    AND r.Attributes.ASPath[OFFSET(0)].Type = 2
    AND r.Attributes.ASPath[OFFSET(0)].ASList[OFFSET(0)] = 3280
    AND DATE(SeenAt) = "2021-11-02"
    LIMIT 100;

## Find update whose last segment is AS_SET

    SELECT Announced, r.Attributes.ASPath
    FROM `public-routing-data-backup.historical_routing_data.updates` as r
    WHERE ARRAY_LENGTH(r.Attributes.ASPath) > 0
    AND r.Attributes.ASPath[OFFSET(ARRAY_LENGTH(r.Attributes.ASPath)-1)].Type = 1
    AND DATE(SeenAt) = "2021-11-02"
    LIMIT 100;

## Exact match of 104.237.172.0/24

    WITH input AS (
        SELECT 4 AS afi, # 16 bytes if it's IPv6
            "104.237.172.0" AS ip,
            24 AS mask
    )
    SELECT Announced, r.Attributes.ASPath
    FROM `public-routing-data-backup.historical_routing_data.updates` as r, input as i
    WHERE (
        SELECT COUNT(*) FROM UNNEST(r.announced) as announcedPrefix
        WHERE
                BYTE_LENGTH(NET.IP_FROM_STRING(announcedPrefix.ip)) = i.afi
            # Change this to ">" or "<" for more-specific or less-specific match.
            AND announcedPrefix.mask = i.mask
            AND (NET.IP_NET_MASK(i.afi, i.mask) & NET.IP_FROM_STRING(announcedPrefix.ip))
                = (NET.IP_NET_MASK(i.afi, i.mask) & NET.IP_FROM_STRING(i.ip))
        ) > 0
    LIMIT 100;

## Find any announcements that have more-specific match of 2c0f:fb50::/32

    WITH input AS (
        SELECT 16 AS afi,
            "2c0f:fb50::" AS ip,
            32 AS mask
    )
    SELECT Announced, r.Attributes.ASPath
    FROM `public-routing-data-backup.historical_routing_data.updates` AS r, input AS i
    WHERE (
        SELECT COUNT(*) FROM UNNEST(r.announced) as announcedPrefix
            WHERE
                BYTE_LENGTH(NET.IP_FROM_STRING(announcedPrefix.ip)) = i.afi
            AND announcedPrefix.mask > i.mask
            AND (NET.IP_NET_MASK(i.afi, i.mask) & NET.IP_FROM_STRING(announcedPrefix.ip))
                    = (NET.IP_NET_MASK(i.afi, i.mask) & NET.IP_FROM_STRING(i.ip))
        ) > 0
    LIMIT 100;
