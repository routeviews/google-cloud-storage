# Find updates whose AS Paths include both 15169 and 36040
SELECT Announced, Attributes.ASPath FROM `netmm-rpki.routeviews.updates`
WHERE DATE(SeenAt) = "2021-11-02" AND (
        36040 IN (SELECT asn FROM UNNEST(Attributes.ASPath) as segment CROSS JOIN UNNEST(segment.ASList) as asn) AND
        15169 IN (SELECT asn FROM UNNEST(Attributes.ASPath) as segment CROSS JOIN UNNEST(segment.ASList) as asn)
    ) limit 100;

# Find update whose first ASN in AS Path (the peer) is 3280 in an AS_SEQ
SELECT Announced, Attributes.ASPath FROM `netmm-rpki.routeviews.updates` as r
    WHERE ARRAY_LENGTH(r.Attributes.ASPath) > 0
    and ARRAY_LENGTH(r.Attributes.ASPath[OFFSET(0)].ASList) > 0
    and r.Attributes.ASPath[OFFSET(0)].Type = 2
    and r.Attributes.ASPath[OFFSET(0)].ASList[OFFSET(0)] = 3280
    and DATE(SeenAt) = "2021-11-02"
    limit 100;

# Find update whose last segment is AS_SET
SELECT Announced, r.Attributes.ASPath FROM `netmm-rpki.routeviews.updates` as r
    WHERE ARRAY_LENGTH(r.Attributes.ASPath) > 0
    and r.Attributes.ASPath[OFFSET(ARRAY_LENGTH(r.Attributes.ASPath)-1)].Type = 1
    and DATE(SeenAt) = "2021-11-02"
    limit 100;


# Exact match of 104.237.172.0/24
WITH input AS (
    SELECT 4 AS afi, # 16 bytes if it's IPv6
    "104.237.172.0" AS ip,
    24 AS mask
)
SELECT Announced, r.Attributes.ASPath FROM `netmm-rpki.routeviews.updates` as r, input as i
    WHERE (
        SELECT COUNT(*) FROM UNNEST(r.announced) as announcedPrefix
        WHERE
            BYTE_LENGTH(NET.IP_FROM_STRING(announcedPrefix.ip)) = i.afi
            AND announcedPrefix.mask = i.mask # Change this to ">" or "<" for more-specific or less-specific match.
            AND (NET.IP_NET_MASK(i.afi, i.mask) & NET.IP_FROM_STRING(announcedPrefix.ip))
                 = (NET.IP_NET_MASK(i.afi, i.mask) & NET.IP_FROM_STRING(i.ip))
    ) > 0
    limit 100;

# Find any announcements that have more-specific match of 2c0f:fb50::/32
WITH input AS (
    SELECT 16 AS afi,
    "2c0f:fb50::" AS ip,
    32 AS mask
)
SELECT Announced, r.Attributes.ASPath FROM `netmm-rpki.routeviews.updates` as r, input as i
    WHERE (
        SELECT COUNT(*) FROM UNNEST(r.announced) as announcedPrefix
        WHERE
            BYTE_LENGTH(NET.IP_FROM_STRING(announcedPrefix.ip)) = i.afi
            AND announcedPrefix.mask > i.mask
            AND (NET.IP_NET_MASK(i.afi, i.mask) & NET.IP_FROM_STRING(announcedPrefix.ip))
                 = (NET.IP_NET_MASK(i.afi, i.mask) & NET.IP_FROM_STRING(i.ip))
    ) > 0 limit 100;
