resource "userclouds_userstore_column" "example" {
  name       = "example_column"
  data_type  = "string"
  is_array   = false
  index_type = "none"
}
