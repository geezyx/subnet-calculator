# The following example shows how to calculate the next available subnet CIDR
# from an available pool of CIDR blocks, and a list of alread-used CIDR blocks.

resource "subnet_calculator_subnet" "example" {
  available_cidr_blocks = ["10.0.0.0/16"]
  used_cidr_blocks      = []
  network_size          = 24
}
