# Query the SQLite DB

This query returns a table of manifest lists with all infos related to that.

```SQL
CREATE VIEW v_manifestlistacks AS
SELECT 
   manifestlist.id AS mlID,
	collection.repo AS collectionRepo,
	collection.tag AS collectionTag,
	profile.name AS profile,
   platform.arch AS Arch,
   platform.features AS Features,
   upstream.reg AS upstreamReg,
   upstream.repo AS upstreamRepo,
   upstream.tag AS upstreamTag
FROM collection
INNER JOIN collection_xtable
 	ON collection.id = collection_xtable.collection_id
INNER JOIN manifestlist
   ON collection_xtable.manifestlist_id = manifestlist.id
INNER JOIN manifestlist_xtable
   ON manifestlist.id = manifestlist_xtable.manifestlist_id
INNER JOIN manifest
   ON manifestlist_xtable.manifest_id = manifest.id
INNER JOIN upstream
   ON manifest.upstream_id = upstream.id
INNER JOIN platform
   ON manifest.platform_id = platform.id
INNER JOIN platform_filter
   ON platform.id = platform_filter.platform_id
INNER JOIN profile_filter
   ON platform_filter.filter_id = profile_filter.filter_id
INNER JOIN profile
   ON profile.id = profile_filter.profile_id
```