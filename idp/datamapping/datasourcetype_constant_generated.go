// NOTE: automatically generated file -- DO NOT EDIT

package datamapping

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t DataSourceType) MarshalText() ([]byte, error) {
	switch t {
	case DataSourceTypeAirflow:
		return []byte("airflow"), nil
	case DataSourceTypeAthena:
		return []byte("athena"), nil
	case DataSourceTypeBigQuery:
		return []byte("bigquery"), nil
	case DataSourceTypeDynamoDB:
		return []byte("dynamodb"), nil
	case DataSourceTypeElasticsearch:
		return []byte("elasticsearch"), nil
	case DataSourceTypeFile:
		return []byte("file"), nil
	case DataSourceTypeFivetran:
		return []byte("fivetran"), nil
	case DataSourceTypeGlue:
		return []byte("glue"), nil
	case DataSourceTypeKafka:
		return []byte("kafka"), nil
	case DataSourceTypeMongoDB:
		return []byte("mongodb"), nil
	case DataSourceTypeMySQL:
		return []byte("mysql"), nil
	case DataSourceTypeOracle:
		return []byte("oracle"), nil
	case DataSourceTypeOther:
		return []byte("other"), nil
	case DataSourceTypePostgres:
		return []byte("postgres"), nil
	case DataSourceTypeRedshift:
		return []byte("redshift"), nil
	case DataSourceTypeS3Bucket:
		return []byte("s3bucket"), nil
	case DataSourceTypeSnowflake:
		return []byte("snowflake"), nil
	case DataSourceTypeSpark:
		return []byte("spark"), nil
	case DataSourceTypeTableau:
		return []byte("tableau"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown DataSourceType value '%s'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *DataSourceType) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "airflow":
		*t = DataSourceTypeAirflow
	case "athena":
		*t = DataSourceTypeAthena
	case "bigquery":
		*t = DataSourceTypeBigQuery
	case "dynamodb":
		*t = DataSourceTypeDynamoDB
	case "elasticsearch":
		*t = DataSourceTypeElasticsearch
	case "file":
		*t = DataSourceTypeFile
	case "fivetran":
		*t = DataSourceTypeFivetran
	case "glue":
		*t = DataSourceTypeGlue
	case "kafka":
		*t = DataSourceTypeKafka
	case "mongodb":
		*t = DataSourceTypeMongoDB
	case "mysql":
		*t = DataSourceTypeMySQL
	case "oracle":
		*t = DataSourceTypeOracle
	case "other":
		*t = DataSourceTypeOther
	case "postgres":
		*t = DataSourceTypePostgres
	case "redshift":
		*t = DataSourceTypeRedshift
	case "s3bucket":
		*t = DataSourceTypeS3Bucket
	case "snowflake":
		*t = DataSourceTypeSnowflake
	case "spark":
		*t = DataSourceTypeSpark
	case "tableau":
		*t = DataSourceTypeTableau
	default:
		return ucerr.Friendlyf(nil, "unknown DataSourceType value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *DataSourceType) Validate() error {
	switch *t {
	case DataSourceTypeAirflow:
		return nil
	case DataSourceTypeAthena:
		return nil
	case DataSourceTypeBigQuery:
		return nil
	case DataSourceTypeDynamoDB:
		return nil
	case DataSourceTypeElasticsearch:
		return nil
	case DataSourceTypeFile:
		return nil
	case DataSourceTypeFivetran:
		return nil
	case DataSourceTypeGlue:
		return nil
	case DataSourceTypeKafka:
		return nil
	case DataSourceTypeMongoDB:
		return nil
	case DataSourceTypeMySQL:
		return nil
	case DataSourceTypeOracle:
		return nil
	case DataSourceTypeOther:
		return nil
	case DataSourceTypePostgres:
		return nil
	case DataSourceTypeRedshift:
		return nil
	case DataSourceTypeS3Bucket:
		return nil
	case DataSourceTypeSnowflake:
		return nil
	case DataSourceTypeSpark:
		return nil
	case DataSourceTypeTableau:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown DataSourceType value '%s'", *t)
	}
}

// Enum implements Enum
func (t DataSourceType) Enum() []any {
	return []any{
		"airflow",
		"athena",
		"bigquery",
		"dynamodb",
		"elasticsearch",
		"file",
		"fivetran",
		"glue",
		"kafka",
		"mongodb",
		"mysql",
		"oracle",
		"other",
		"postgres",
		"redshift",
		"s3bucket",
		"snowflake",
		"spark",
		"tableau",
	}
}

// AllDataSourceTypes is a slice of all DataSourceType values
var AllDataSourceTypes = []DataSourceType{
	DataSourceTypeAirflow,
	DataSourceTypeAthena,
	DataSourceTypeBigQuery,
	DataSourceTypeDynamoDB,
	DataSourceTypeElasticsearch,
	DataSourceTypeFile,
	DataSourceTypeFivetran,
	DataSourceTypeGlue,
	DataSourceTypeKafka,
	DataSourceTypeMongoDB,
	DataSourceTypeMySQL,
	DataSourceTypeOracle,
	DataSourceTypeOther,
	DataSourceTypePostgres,
	DataSourceTypeRedshift,
	DataSourceTypeS3Bucket,
	DataSourceTypeSnowflake,
	DataSourceTypeSpark,
	DataSourceTypeTableau,
}
