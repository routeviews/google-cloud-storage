# Cloud storage mass uploader

Compare an FTP archive to a cloud storage bucket, copy files which are either
not included in the cloud storage bucket OR where the MD5 mismatches from the
ftp archive to the cloud storage bucket.

# Operations

## Compile Code

```shell
$ go build -o mass_uploader mass_uploader.go
```

## Create Service Account

Create a service account to be used in uploading and querying the cloud storage
resources. Go to the [IAM -> ServiceAccounts]
(https://console.cloud.google.com/iam-admin/serviceaccounts) page, create a new
service account.

Create a new key for the service-account, download the JSON format key material.
Store this key material securely on the machine which will be performing the
synchronization.

Grant the service account the following roles/permissions:

   * Cloud Run Invoker - configured in the CloudRun instance permissions.
   * Storage Object Creator - configured in IAM permissions.
   * Storage Object Viewer - configured in IAM permissions.

## Run the Upload Process

Run the program, provide the bucket and ftp archive as flags, provide the key
file location through the environment variable: GOOGLE_APPLICATION_CREDENTIALS.

```shell
$ $ GOOGLE_APPLICATION_CREDENTIALS=<filesystem_path_to_key> mass_upload -bucket routeviews-archives -archive ftp://archive.routeviews.org/bgpdata
```

## Review Logs

Review logged data for errors, address as required.
