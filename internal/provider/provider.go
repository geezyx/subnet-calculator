// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/netip"
	"sync"

	"github.com/geezyx/subnet-calculator/internal/subnet"
	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure NetcalcProvider satisfies various provider interfaces.
var _ provider.Provider = &NetcalcProvider{}

// NetcalcProvider defines the provider implementation.
type NetcalcProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string

	calculator *syncCalculator
}

type SubnetCalculator interface {
	AddPool(prefix netip.Prefix)
	AddAllocatedPrefix(prefix netip.Prefix)
	NextAvailableIPv4Subnet(numBits int) (netip.Prefix, error)
	NextAvailableIPv6Subnet(numBits int) (netip.Prefix, error)
	DeleteAllocatedPrefix(prefix netip.Prefix)
	PrefixInPools(prefix netip.Prefix) bool
}

// SubnetCalculatorProviderModel describes the provider data model.
type SubnetCalculatorProviderModel struct {
	PoolCIDRBlocks    types.List `tfsdk:"pool_cidr_blocks"`
	ClaimedCIDRBlocks types.List `tfsdk:"claimed_cidr_blocks"`
}

func (p *NetcalcProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "netcalc"
	resp.Version = p.version
}

func (p *NetcalcProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"pool_cidr_blocks": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "IPv4 and/or IPv6 CIDR blocks that form a collective pool to be allocated in this provider.",
				Validators:          []validator.List{listvalidator.ValueStringsAre(ipAddressValidator{})},
			},
			"claimed_cidr_blocks": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "IPv4 and/or IPv6 CIDR blocks that are already claimed by other resources.",
				Validators:          []validator.List{listvalidator.ValueStringsAre(ipAddressValidator{})},
			},
		},
	}
}

type ipAddressValidator struct {
}

func (v ipAddressValidator) Description(ctx context.Context) string {
	return "value must be a valid IPv4 or IPv6 CIDR block"
}

func (v ipAddressValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v ipAddressValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()

	if _, err := netip.ParsePrefix(value); err != nil {
		response.Diagnostics.Append(validatordiag.InvalidAttributeValueMatchDiagnostic(
			request.Path,
			v.Description(ctx),
			value,
		))
	}
}

var _ validator.String = &ipAddressValidator{}

func (p *NetcalcProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data SubnetCalculatorProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Configured new netcalc provider")
	p.calculator = &syncCalculator{
		c: subnet.NewCalculator(),
	}

	for _, prefix := range parsePrefixList(data.PoolCIDRBlocks, &resp.Diagnostics) {
		p.calculator.AddPool(prefix)
	}
	for _, prefix := range parsePrefixList(data.ClaimedCIDRBlocks, &resp.Diagnostics) {
		p.calculator.AddAllocatedPrefix(prefix)
	}

	resp.DataSourceData = p.calculator
	resp.ResourceData = p.calculator
}

func parsePrefixList(data types.List, diagnostics *diag.Diagnostics) []netip.Prefix {
	var prefixes []netip.Prefix
	for _, elem := range data.Elements() {
		cidr, ok := elem.(types.String)
		if !ok {
			diagnostics.AddError("Value conversion error", "Unable to build a value from the the list of pool CIDR blocks.")
			continue
		}
		n, err := netip.ParsePrefix(cidr.ValueString())
		if err != nil {
			diagnostics.AddError("CIDR parsing error", fmt.Sprintf("Unable to parse pool CIDR: %q, %v", cidr, err))
			continue
		}
		prefixes = append(prefixes, n)
	}
	return prefixes
}

func parsePrefix(cidr types.String, diagnostics diag.Diagnostics) netip.Prefix {
	n, err := netip.ParsePrefix(cidr.ValueString())
	if err != nil {
		diagnostics.AddError("CIDR parsing error", fmt.Sprintf("Unable to parse CIDR: %q, %v", cidr, err))
	}
	return n
}

func (p *NetcalcProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSubnetResource,
		NewSubnetsResource,
	}
}

func (p *NetcalcProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &NetcalcProvider{
			version: version,
		}
	}
}

type syncCalculator struct {
	c SubnetCalculator
	m sync.Mutex
}

func (s *syncCalculator) AddPool(prefix netip.Prefix) {
	s.m.Lock()
	defer s.m.Unlock()
	s.c.AddPool(prefix)
}

func (s *syncCalculator) AddAllocatedPrefix(prefix netip.Prefix) {
	s.m.Lock()
	defer s.m.Unlock()
	s.c.AddAllocatedPrefix(prefix)
}

func (s *syncCalculator) NextAvailableIPv4Subnet(numBits int) (netip.Prefix, error) {
	s.m.Lock()
	defer s.m.Unlock()
	return s.c.NextAvailableIPv4Subnet(numBits)
}

func (s *syncCalculator) NextAvailableIPv6Subnet(numBits int) (netip.Prefix, error) {
	s.m.Lock()
	defer s.m.Unlock()
	return s.c.NextAvailableIPv6Subnet(numBits)
}

func (s *syncCalculator) DeleteAllocatedPrefix(prefix netip.Prefix) {
	s.m.Lock()
	defer s.m.Unlock()
	s.c.DeleteAllocatedPrefix(prefix)
}

func (s *syncCalculator) PrefixInPools(prefix netip.Prefix) bool {
	s.m.Lock()
	defer s.m.Unlock()
	return s.c.PrefixInPools(prefix)
}

var _ SubnetCalculator = &syncCalculator{}
