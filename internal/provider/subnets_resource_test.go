// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"errors"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func isUUID(value string) error {
	re := regexp.MustCompile(`^[0-9a-fA-F]{8}\b-[0-9a-fA-F]{4}\b-[0-9a-fA-F]{4}\b-[0-9a-fA-F]{4}\b-[0-9a-fA-F]{12}$`)
	if !re.Match([]byte(value)) {
		return errors.New("value is not UUID")
	}
	return nil
}

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
					resource.TestCheckResourceAttrWith("netcalc_subnets.test", "id", isUUID),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.1", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.2", "10.0.2.0/24"),
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
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith("netcalc_subnets.test", "id", isUUID),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "10.0.0.0/24"),
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
					resource.TestCheckResourceAttrWith("netcalc_subnets.test", "id", isUUID),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "10.0.0.0/24"),
				),
			},
			// Update and Read testing
			{
				Config: `
				resource "netcalc_subnets" "test" {
					pool_cidr_blocks     = ["10.0.0.0/16"]
					existing_cidr_blocks = ["10.0.0.0/24"]
					cidr_mask_length     = 24
					cidr_count           = 3
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith("netcalc_subnets.test", "id", isUUID),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.1", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.2", "10.0.2.0/24"),
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
					resource.TestCheckResourceAttrWith("netcalc_subnets.test", "id", isUUID),
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
					resource.TestCheckResourceAttrWith("netcalc_subnets.test", "id", isUUID),
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
					resource.TestCheckResourceAttrWith("netcalc_subnets.test", "id", isUUID),
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
					resource.TestCheckResourceAttrWith("netcalc_subnets.test", "id", isUUID),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "fd18:fad4:bce5:4400::/64"),
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
					resource.TestCheckResourceAttrWith("netcalc_subnets.test", "id", isUUID),
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
					resource.TestCheckResourceAttrWith("netcalc_subnets.test", "id", isUUID),
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
					resource.TestCheckResourceAttrWith("netcalc_subnets.test", "id", isUUID),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.1", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.2", "10.0.2.0/24"),
					resource.TestCheckResourceAttrWith("netcalc_subnets.test_chained", "id", isUUID),
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
					cidr_count           = 4
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith("netcalc_subnets.test", "id", isUUID),
					resource.TestCheckResourceAttr("netcalc_subnets.test", "cidr_blocks.0", "10.0.0.0/24"),
					resource.TestCheckResourceAttrWith("netcalc_subnets.test_chained", "id", isUUID),
					resource.TestCheckResourceAttr("netcalc_subnets.test_chained", "cidr_blocks.0", "10.0.3.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test_chained", "cidr_blocks.1", "10.0.4.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test_chained", "cidr_blocks.2", "10.0.5.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnets.test_chained", "cidr_blocks.3", "10.0.1.0/24"),
				),
			},
		},
	})
}

// ImportState testing
// {
// 	ResourceName:      "scaffolding_example.test",
// 	ImportState:       true,
// 	ImportStateVerify: true,
// 	// This is not normally necessary, but is here because this
// 	// example code does not have an actual upstream service.
// 	// Once the Read method is able to refresh information from
// 	// the upstream service, this can be removed.
// 	ImportStateVerifyIgnore: []string{"configurable_attribute", "defaulted"},
// },
// Update and Read testing
// {
// 	Config: testAccExampleResourceConfig("two"),
// 	Check: resource.ComposeAggregateTestCheckFunc(
// 		resource.TestCheckResourceAttr("scaffolding_example.test", "configurable_attribute", "two"),
// 	),
// },
// Delete testing automatically occurs in TestCase
