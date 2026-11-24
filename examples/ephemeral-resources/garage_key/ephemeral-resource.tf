resource "garage_key" "example" {
  name = "my-application-key"
}

# Read the secret of the key from Garage. Terraform does not keep the
# values of an ephemeral resource in the plan or the state.
ephemeral "garage_key" "example" {
  id = garage_key.example.id
}

# Give the secret to a write-only attribute of a different provider.
resource "vault_kv_secret_v2" "example" {
  mount = "secret"
  name  = "my-application"
  data_json_wo = jsonencode({
    access_key_id     = garage_key.example.id
    secret_access_key = ephemeral.garage_key.example.secret_access_key
  })
  data_json_wo_version = 1
}
