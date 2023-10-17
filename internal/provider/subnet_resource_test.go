// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSubnetResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: `
				provider "netcalc" {
					pool_cidr_blocks = ["10.0.0.0/16"]
				}
				resource "netcalc_subnet" "test" {
					cidr_mask_length = 24
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "10.0.0.0/24"),
				),
			},
			// Changing cidr_mask_length causes recalculation
			{
				Config: `
				provider "netcalc" {
					pool_cidr_blocks = ["10.0.0.0/16"]
				}
				resource "netcalc_subnet" "test" {
					cidr_mask_length = 25
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "10.0.0.0/25"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "10.0.0.0/25"),
				),
			},
			// Only available CIDR blocks are chosen
			{
				Config: `
				provider "netcalc" {
					pool_cidr_blocks    = ["10.0.0.0/16"]
					claimed_cidr_blocks = ["10.0.0.0/25","10.0.2.0/24","10.0.3.0/24","10.0.4.128/25","10.0.6.0/24"]
				}
				resource "netcalc_subnet" "test" {
					count = 4
					cidr_mask_length = 24
				}
				output "ids" {
					value = join(",", sort(netcalc_subnet.test.*.id))
				}
				output "subnets" {
					value = join(",", sort(netcalc_subnet.test.*.cidr_block))
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckOutput("ids", "10.0.0.0/24,10.0.1.0/24,10.0.5.0/24,10.0.7.0/24"),
					resource.TestCheckOutput("subnets", "10.0.0.0/24,10.0.1.0/24,10.0.5.0/24,10.0.7.0/24"),
				),
			},
			// CIDR blocks from the appropriate family are chosen
			{
				Config: `
				provider "netcalc" {
					pool_cidr_blocks = ["10.0.0.0/16", "fd18:fad4:bce5:4400::/56"]
				}
				resource "netcalc_subnet" "ipv4" {
					cidr_mask_length = 24
				}
				resource "netcalc_subnet" "ipv6" {
					ip_family        = "ipv6"
					cidr_mask_length = 64
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.ipv4", "id", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnet.ipv4", "cidr_block", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnet.ipv6", "id", "fd18:fad4:bce5:4400::/64"),
					resource.TestCheckResourceAttr("netcalc_subnet.ipv6", "cidr_block", "fd18:fad4:bce5:4400::/64"),
				),
			},
		},
	})
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: `
				provider "netcalc" {
					pool_cidr_blocks = ["10.0.0.0/16"]
				}
				resource "netcalc_subnet" "test" {
					cidr_mask_length = 24
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "10.0.0.0/24"),
				),
			},
			// Updating the existing cidr blocks should not cause changes
			{
				Config: `
				provider "netcalc" {
					pool_cidr_blocks    = ["10.0.0.0/16"]
					claimed_cidr_blocks = ["10.0.0.0/24"]
				}
				resource "netcalc_subnet" "test" {
					cidr_mask_length = 24
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "10.0.0.0/24"),
				),
			},
			// Changing the CIDR block size should cause a recalculation
			{
				Config: `
				provider "netcalc" {
					pool_cidr_blocks    = ["10.0.0.0/16"]
					claimed_cidr_blocks = ["10.0.0.0/24"]
				}
				resource "netcalc_subnet" "test" {
					cidr_mask_length = 22
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "10.0.0.0/22"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "10.0.0.0/22"),
				),
			},
		},
	})
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: `
				provider "netcalc" {
					pool_cidr_blocks = ["10.0.0.0/16"]
				}
				resource "netcalc_subnet" "test" {
					cidr_mask_length = 24
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "10.0.0.0/24"),
				),
			},
			// Adding to the pool_cidr_blocks does not cause recalculation
			{
				Config: `
				provider "netcalc" {
					pool_cidr_blocks = ["10.0.0.0/16", "10.1.0.0/16"]
				}
				resource "netcalc_subnet" "test" {
					cidr_mask_length = 24
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "10.0.0.0/24"),
				),
			},
		},
	})
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: `
				provider "netcalc" {
					pool_cidr_blocks = ["10.0.0.0/16"]
				}
				resource "netcalc_subnet" "test" {
					cidr_mask_length = 24
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "10.0.0.0/24"),
				),
			},
			// Removing an available CIDR that contained the calculated CIDR
			// should cause the resource to be recalculated.
			{
				Config: `
				provider "netcalc" {
					pool_cidr_blocks = ["192.168.0.0/16"]
				}
				resource "netcalc_subnet" "test" {
					cidr_mask_length = 24
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "192.168.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "192.168.0.0/24"),
				),
			},
		},
	})
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: `
				provider "netcalc" {
					pool_cidr_blocks = ["fd18:fad4:bce5:4400::/56"]
				}
				resource "netcalc_subnet" "test" {
					ip_family        = "ipv6"
					cidr_mask_length = 64
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "fd18:fad4:bce5:4400::/64"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "fd18:fad4:bce5:4400::/64"),
				),
			},
			// Update and Read testing
			{
				Config: `
				provider "netcalc" {
					pool_cidr_blocks    = ["fd18:fad4:bce5:4400::/56"]
					claimed_cidr_blocks = ["fd18:fad4:bce5:4400::/64"]
				}
				resource "netcalc_subnet" "test" {
					ip_family        = "ipv6"
					cidr_mask_length = 64
				}
				resource "netcalc_subnet" "test2" {
					ip_family        = "ipv6"
					cidr_mask_length = 64
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "fd18:fad4:bce5:4400::/64"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "fd18:fad4:bce5:4400::/64"),
					resource.TestCheckResourceAttr("netcalc_subnet.test2", "id", "fd18:fad4:bce5:4401::/64"),
					resource.TestCheckResourceAttr("netcalc_subnet.test2", "cidr_block", "fd18:fad4:bce5:4401::/64"),
				),
			},
			// Only available CIDR blocks are chosen
			// CIDR blocks which are deleted are then made available
			{
				Config: `
				provider "netcalc" {
					pool_cidr_blocks    = ["fd18:fad4:bce5:4400::/56", "10.0.0.0/8"]
					claimed_cidr_blocks = ["fd18:fad4:bce5:4400::/64","fd18:fad4:bce5:4401::/64"]
				}
				resource "netcalc_subnet" "many" {
					count            = 200
					ip_family        = "ipv6"
					cidr_mask_length = 64
				}
				resource "netcalc_subnet" "many_ipv4" {
					count            = 200
					ip_family        = "ipv4"
					cidr_mask_length = 28
				}
				output "distinct_count" {
					value = length(distinct(flatten([
						netcalc_subnet.many.*.cidr_block,
						netcalc_subnet.many_ipv4.*.cidr_block,
					])))
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckOutput("distinct_count", "400"),
				),
			},
		},
	})
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a typical config
			{
				Config: `
				provider "netcalc" {
					pool_cidr_blocks = ["10.0.0.0/16"]
				}
				resource "netcalc_subnet" "test" {
					cidr_mask_length = 24
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "10.0.0.0/24"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "netcalc_subnet.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Change the CIDR blocks after import:
			{
				Config: `
				provider "netcalc" {
					pool_cidr_blocks = ["192.168.0.0/16"]
				}
				resource "netcalc_subnet" "test" {
					cidr_mask_length = 22
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "192.168.0.0/22"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "192.168.0.0/22"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "netcalc_subnet.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Change the existing CIDR blocks after import:
			{
				Config: `
				provider "netcalc" {
					pool_cidr_blocks    = ["192.168.0.0/16"]
					claimed_cidr_blocks = ["192.168.0.0/22"]
				}
				resource "netcalc_subnet" "test" {
					cidr_mask_length = 22
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "192.168.0.0/22"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "192.168.0.0/22"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "netcalc_subnet.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
