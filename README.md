This utility is used to know if all the raw ORCs in S3 are present in the DB.
If any duplicates are present in the DB, those will be identified.

Finally, a report s3vsdb-Report.txt will be generated which contains information in the following format:

Date, RoutingCriteria, DatasetId, TimeBucketInEpoch, TimeBucketHumanReadable, S3RawOrcCount, DBRawOrcCount, NoOfDuplicatesinDB, S3vsDBSync

Requirements:
aws has to be configured (using "aws configure")

How to run:
./main