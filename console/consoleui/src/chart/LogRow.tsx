/* // CounterData provides data for row in the event log
type CounterRow struct {
	ID        uint64 `json:"id"`
	EventName string `json:"event_name"`
	EventType string `json:"event_type"`
	Service   string `json:"service"`
	Timestamp int64  `json:"timestamp"`
	Count     int    `json:"count"`
} */
type LogRow = {
  id: string;
  service: string;
  event_name: string;
  event_type: string;
  timestamp: string;
  count: string;
};

export default LogRow;
