package main

import (
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Config struct {
	DB struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		Name     string `json:"name"`
	} `json:"db"`
}

// Dataset ...
type Dataset struct {
	gorm.Model
	RoutingCriteria        postgres.Jsonb `gorm:"column:routing_criteria"`
	ProfileKey             string         `gorm:"column:profile_key"`
	EntityType             string         `gorm:"column:entity_type"`
	EntityID               string         `gorm:"column:entity_id;unique_index:idx_entityid_rawcloudloc"`
	HiveTable              postgres.Jsonb `gorm:"column:hive_table_details"`
	RawCloudLocation       string         `gorm:"column:raw_cloud_location;unique_index:idx_entityid_rawcloudloc"`
	CompactedCloudLocation string         `gorm:"column:compacted_cloud_location"`
	Size                   uint64         `gorm:"column:size"`
	Status                 string         `gorm:"column:status"`
	StreamType             string         `gorm:"column:stream_type"`
	ExpirationDays         int            `gorm:"Column:expiration_days" json:"expirationDays" sql:"DEFAULT:1095"`
}

// TableName returned will be used to create the table in db
func (Dataset) TableName() string {
	return "datasets_new"
}

// RoutingCriteria ...
type RoutingCriteria struct {
	Project  string `json:"_tag_projectName" binding:"required"`
	App      string `json:"_tag_appName" binding:"required"`
	Plugin   string `json:"_plugin" binding:"required"`
	Document string `json:"_documentType" binding:"required"`
}

// DatasetRawDataDayWise ...
type DatasetRawDataDayWise struct {
	gorm.Model
	DatasetID   uint                       `gorm:"Column:dataset_id;uniqueIndex:dataset_year_month_day_idx"`
	Year        uint                       `gorm:"Column:year;uniqueIndex:dataset_year_month_day_idx"`
	Month       uint                       `gorm:"Column:month;uniqueIndex:dataset_year_month_day_idx"`
	Day         uint                       `gorm:"Column:day;uniqueIndex:dataset_year_month_day_idx"`
	IsProcessed bool                       `gorm:"Column:is_processed" sql:"DEFAULT:false"`
	IsCompacted bool                       `gorm:"Column:is_compacted" sql:"DEFAULT:false"`
	TimeBuckets []DatasetRawDataBucketWise `gorm:"Constraint:OnDelete:CASCADE;"`
	RawSizeMap  postgres.Jsonb             `gorm:"Column:raw_size_map;type:jsonb"`
}

// TableName ...
func (DatasetRawDataDayWise) TableName() string {
	return "dataset_raw_data_day_wise"
}

// DatasetRawDataBucketWise ...
type DatasetRawDataBucketWise struct {
	ID                      uint           `gorm:"primarykey"`
	DatasetRawDataDayWiseID uint           `gorm:"Column:dataset_raw_data_day_wise_id;uniqueIndex:parent_and_time_bucket_idx"`
	PartitionedBucket       uint64         `gorm:"Column:partitioned_bucket;uniqueIndex:parent_and_time_bucket_idx"`
	BoundaryUnixTimes       postgres.Jsonb `gorm:"Column:boundary_unix_times;type:jsonb" json:"boundary_unix_times"`
	HiveAddedPartitions     bool           `gorm:"Column:is_added_to_hive" json:"is_added_to_hive"`
	IsAnalyzed              bool           `gorm:"Column:is_analyzed" json:"is_analyzed"`
	Size                    int64          `gorm:"Column:size"`
	OrcFiles                pq.StringArray `gorm:"Column:orc_files;type:text[]"`
}

// TableName ...
func (DatasetRawDataBucketWise) TableName() string {
	return "dataset_raw_data_bucket_wise"
}

type DatasetsForOrc struct {
	ID               uint           `gorm:"column:id"`
	RawCloudLocation string         `gorm:"column:raw_cloud_location"`
	RoutingCriteria  postgres.Jsonb `gorm:"column:routing_criteria"`
}

type OrcDatasetDayId struct {
	ID    int `gorm:"column:id"`
	Year  int `gorm:"Column:year"`
	Month int `gorm:"Column:month"`
	Day   int `gorm:"Column:day"`
}

type OrcTimeBucket struct {
	PartitionedBucket int64          `gorm:"Column:partitioned_bucket;uniqueIndex:parent_and_time_bucket_idx"`
	OrcFiles          pq.StringArray `gorm:"Column:orc_files;type:text[]"`
}
