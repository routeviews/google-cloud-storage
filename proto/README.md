## FileRequest Specification.
The following attributes should be included in the grpc FileRequest message

 1. name: `filename`  
    type: `string`  
    description: `The full path of the file from the rsync top directory.`  
    example1:   
     ```
     path: rsync://archive.routeviews.org/bgpdata/2021.03/UPDATES/update.20210331.2345.bz2  
     becomes: /bgpdata/2021.03/UPDATES/updates.20210331.2345.bz2
     ```  
    example2:  
     ```
     path: rsync://archive.routeviews.org/route-views.amsix/bgpdata/2021.03/UPDATES/update.20210331.2345.bz2  
     becomes: /route-views.amsix/bgpdata/2021.03/UPDATES/updates.20210331.2345.bz2
     ```
 2. name: `sha256`  
    type: `string`  
    description: `A sha256 checksum of the content field (The metadata already includes an md5_hash?)`  
 3. name: `content`  
    type: `bytes`  
    description: `The actual MRT RIB or UPDATE bzipped file content, as bytes`  
 4. name: `convert_sql`  
    type: `bool`  
    description: `Whether or not to convert the file to SQL for BigQuery. not all files uploaded show be converted`
 5. name: `project`  
    type: `string`  
    description: `A project name that idenifies where the data is coming from, e.g RouteViews, RIS, Isolario, etc.`  
,
