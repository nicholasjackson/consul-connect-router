data_dir = "/tmp/consul"
datacenter = "dc1"
log_level = "DEBUG"
connect {
  enabled = true
  proxy {
    allow_managed_api_registration = true
    allow_managed_root = true
  }
}
