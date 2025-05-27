package datamapping

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
)

// DataSourceType represents the type of data source
type DataSourceType string

const (
	// DataSourceTypeFile represents a file data source
	DataSourceTypeFile DataSourceType = "file"
	// DataSourceTypePostgres represents a postgres data source
	DataSourceTypePostgres DataSourceType = "postgres"
	// DataSourceTypeRedshift represents a redshift data source
	DataSourceTypeRedshift DataSourceType = "redshift"
	// DataSourceTypeAirflow represents an airflow data source
	DataSourceTypeAirflow DataSourceType = "airflow"
	// DataSourceTypeSpark represents a spark data source
	DataSourceTypeSpark DataSourceType = "spark"
	// DataSourceTypeBigQuery represents a bigquery data source
	DataSourceTypeBigQuery DataSourceType = "bigquery"
	// DataSourceTypeSnowflake represents a snowflake data source
	DataSourceTypeSnowflake DataSourceType = "snowflake"
	// DataSourceTypeAthena represents an athena data source
	DataSourceTypeAthena DataSourceType = "athena"
	// DataSourceTypeOracle represents an oracle data source
	DataSourceTypeOracle DataSourceType = "oracle"
	// DataSourceTypeMongoDB represents a mongodb data source
	DataSourceTypeMongoDB DataSourceType = "mongodb"
	// DataSourceTypeDynamoDB represents a dynamodb data source
	DataSourceTypeDynamoDB DataSourceType = "dynamodb"
	// DataSourceTypeElasticsearch represents an elasticsearch data source
	DataSourceTypeElasticsearch DataSourceType = "elasticsearch"
	// DataSourceTypeKafka represents a kafka data source
	DataSourceTypeKafka DataSourceType = "kafka"
	// DataSourceTypeGlue represents an amazon glue data source
	DataSourceTypeGlue DataSourceType = "glue"
	// DataSourceTypeFivetran represents a fivetran data source
	DataSourceTypeFivetran DataSourceType = "fivetran"
	// DataSourceTypeMySQL represents a mysql data source
	DataSourceTypeMySQL DataSourceType = "mysql"
	// DataSourceTypeTableau represents a tableau data source
	DataSourceTypeTableau DataSourceType = "tableau"
	// DataSourceTypeS3Bucket represents an s3 bucket data source
	DataSourceTypeS3Bucket DataSourceType = "s3bucket"
	// DataSourceTypeOther represents an other data source
	DataSourceTypeOther DataSourceType = "other"
)

//go:generate genconstant DataSourceType

// DataSourceConfig represents the configuration of a data source
type DataSourceConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
}

//go:generate gendbjson DataSourceConfig

//go:generate genvalidate DataSourceConfig

// DataSourceMetadata represents the metadata of a data source
type DataSourceMetadata map[string]any

// Validate implements Validateable
func (o DataSourceMetadata) Validate() error {
	return nil
}

// Value implements sql.Valuer
func (o DataSourceMetadata) Value() (driver.Value, error) {
	return json.Marshal(o)
}

// Scan implements sql.Scanner
func (o *DataSourceMetadata) Scan(value any) error {
	b, ok := value.([]byte)
	if !ok {
		return ucerr.New("type assertion failed for DataSourceMetadata.Scan()")
	}
	return ucerr.Wrap(json.Unmarshal(b, &o))
}

// DataSource represents a data source
type DataSource struct {
	ucdb.BaseModel
	Name     string             `db:"name" json:"name" validate:"notempty"`
	Type     DataSourceType     `db:"type" json:"type"`
	Config   DataSourceConfig   `db:"config" json:"config"`
	Metadata DataSourceMetadata `db:"metadata" json:"metadata"`
}

//go:generate genvalidate DataSource

func (ds DataSource) getCursor(key pagination.Key, cursor *pagination.Cursor) {
	switch key {
	case "name,id":
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"name:%v,id:%v",
				ds.Name,
				ds.ID,
			),
		)
	case "created,id":
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"created:%v,id:%v",
				ds.Created.UnixMicro(),
				ds.ID,
			),
		)
	case "updated,id":
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"updated:%v,id:%v",
				ds.Updated.UnixMicro(),
				ds.ID,
			),
		)
	}
}

func (DataSource) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"name":                         pagination.StringKeyType,
		"created":                      pagination.TimestampKeyType,
		"updated":                      pagination.TimestampKeyType,
		"type":                         pagination.StringKeyType,
		"metadata->>'format'":          pagination.StringKeyType,
		"metadata->>'classifications'": pagination.StringKeyType,
		"metadata->>'storage'":         pagination.StringKeyType,
	}
}

//go:generate genpageable DataSource

// DataSourceElementMetadata represents the metadata of a data source element
type DataSourceElementMetadata map[string]any

// Value implements sql.Valuer
func (o DataSourceElementMetadata) Value() (driver.Value, error) {
	return json.Marshal(o)
}

// Scan implements sql.Scanner
func (o *DataSourceElementMetadata) Scan(value any) error {
	b, ok := value.([]byte)
	if !ok {
		return ucerr.New("type assertion failed for DataSourceElementMetadata.Scan()")
	}
	return ucerr.Wrap(json.Unmarshal(b, &o))
}

// Validate implements Validatable
func (o DataSourceElementMetadata) Validate() error {
	return nil
}

// DataSourceElement represents a data source element
type DataSourceElement struct {
	ucdb.BaseModel
	DataSourceID uuid.UUID                 `db:"data_source_id" json:"data_source_id" validate:"notnil"`
	Path         string                    `db:"path" json:"path"`
	Type         string                    `db:"type" json:"type"`
	Metadata     DataSourceElementMetadata `db:"metadata" json:"metadata"`
}

//go:generate genvalidate DataSourceElement

func (dse DataSourceElement) getCursor(key pagination.Key, cursor *pagination.Cursor) {
	if key == "path,id" {
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"path:%v,id:%v",
				dse.Path,
				dse.ID,
			),
		)
	}
}

func (DataSourceElement) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"path":               pagination.StringKeyType,
		"type":               pagination.StringKeyType,
		"data_source_id":     pagination.UUIDKeyType,
		"metadata->>'owner'": pagination.StringKeyType,
	}
}

//go:generate genpageable DataSourceElement
