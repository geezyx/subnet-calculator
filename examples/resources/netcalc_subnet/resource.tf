# By default, an IPv4 CIDR block is allocated.
resource "netcalc_subnet" "example" {
  cidr_mask_length = 22
}

# Resources in the same terraform apply will not allocate
# overlapping CIDR blocks. Resources that are created in
# separate applies rely on the existing usage being reported
# to the provider configuration.
resource "netcalc_subnet" "example_27" {
  cidr_mask_length = 27
}

# IPv6 subnets can be allocated by specifying the IP family.
resource "netcalc_subnet" "example_ipv6" {
  ip_family        = "ipv6"
  cidr_mask_length = 64
}
