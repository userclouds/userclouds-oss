package auditlog

// CustomFilterConfigChange is a custom filter for auditlog entries to only return config changes
const CustomFilterConfigChange = "(type LIKE 'Create%' OR type LIKE 'Update%')"
