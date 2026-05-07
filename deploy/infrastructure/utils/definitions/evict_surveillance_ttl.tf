variable "evict_surveillance_ttl" {
  type        = string
  description = <<-EOT
  How long expired surveillance items should stay before being automatically removed; expressed in Go duration format (https://pkg.go.dev/time#ParseDuration).

  EOT

  default = "30m"
}
