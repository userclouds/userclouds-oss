package datahub

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"reflect"
	"text/template"

	"userclouds.com/infra/multirun"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// Field represents a field in a schema
type Field struct {
	Name        string
	NativeType  string
	GeneralType string
}

// Schema represents a schema
type Schema map[string][]Field

type datahubTemplate interface {
	setFilename(filename string)
}

var postgresRecipe = template.Must(template.New("postgresRecipe").Delims("<<", ">>").Parse(`
source:
  type: postgres
  config:
    host_port: << .Host >>:<< .Port >>
    database: << .Database >>
    username: << .Username >>
    password: << .Password >>
    profiling: {'enabled': True}

sink:
  type: file
  config:
    filename: << .Filename >>`))

type postgresTemplateData struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
	Filename string
}

func (td *postgresTemplateData) setFilename(filename string) {
	td.Filename = filename
}

// ExtractSchemaFromPostgres extracts a schema from a postgres database
func ExtractSchemaFromPostgres(ctx context.Context, host string, port int, database string, username string, password string) (Schema, int, error) {
	templateData := &postgresTemplateData{
		Host:     host,
		Port:     port,
		Database: database,
		Username: username,
		Password: password,
	}

	return extractSchema(ctx, postgresRecipe, templateData)
}

var redshiftRecipe = template.Must(template.New("redshiftRecipe").Delims("<<", ">>").Parse(`
source:
  type: redshift
  config:
    host_port: << .Host >>:<< .Port >>
    database: << .Database >>
    username: << .Username >>
    password: << .Password >>
    include_copy_lineage: false
    include_operational_stats: false
    include_table_lineage: false
    include_table_location_lineage: false
    include_table_rename_lineage: false
    include_top_n_queries: false
    include_unload_lineage: false
    include_view_column_lineage: false
    include_view_lineage: false
    include_views: false
    profiling:
      enabled: true
      profile_table_level_only: true

sink:
  type: file
  config:
    filename: << .Filename >>`))

type redshiftTemplateData struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
	Filename string
}

func (td *redshiftTemplateData) setFilename(filename string) {
	td.Filename = filename
}

// ExtractSchemaFromRedshift extracts a schema from a redshift database
func ExtractSchemaFromRedshift(ctx context.Context, host string, port int, database string, username string, password string) (Schema, int, error) {
	templateData := &redshiftTemplateData{
		Host:     host,
		Port:     port,
		Database: database,
		Username: username,
		Password: password,
	}

	return extractSchema(ctx, redshiftRecipe, templateData)
}

func extractSchema(ctx context.Context, postgresRecipe *template.Template, templateData datahubTemplate) (Schema, int, error) {
	tmpDir, err := os.MkdirTemp("/tmp", "datahub-")
	if err != nil {
		return nil, 0, ucerr.Wrap(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			uclog.Errorf(ctx, "Error removing temp dir %s: %s", tmpDir, err)
		}
	}()

	outputFilename := tmpDir + "/output.json"
	templateData.setFilename(outputFilename)

	var bs []byte
	buf := bytes.NewBuffer(bs)
	if err := postgresRecipe.Execute(buf, templateData); err != nil {
		return nil, 0, ucerr.Wrap(err)
	}

	recipeFilename := tmpDir + "/recipe.yaml"
	if err := os.WriteFile(recipeFilename, buf.Bytes(), 0644); err != nil {
		return nil, 0, ucerr.Wrap(err)
	}

	if err := multirun.RunSingleCommand(ctx, "/var/app/current/bin/datahub-ingester.pex", recipeFilename); err != nil {
		return nil, 0, ucerr.Errorf("Failed to run datahub-ingester: %w", err)
	}

	output, err := os.ReadFile(outputFilename)
	if err != nil {
		return nil, 0, ucerr.Wrap(err)
	}

	schema, rows := parseSchema(output)
	return schema, rows, nil
}

func parseSchema(output []byte) (Schema, int) {
	var data any
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, 0
	}

	numRows := 0

	schema := make(Schema)
	if data, ok := data.([]any); ok {
		for _, dataset := range data {
			if dataset, ok := dataset.(map[string]any); ok {
				if snapshot := dataset["proposedSnapshot"]; snapshot != nil {
					if snapshot, ok := snapshot.(map[string]any); ok {
						snapshotDetails := snapshot["com.linkedin.pegasus2avro.metadata.snapshot.DatasetSnapshot"]
						if snapshotDetails, ok := snapshotDetails.(map[string]any); ok {
							aspects := snapshotDetails["aspects"]
							if aspects, ok := aspects.([]any); ok {
								for _, aspect := range aspects {
									if aspect, ok := aspect.(map[string]any); ok {
										if schemaMetadata := aspect["com.linkedin.pegasus2avro.schema.SchemaMetadata"]; schemaMetadata != nil {
											if schemaMetadata, ok := schemaMetadata.(map[string]any); ok {
												name, ok := schemaMetadata["schemaName"].(string)
												if !ok {
													continue
												}
												fields, ok := schemaMetadata["fields"].([]any)
												if !ok {
													continue
												}
												for _, field := range fields {
													if field, ok := field.(map[string]any); ok {
														generalTypeMap := field["type"].(map[string]any)["type"].(map[string]any)
														keys := reflect.ValueOf(generalTypeMap).MapKeys()
														generalType := keys[0].String()

														schema[name] = append(schema[name], Field{
															Name:        field["fieldPath"].(string),
															NativeType:  field["nativeDataType"].(string),
															GeneralType: generalType,
														})
													}
												}
											}
										}
									}
								}
							}
						}
					}
				} else if dataset["entityType"] == "dataset" && dataset["aspectName"] == "schemaMetadata" {
					if aspect, ok := dataset["aspect"].(map[string]any); ok {
						if j := aspect["json"]; j != nil {
							if j, ok := j.(map[string]any); ok {
								name, ok := j["schemaName"].(string)
								if !ok {
									continue
								}
								fields, ok := j["fields"].([]any)
								if !ok {
									continue
								}
								for _, field := range fields {
									if field, ok := field.(map[string]any); ok {
										generalTypeMap := field["type"].(map[string]any)["type"].(map[string]any)
										keys := reflect.ValueOf(generalTypeMap).MapKeys()
										generalType := keys[0].String()

										schema[name] = append(schema[name], Field{
											Name:        field["fieldPath"].(string),
											NativeType:  field["nativeDataType"].(string),
											GeneralType: generalType,
										})
									}
								}
							}
						}
					}
				} else if dataset["entityType"] == "dataset" && dataset["aspectName"] == "datasetProfile" {
					if aspect, ok := dataset["aspect"].(map[string]any); ok {
						if j := aspect["json"]; j != nil {
							if j, ok := j.(map[string]any); ok {
								rowCount, ok := j["rowCount"].(float64)
								if ok {
									numRows += int(rowCount)
								}
							}
						}
					}
				}
			}
		}
	}

	return schema, numRows
}
