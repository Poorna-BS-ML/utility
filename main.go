package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	_ "github.com/lib/pq"
	"gorm.io/gorm"
)

var firstWriteToFile bool

func main() {
	config := InitConfig()
	db := InitDB(config)
	firstWriteToFile = true
	postgresOrcEntries(db)
}

func postgresOrcEntries(db *gorm.DB) {
	todaysYear, todaysMonth, todaysDay := time.Now().Date()
	yesterday := time.Now().AddDate(0, 0, -1)

	var orcDatasets []DatasetsForOrc
	err := db.Model(Dataset{}).Select("id", "raw_cloud_location", "routing_criteria").Find(&orcDatasets).Error
	if err != nil {
		fmt.Println("error while fetching the id from datasets_new", err)
		os.Exit(1)
	}
	for _, datasetId := range orcDatasets {
		rawLocationString := strings.SplitAfter(datasetId.RawCloudLocation, "/")
		bucket := strings.Trim(rawLocationString[2], "/")
		var orcDatasetDayId []OrcDatasetDayId

		err := db.Model(DatasetRawDataDayWise{}).Select("id", "year", "month", "day").Where("dataset_id = ? and year=? and month=? and day=? ", datasetId.ID, todaysYear, int(todaysMonth), todaysDay).Or("dataset_id = ? and year=? and month=? and day=? ", datasetId.ID, yesterday.Year(), int(yesterday.Month()), yesterday.Day()).Find(&orcDatasetDayId).Error
		if err != nil {
			fmt.Println("error while fetching the id,year,month,day from dataset_raw_data_day_wise", err)
			os.Exit(1)
		}
		rawPrefix := strings.TrimPrefix(datasetId.RawCloudLocation, ("s3a://"+bucket+"/")) + "/"

		var rc RoutingCriteria
		jsonErr := json.Unmarshal(datasetId.RoutingCriteria.RawMessage, &rc)
		if jsonErr != nil {
			fmt.Println("Error while json unmarshalling: ", jsonErr)
			os.Exit(1)
		}

		for _, dayId := range orcDatasetDayId {
			yearMonthDayPrefix := "year=" + strconv.Itoa(dayId.Year) + "/" + "month=" + strconv.Itoa(dayId.Month) + "/" + "day=" + strconv.Itoa(dayId.Day) + "/"
			var orcTimeBuckets []OrcTimeBucket

			err := db.Model(DatasetRawDataBucketWise{}).Select("partitioned_bucket", "orc_files").Where("dataset_raw_data_day_wise_id = ?", dayId.ID).Order("partitioned_bucket").Find(&orcTimeBuckets).Error
			if err != nil {
				fmt.Println("error while fetching the partion-buckets and orcfiles from dataset_raw_data_bucket_wise", err)
				os.Exit(1)
			}
			length := len(orcTimeBuckets)
			if dayId.Day == todaysDay {
				length = len(orcTimeBuckets) - 2
			}
			for _, timeBucket := range orcTimeBuckets {
				if length > 0 {
					var report string
					rawBucketPrefix := rawPrefix + yearMonthDayPrefix + "time-bucket=" + strconv.FormatInt(timeBucket.PartitionedBucket, 10) + "/"
					var dbOrcNamesList []string
					for _, orc := range timeBucket.OrcFiles {
						res := strings.Split(orc, "+")
						dbOrcNamesList = append(dbOrcNamesList, res[0]+"+"+res[1])
					}
					orcNameListFromS3 := s3OrcEntries(bucket, rawBucketPrefix)
					s3vsdborcs, noOfDuplicatesinDB := compareOrcEntries(orcNameListFromS3, dbOrcNamesList)
					if s3vsdborcs {
						fmt.Println("all the orcs which are in S3 are in DB")
					} else {
						fmt.Println("all the orcs which are in S3 are not in DB")
					}
					report = fmt.Sprintf("%d-%d-%d,  %s/%s/%s/%s, %d, %d, %s, %d, %d, %d, %t\n", dayId.Day, dayId.Month, dayId.Year, rc.Project, rc.App, rc.Plugin, rc.Document, datasetId.ID, timeBucket.PartitionedBucket, time.UnixMilli(timeBucket.PartitionedBucket), len(orcNameListFromS3), len(dbOrcNamesList), noOfDuplicatesinDB, s3vsdborcs)
					writeToFile(report)
				}
				length = length - 1
			}
		}
	}
}

func s3OrcEntries(bucket string, bucketPrefix string) []string {
	var orcNameList []string
	OrcCount := 0
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		//Config:            aws.Config{Region: aws.String("us-west-2")},
	}))
	s3client := s3.New(sess)
	params := &s3.ListObjectsV2Input{
		Bucket: &bucket,
		Prefix: &bucketPrefix,
	}
	listUrl := getListFiles(bucket, s3client, params)
	for _, file := range listUrl {
		if strings.HasSuffix(file, ".orc") {
			orcName := strings.SplitAfter(file, "/")
			res := strings.TrimSuffix(orcName[len(orcName)-1], ".orc")
			splitRes := strings.Split(res, "+")
			orcNameList = append(orcNameList, splitRes[1]+"+"+splitRes[2])
			OrcCount += 1
		}
	}

	fmt.Printf("orc count in bucket %s%s is: %d ", bucket, bucketPrefix, OrcCount)
	return orcNameList
}
func compareOrcEntries(orcNameListFromS3 []string, dbOrcNamesList []string) (bool, int) {
	uniqueDBOrc, noOfDuplicatesinDB := unique(dbOrcNamesList)
	less := func(a, b string) bool { return a < b }
	if cmp.Diff(orcNameListFromS3, uniqueDBOrc, cmpopts.SortSlices(less)) == "" {
		return true, noOfDuplicatesinDB
	} else {
		return false, noOfDuplicatesinDB
	}
}

func getListFiles(bucket string, svc *s3.S3, params *s3.ListObjectsV2Input) []string {
	var listUrl []string
	truncatedListing := true
	for truncatedListing {
		resp, err := svc.ListObjectsV2(params)
		if err != nil {
			fmt.Printf("Unable to list items in bucket %q, %v", bucket, err)
		}
		for _, key := range resp.Contents {
			fmt.Printf("  🔍 Loading... \r")
			listUrl = append(listUrl, *key.Key)
		}
		params.ContinuationToken = resp.NextContinuationToken
		truncatedListing = *resp.IsTruncated
	}
	return listUrl
}

func writeToFile(line string) {

	f, err := os.OpenFile("s3vsdb-Report.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	w := bufio.NewWriter(f)

	if firstWriteToFile {
		w.WriteString("\n")
		w.WriteString(fmt.Sprintf("Time of this report: %s\n", time.Now().String()))
		w.WriteString("Date, RoutingCriteria, DatasetId, TimeBucketInEpoch, TimeBucketHumanReadable, S3RawOrcCount, DBRawOrcCount, NoOfDuplicatesinDB, S3vsDBSync\n")
		fmt.Println(line)
		w.WriteString(line)
		firstWriteToFile = false
	} else {
		fmt.Println(line)
		w.WriteString(line)
	}

	w.Flush()
}

func unique(arr []string) ([]string, int) {
	occurred := map[string]bool{}
	result := []string{}
	dup := []string{}
	for e := range arr {
		if !occurred[arr[e]] {
			occurred[arr[e]] = true
			result = append(result, arr[e])
		} else {
			dup = append(dup, arr[e])
		}
	}
	fmt.Printf("found %d duplicates: %s", len(dup), dup)
	return result, len(dup)
}
