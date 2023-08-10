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
				resource "netcalc_subnet" "test" {
					available_cidr_blocks = ["10.0.0.0/16"]
					network_size          = 24
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "10.0.0.0/24"),
				),
			},
			// Update and Read testing
			{
				Config: `
				resource "netcalc_subnet" "test" {
					available_cidr_blocks = ["10.0.0.0/16"]
					used_cidr_blocks      = ["10.0.0.0/24"]
					network_size          = 24
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "10.0.0.0/24"),
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
				resource "netcalc_subnet" "test" {
					count                 = 4
					available_cidr_blocks = ["10.0.0.0/16"]
					network_size          = 24
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test.0", "cidr_block", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnet.test.0", "id", "10.0.0.0/24"),
					// TODO: count/for_each should support keeping some known state to generate unique CIDRs.
					// resource.TestCheckResourceAttr("netcalc_subnet.test.1", "cidr_block", "10.0.1.0/24"),
					// resource.TestCheckResourceAttr("netcalc_subnet.test.1", "id", "10.0.1.0/24"),
					// resource.TestCheckResourceAttr("netcalc_subnet.test.2", "cidr_block", "10.0.2.0/24"),
					// resource.TestCheckResourceAttr("netcalc_subnet.test.2", "id", "10.0.2.0/24"),
					// resource.TestCheckResourceAttr("netcalc_subnet.test.3", "cidr_block", "10.0.3.0/24"),
					// resource.TestCheckResourceAttr("netcalc_subnet.test.3", "id", "10.0.3.0/24"),
				),
			},
			// Update and Read testing
			{
				Config: `
				resource "netcalc_subnet" "test" {
					available_cidr_blocks = ["10.0.0.0/16"]
					used_cidr_blocks      = ["10.0.0.0/24"]
					network_size          = 24
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "10.0.0.0/24"),
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
				resource "netcalc_subnet" "test" {
					available_cidr_blocks = ["10.0.0.0/16"]
					network_size          = 24
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "10.0.0.0/24"),
				),
			},
			// Removing an available CIDR that contained the calculated CIDR
			// should cause the resource to be replaced.
			{
				Config: `
				resource "netcalc_subnet" "test" {
					available_cidr_blocks = ["192.168.0.0/16"]
					network_size          = 24
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
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
				resource "netcalc_subnet" "test" {
					available_cidr_blocks = ["fd18:fad4:bce5:4400::/56"]
					network_size          = 64
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "fd18:fad4:bce5:4400::/64"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "fd18:fad4:bce5:4400::/64"),
				),
			},
			// Update and Read testing
			{
				Config: `
				resource "netcalc_subnet" "test" {
					available_cidr_blocks = ["fd18:fad4:bce5:4400::/56"]
					used_cidr_blocks      = ["fd18:fad4:bce5:4400::/64"]
					network_size          = 64
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "fd18:fad4:bce5:4400::/64"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "fd18:fad4:bce5:4400::/64"),
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
				resource "netcalc_subnet" "test" {
					available_cidr_blocks = ["fd18:fad4:bce5:4400::/56"]
					network_size          = 64
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "fd18:fad4:bce5:4400::/64"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "fd18:fad4:bce5:4400::/64"),
				),
			},
			// Update and Read testing
			{
				Config: `
				resource "netcalc_subnet" "test" {
					available_cidr_blocks = ["fd18:fad4:bce5:5500::/56"]
					network_size          = 64
				  }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcalc_subnet.test", "cidr_block", "fd18:fad4:bce5:5500::/64"),
					resource.TestCheckResourceAttr("netcalc_subnet.test", "id", "fd18:fad4:bce5:5500::/64"),
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
