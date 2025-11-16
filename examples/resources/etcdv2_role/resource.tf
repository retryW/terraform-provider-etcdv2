resource "etcdv2_role" "app_readonly" {
  name = "app_readonly"

  permissions = [
    {
      key_path = "/app/config"
      read     = true
      write    = false
    }
  ]
}

resource "etcdv2_role" "monitoring" {
  name = "monitoring"
  permissions = [
    {
      key_path = "/metrics"
      read     = true
      write    = false
    },
    {
      key_path = "/health"
      read     = true
      write    = false
    }
  ]
}

