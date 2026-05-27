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

# Create an access key
resource "garage_key" "example" {
  name = "my-app-key"
}

# Look up access key by ID
data "garage_key" "example" {
  id = garage_key.example.id
}

# Use data source output
output "key_info" {
  value = {
    id            = data.garage_key.example.id
    name          = data.garage_key.example.name
    expired       = data.garage_key.example.expired
    created       = data.garage_key.example.created
    expiration    = data.garage_key.example.expiration
    create_bucket = data.garage_key.example.create_bucket
  }
}
