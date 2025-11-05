terraform {
  required_providers {
    garage = {
      source = "jkossis/garage"
    }
  }
}

provider "garage" {
  endpoint = "http://localhost:3903"
  token    = "your-admin-token-here"
}

# Create a bucket
resource "garage_bucket" "example" {
  global_alias = "my-bucket"
}

# Create an access key
resource "garage_key" "example" {
  name = "my-app-key"
}

# Grant read and write permissions
resource "garage_bucket_permission" "example" {
  bucket_id     = garage_bucket.example.id
  access_key_id = garage_key.example.id
  read          = true
  write         = true
  owner         = false
}

# Example: Read-only access
resource "garage_bucket_permission" "readonly" {
  bucket_id     = garage_bucket.example.id
  access_key_id = garage_key.readonly.id
  read          = true
  write         = false
  owner         = false
}

resource "garage_key" "readonly" {
  name = "readonly-key"
}

# Example: Owner access (full permissions)
resource "garage_key" "admin" {
  name = "admin-key"
}

resource "garage_bucket_permission" "admin" {
  bucket_id     = garage_bucket.example.id
  access_key_id = garage_key.admin.id
  read          = true
  write         = true
  owner         = true
}
