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

# Basic access key with a name
resource "garage_key" "example" {
  name = "my-application-key"
}

# Access key without a name
resource "garage_key" "unnamed" {
}

# Output the credentials
output "access_key_id" {
  value = garage_key.example.id
}

output "secret_access_key" {
  value     = garage_key.example.secret_access_key
  sensitive = true
}
