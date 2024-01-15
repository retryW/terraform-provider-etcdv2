provider "etcdv2" {
  host     = "http://localhost:4001"
  username = "myuser"
  password = "mypass"

  // Optionally ENV vars are supported in format ETCDV2_<varname>
}
