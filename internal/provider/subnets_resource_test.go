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
				resource "netcalc_subnets" "test" {
					pool_cidr_blocks = ["10.0.0.0/16"]
					cidr_mask_length = 24
					cidr_count       = 3
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnets.test", "id", "10.0.0.0/24,10.0.1.0/24,10.0.2.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.1", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.2", "10.0.2.0/24"),
				),
			},
			// Changing cidr_count causes recalculation
			{
				Config: `
				resource "netcalc_subnets" "test" {
					pool_cidr_blocks     = ["10.0.0.0/16"]
					existing_cidr_blocks = ["10.0.0.0/24"]
					cidr_mask_length     = 24
					cidr_count           = 1
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnets.test", "id", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "10.0.1.0/24"),
				),
			},
			// Only available CIDR blocks are chosen
			{
				Config: `
				resource "netcalc_subnets" "test" {
					pool_cidr_blocks     = ["10.0.0.0/16"]
					existing_cidr_blocks = ["10.0.0.0/24","10.0.2.0/24","10.0.3.0/24","10.0.4.128/25","10.0.6.0/24"]
					cidr_mask_length     = 24
					cidr_count           = 3
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnets.test", "id", "10.0.1.0/24,10.0.5.0/24,10.0.7.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.1", "10.0.5.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.2us", "10.0.7.0/24"),
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
				resource "netcalc_subnets" "test" {
					pool_cidr_blocks = ["10.0.0.0/16"]
					cidr_mask_length = 24
					cidr_count       = 1
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnets.test", "id", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "10.0.0.0/24"),
				),
			},
			// Updating the existing cidr blocks should not cause changes
			{
				Config: `
				resource "netcalc_subnets" "test" {
					pool_cidr_blocks = ["10.0.0.0/16"]
					existing_cidr_blocks = ["10.0.0.0/24"]
					cidr_mask_length = 24
					cidr_count       = 1
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnets.test", "id", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "10.0.0.0/24"),
				),
			},
			// Changing the CIDR block count should cause a recalculation
			{
				Config: `
				resource "netcalc_subnets" "test" {
					pool_cidr_blocks     = ["10.0.0.0/16"]
					existing_cidr_blocks = ["10.0.0.0/24"]
					cidr_mask_length     = 24
					cidr_count           = 3
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnets.test", "id", "10.0.1.0/24,10.0.2.0/24,10.0.3.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.1", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.2", "10.0.3.0/24"),
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
				resource "netcalc_subnets" "test" {
					pool_cidr_blocks = ["10.0.0.0/16"]
					existing_cidr_blocks = ["10.0.0.0/24"]
					cidr_mask_length = 24
					cidr_count       = 2
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnets.test", "id", "10.0.1.0/24,10.0.2.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.1", "10.0.2.0/24"),
				),
			},
			// Adding to the pool_cidr_blocks or existing_cidr_blocks does not cause recalculation
			{
				Config: `
				resource "netcalc_subnets" "test" {
					pool_cidr_blocks     = ["10.0.0.0/16", "10.1.0.0/16"]
					existing_cidr_blocks = ["10.0.0.0/24","10.0.2.0/24","10.0.3.0/24","10.0.4.0/24","10.0.5.0/24"]
					cidr_mask_length     = 24
					cidr_count           = 2
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnets.test", "id", "10.0.1.0/24,10.0.2.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.1", "10.0.2.0/24"),
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
				resource "netcalc_subnets" "test" {
					pool_cidr_blocks = ["10.0.0.0/16"]
					cidr_mask_length = 24
					cidr_count       = 1
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnets.test", "id", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "10.0.0.0/24"),
				),
			},
			// Removing an available CIDR that contained the calculated CIDR
			// should cause the resource to be replaced.
			{
				Config: `
				resource "netcalc_subnets" "test" {
					pool_cidr_blocks = ["192.168.0.0/16"]
					cidr_mask_length = 24
					cidr_count       = 1
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnets.test", "id", "192.168.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "192.168.0.0/24"),
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
				resource "netcalc_subnets" "test" {
					pool_cidr_blocks = ["fd18:fad4:bce5:4400::/56"]
					cidr_mask_length = 64
					cidr_count       = 3
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnets.test", "id", "fd18:fad4:bce5:4400::/64,fd18:fad4:bce5:4401::/64,fd18:fad4:bce5:4402::/64"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "fd18:fad4:bce5:4400::/64"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.1", "fd18:fad4:bce5:4401::/64"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.2", "fd18:fad4:bce5:4402::/64"),
				),
			},
			// Update and Read testing
			{
				Config: `
				resource "netcalc_subnets" "test" {
					pool_cidr_blocks     = ["fd18:fad4:bce5:4400::/56"]
					existing_cidr_blocks = ["fd18:fad4:bce5:4400::/64","fd18:fad4:bce5:4401::/64","fd18:fad4:bce5:4402::/64"]
					cidr_mask_length     = 64
					cidr_count           = 1
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnets.test", "id", "fd18:fad4:bce5:4403::/64"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "fd18:fad4:bce5:4403::/64"),
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
				resource "netcalc_subnets" "test" {
					pool_cidr_blocks = ["fd18:fad4:bce5:4400::/56"]
					cidr_mask_length = 64
					cidr_count       = 1
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnets.test", "id", "fd18:fad4:bce5:4400::/64"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "fd18:fad4:bce5:4400::/64"),
				),
			},
			// Update and Read testing
			{
				Config: `
				resource "netcalc_subnets" "test" {
					pool_cidr_blocks = ["fd18:fad4:bce5:5500::/56"]
					cidr_mask_length = 64
					cidr_count       = 1
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnets.test", "id", "fd18:fad4:bce5:5500::/64"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "fd18:fad4:bce5:5500::/64"),
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
				resource "netcalc_subnets" "test" {
					pool_cidr_blocks = ["10.0.0.0/16"]
					cidr_mask_length = 24
					cidr_count       = 3
				}
				resource "netcalc_subnets" "test_chained" {
					pool_cidr_blocks     = ["10.0.0.0/16"]
					existing_cidr_blocks = netcalc_subnets.test.cidr_blocks
					cidr_mask_length     = 24
					cidr_count           = 3
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnets.test", "id", "10.0.0.0/24,10.0.1.0/24,10.0.2.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.1", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.2", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test_chained", "id", "10.0.3.0/24,10.0.4.0/24,10.0.5.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test_chained", "cidr_blocks.0", "10.0.3.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test_chained", "cidr_blocks.1", "10.0.4.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test_chained", "cidr_blocks.2", "10.0.5.0/24"),
				),
			},
			// Update and Read testing
			{
				Config: `
				resource "netcalc_subnets" "test" {
					pool_cidr_blocks     = ["10.0.0.0/16"]
					existing_cidr_blocks = ["10.0.0.0/24"]
					cidr_mask_length     = 24
					cidr_count           = 1
				}
				resource "netcalc_subnets" "test_chained" {
					pool_cidr_blocks     = ["10.0.0.0/16"]
					existing_cidr_blocks = netcalc_subnets.test.cidr_blocks
					cidr_mask_length     = 24
					cidr_count           = 3
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnets.test", "id", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test_chained", "id", "10.0.3.0/24,10.0.4.0/24,10.0.5.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test_chained", "cidr_blocks.0", "10.0.3.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test_chained", "cidr_blocks.1", "10.0.4.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test_chained", "cidr_blocks.2", "10.0.5.0/24"),
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
				resource "netcalc_subnets" "test" {
					pool_cidr_blocks = ["10.0.0.0/16"]
					cidr_mask_length = 24
					cidr_count       = 3
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnets.test", "id", "10.0.0.0/24,10.0.1.0/24,10.0.2.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.1", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.2", "10.0.2.0/24"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "netcalc_subnets.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"pool_cidr_blocks", "existing_cidr_blocks"},
			},
			// Change the pool CIDR blocks after import:
			{
				Config: `
				resource "netcalc_subnets" "test" {
					pool_cidr_blocks     = ["10.2.0.0/16"]
					cidr_mask_length     = 24
					cidr_count           = 3
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnets.test", "id", "10.2.0.0/24,10.2.1.0/24,10.2.2.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "10.2.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.1", "10.2.1.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.2", "10.2.2.0/24"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "netcalc_subnets.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"pool_cidr_blocks", "existing_cidr_blocks"},
			},
			// Change the existing CIDR blocks after import:
			{
				Config: `
				resource "netcalc_subnets" "test" {
					pool_cidr_blocks     = ["10.2.0.0/16"]
					existing_cidr_blocks = ["10.2.0.0/24","10.2.1.0/24","10.2.2.0/24"]
					cidr_mask_length     = 24
					cidr_count           = 3
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnets.test", "id", "10.2.0.0/24,10.2.1.0/24,10.2.2.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "10.2.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.1", "10.2.1.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.2", "10.2.2.0/24"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "netcalc_subnets.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"pool_cidr_blocks", "existing_cidr_blocks"},
			},
		},
	})
}
