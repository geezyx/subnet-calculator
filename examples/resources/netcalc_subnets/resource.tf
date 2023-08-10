# Calculated unused IPv4 subnet CIDR for use in an AWS VPC
# using associated VPC CIDRs and existing subnets in the VPC.
resource "netcalc_subnets" "example" {
  pool_cidr_blocks     = local.vpc_cidrs
  existing_cidr_blocks = local.existing_subnet_cidrs
  cidr_mask_length     = 22
  cidr_count           = 3
}

# Chain netcalc_subnets resources together for calculating
# different sized CIDR blocks. Use shorter mask lengths first
# for optimal CIDR allocation.
resource "netcalc_subnets" "example_chained" {
  pool_cidr_blocks = local.vpc_cidrs
  existing_cidr_blocks = concat(
    local.existing_subnet_cidrs,
    netcalc_subnets.example.cidr_blocks,
  )
  cidr_mask_length = 27
  cidr_count       = 3
}

locals {
  # Existing IPv4 VPC CIDR blocks in the "associated" state.
  vpc_cidrs = [
    for association in data.aws_vpc.vpc.cidr_block_associations :
    association.cidr_block if association.state == "associated"
  ]
  # IPv4 CIDR blocks in use by existing subnets.
  existing_subnet_cidrs = [
    for k, v in data.aws_subnet.subnet :
    v.cidr_block
  ]
}

data "aws_vpc" "vpc" {
  id = var.vpc_id
}

data "aws_subnets" "vpc_subnets" {
  filter {
    name   = "vpc-id"
    values = [var.vpc_id]
  }
}

data "aws_subnet" "subnet" {
  for_each = toset(data.aws_subnets.vpc_subnets.ids)

  id = each.key
}
