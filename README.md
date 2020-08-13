<!--
title: MinIO
weight: 4705
-->
# MinIO

**This plugin is in ALPHA stage**

This activity allows you to manage MinIO object.

## Installation

### Flogo CLI
```bash
flogo install github.com/codelity-co/flogo-minio-activity
```

## Configuration

### Settings:
  | Name                | Type   | Description
  | :---                | :---   | :---
  | endpoint            | string | The MinIO endpoint - ***REQUIRED***
  | accessKey           | string | Access Key
  | secretKey           | string | Secret Key
  | enableSSL           | bool   | Enable SSL connection
  | bucketName          | string | MinIO bucket name
  | region              | string | MinIO Region/Zone name
  | methodName          | string | MinIO SDK method name
  | methodOptions       | object | MinIO method options

### Input
  | Name                | Type   | Description
  | :---                | :---   | :---
  | objectName          | string | Minio object name - ***REQUIRED***
  | foramt              | string | json or csv
  | data                | any    | data - ***REQUIRED***

## Output
  | Name                | Type   | Description
  | :---                | :---   | :---
  | status              | string | status text - ***REQUIRED***
  | result              | object | result object - ***REQUIRED***

#### Method

MinIO provides many methods to maintain buckets. Here are the list of all avialabe methods of this plugins:

* PutObject

#### Method opitons

Method options are documented in [MinIO Golang SDK](https://docs.min.io/docs/golang-client-api-reference#).  Please reference specific method options based on your setting.

## Example

```json
{
  "id": "codelity-minio-activity",
  "name": "Codelity MinIO Activity",
  "ref": "github.com/codelity-co/flogo-minio-activity",
  "settings": {
    "endpoint" : "minio:9000",
    "accessKey": "minioadmin",
    "secretKey": "minioadmin",
    "enableSsl": false,
    "bucketName": "flogo",
    "methodName": "PutObject"
  },
  "input": {
    "objectName": "inbox/test.json",
    "data": "{\"abc\": \"123\"}"
  }
}
```