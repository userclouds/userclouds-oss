/* type ActivityRecord struct {
	InstanceID           uuid.UUID      `json:"instance_id"`
	LastTenantID         uuid.UUID      `json:"last_tenant_id"`
	MultiTenant          bool           `json:"multitenant"`
	Service              string         `json:"service"`
	Region               string         `json:"region"`
	CodeVersion          string         `json:"code_version"`
	StartupTime          time.Time      `json:"startup_time"`
	EventMetadataVersion int            `json:"event_metadata_version"`
	LastActivity         time.Time      `json:"last_activity"`
	CallCount            int            `json:"call_count"`
	EventCount           int            `json:"event_count"`
	SendRawData          bool           `json:"send_raw_data"`
	LogLevel             uclog.LogLevel `json:"log_level"`
	MessageInterval      int            `json:"message_interval"`
	CountersInterval     int            `json:"counters_interval"`
	ProcessedStartup     bool           `json:"processed_startup"`
	mutex                sync.Mutex
}
*/
type ActiveInstance = {
  instance_id: string;
  service: string;
  startup_time: string;
  last_activity: string;
  event_count: string;
  error_input_count: string;
  error_internal_count: string;
  healthy: boolean;
};

export default ActiveInstance;
