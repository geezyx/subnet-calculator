// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net"

	"github.com/geezyx/subnet-calculator/internal/subnetcalculator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &SubnetResource{}
var _ resource.ResourceWithImportState = &SubnetResource{}

func NewSubnetResource() resource.Resource {
	return &SubnetResource{}
}

// SubnetResource defines the resource implementation.
type SubnetResource struct{}

// SubnetResourceModel describes the resource data model.
type SubnetResourceModel struct {
	AvailableCIDRBlocks types.List   `tfsdk:"available_cidr_blocks"`
	UsedCIDRBlocks      types.List   `tfsdk:"used_cidr_blocks"`
	SubnetSize          types.Int64  `tfsdk:"network_size"`
	CIDRBlock           types.String `tfsdk:"cidr_block"`
}

func (r *SubnetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subnet"
}

func (r *SubnetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Subnet resource",

		Attributes: map[string]schema.Attribute{
			"available_cidr_blocks": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of CIDR blocks from which to select an available subnet.",
				Required:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplaceIf(AvailableCIDRBlocksNoLongerContainsResourceCIDR, "Calculated CIDR block no longer falls within the available CIDR blocks, new CIDR will be calculated.", ""),
				},
			},
			"used_cidr_blocks": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of CIDR blocks which are already in use.",
				Optional:            true,
			},
			"network_size": schema.Int64Attribute{
				MarkdownDescription: "Network size in bits. e.g. if you wanted a /27 network, 27 would be the value here.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"cidr_block": schema.StringAttribute{
				MarkdownDescription: "Calculated CIDR block.",
				Computed:            true,
			},
		},
	}
}

func (r *SubnetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
}

func (r *SubnetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SubnetResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var supernets, subnets []net.IPNet

	for _, cidr := range data.AvailableCIDRBlocks.Elements() {
		_, n, err := net.ParseCIDR(cidr.String())
		if err != nil || n == nil {
			resp.Diagnostics.AddError("CIDR parsing error", fmt.Sprintf("Unable to parse CIDR: %s, %v", cidr.String(), err))
		}
		supernets = append(supernets, *n)
	}

	for _, cidr := range data.UsedCIDRBlocks.Elements() {
		_, n, err := net.ParseCIDR(cidr.String())
		if err != nil || n == nil {
			resp.Diagnostics.AddError("CIDR parsing error", fmt.Sprintf("Unable to parse CIDR: %s, %v", cidr.String(), err))
		}
		subnets = append(subnets, *n)
	}

	calc := subnetcalculator.New(supernets, subnets)

	next, err := calc.NextAvailableSubnet(net.CIDRMask(int(data.SubnetSize.ValueInt64()), 32))
	if err != nil {
		resp.Diagnostics.AddError("CIDR calculation error", fmt.Sprintf("Unable to calculate next available CIDR: %v", err))
	}

	// Save the calculated CIDR block into the Terraform state.
	data.CIDRBlock = types.StringValue(next.String())

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SubnetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SubnetResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SubnetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SubnetResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SubnetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SubnetResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *SubnetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// AvailableCIDRBlocksNoLongerContainsResourceCIDR checks the existing calculated CIDR block (if it exists in the current state)
// against the list of available CIDR blocks in the configuration. If the calculated CIDR no longer belongs to one of the available
// blocks, it will require replacement.
func AvailableCIDRBlocksNoLongerContainsResourceCIDR(ctx context.Context, req planmodifier.ListRequest, resp *listplanmodifier.RequiresReplaceIfFuncResponse) {
	var state SubnetResourceModel
	if err := req.State.Raw.As(&state); err != nil {
		resp.Diagnostics.AddError("Current state error", fmt.Sprintf("Unable to load current state of subnet resource: %v", err))
	}
	if state.CIDRBlock.ValueString() == "" {
		return
	}
	_, subnet, err := net.ParseCIDR(state.CIDRBlock.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("CIDR parsing error", fmt.Sprintf("Parsing existing cidr_block value of subnet resource: %v", err))
	}

	var config SubnetResourceModel
	if err := req.Config.Raw.As(&config); err != nil {
		resp.Diagnostics.AddError("Current state error", fmt.Sprintf("Unable to load current state of subnet resource: %v", err))
	}
	for _, cidr := range config.AvailableCIDRBlocks.Elements() {
		_, n, err := net.ParseCIDR(cidr.String())
		if err != nil || n == nil {
			resp.Diagnostics.AddError("CIDR parsing error", fmt.Sprintf("Unable to parse CIDR: %s, %v", cidr.String(), err))
		}
		if n.Contains(subnet.IP) {
			return
		}
	}
	resp.RequiresReplace = true
}
