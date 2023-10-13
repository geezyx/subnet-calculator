// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"net/netip"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &SubnetResource{}
var _ resource.ResourceWithImportState = &SubnetResource{}
var _ resource.ResourceWithConfigure = &SubnetResource{}

func NewSubnetResource() resource.Resource {
	return &SubnetResource{}
}

// SubnetResource defines the resource implementation.
type SubnetResource struct {
	calculator SubnetCalculator
}

// SubnetResourceModel describes the resource data model.
type SubnetResourceModel struct {
	IPFamily       types.String `tfsdk:"ip_family"`
	CIDRMaskLength types.Int64  `tfsdk:"cidr_mask_length"`
	CIDRBlock      types.String `tfsdk:"cidr_block"`
	ID             types.String `tfsdk:"id"`
}

const (
	ipFamilyIPv4 = "ipv4"
	ipFamilyIPv6 = "ipv6"
)

func (r *SubnetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subnet"
}

func (r *SubnetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Subnet resource",

		Attributes: map[string]schema.Attribute{
			"ip_family": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(ipFamilyIPv4),
				MarkdownDescription: "The IP family for the calculated addresses. Must be one of ipv4 or ipv6.",
				Validators:          []validator.String{stringvalidator.OneOf(ipFamilyIPv4, ipFamilyIPv6)},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cidr_mask_length": schema.Int64Attribute{
				MarkdownDescription: "Network size in bits. e.g. if you wanted a /27 network, 27 would be the value here.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"cidr_block": schema.StringAttribute{
				MarkdownDescription: "Calculated CIDR block.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource ID, same as the calculated cidr_block.",
				Computed:            true,
			},
		},
	}
}

func (r *SubnetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	switch calc := req.ProviderData.(type) {
	case SubnetCalculator:
		r.calculator = calc
	case nil:
		return
	default:
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected SubnetCalculator, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
	}
}

func (r *SubnetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SubnetResourceModel

	// Read Terraform plan data into the model.
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.calculateSubnet(&data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Info(ctx, "created a subnet resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SubnetResource) calculateSubnet(plan *SubnetResourceModel) (diagnostics diag.Diagnostics) {
	cidrMaskLength := int(plan.CIDRMaskLength.ValueInt64())
	nextFunc := r.calculator.NextAvailableIPv4Subnet
	if plan.IPFamily.ValueString() == ipFamilyIPv6 {
		nextFunc = r.calculator.NextAvailableIPv6Subnet
	}
	next, err := nextFunc(cidrMaskLength)
	if err != nil {
		diagnostics.AddError("CIDR calculation error", fmt.Sprintf("Unable to calculate next available CIDR: %v", err))
		return diagnostics
	}

	// Save the calculated CIDR blocks into the Terraform state.
	plan.CIDRBlock = types.StringValue(next.String())
	plan.ID = types.StringValue(next.String())
	return diagnostics
}

func (r *SubnetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SubnetResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// See if the CIDR blocks are still valid
	p := parsePrefix(data.CIDRBlock, resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.calculator.PrefixInPools(p) {
		tflog.Info(ctx, "CIDR block is no longer valid; removing state in order to recalculate resource")
		resp.State.RemoveResource(ctx)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SubnetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SubnetResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	var state SubnetResourceModel
	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	// Set state values. Update operations are always modeled as a replacement, so we don't do any reallocation here.
	if plan.CIDRBlock.IsNull() || plan.CIDRBlock.IsUnknown() {
		tflog.Info(ctx, "Updating a CIDR block")
		resp.Diagnostics.Append(r.calculateSubnet(&plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		plan.CIDRBlock = state.CIDRBlock
		plan.ID = state.ID
	}

	// Save updated data into Terraform state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SubnetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SubnetResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	prefix := parsePrefix(data.CIDRBlock, resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.calculator.DeleteAllocatedPrefix(prefix)
	tflog.Info(ctx, "deleted a subnet resource")
}

func (r *SubnetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the CIDR from the ID.
	p, err := netip.ParsePrefix(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("CIDR parsing error", fmt.Sprintf("Unable to parse CIDR from ID: %q, %v", req.ID, err))
		return
	}
	cidrBlock := types.StringValue(req.ID)
	maskLength := types.Int64Value(int64(p.Bits()))
	ipFamily := ipFamilyIPv4
	if p.Addr().Is6() {
		ipFamily = ipFamilyIPv6
	}

	// Save the calculated CIDR blocks into the Terraform state.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cidr_block"), cidrBlock)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("ip_family"), ipFamily)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cidr_mask_length"), maskLength)...)

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	tflog.Info(ctx, "imported a resource")
}
