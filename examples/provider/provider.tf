terraform {
  required_providers {
    garage = {
      source = "jkossis/garage"
    }
  }
}

provider "garage" {
  endpoint = "http://localhost:3903"
  token    = "your-garage-admin-token"
}
