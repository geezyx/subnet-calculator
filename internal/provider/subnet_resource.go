// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/geezyx/subnet-calculator/internal/subnetcalculator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
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
	AvailableCIDRBlocks types.Set    `tfsdk:"available_cidr_blocks"`
	UsedCIDRBlocks      types.Set    `tfsdk:"used_cidr_blocks"`
	SubnetSize          types.Int64  `tfsdk:"network_size"`
	CIDRBlock           types.String `tfsdk:"cidr_block"`
	ID                  types.String `tfsdk:"id"`
}

func (s *SubnetResourceModel) ParseIPNets(ctx context.Context) (available, used []netip.Prefix, diagnostics diag.Diagnostics) {
	var availableCIDRBlocks []string
	diagnostics.Append(s.AvailableCIDRBlocks.ElementsAs(ctx, &availableCIDRBlocks, false)...)

	for _, cidr := range availableCIDRBlocks {
		n, err := netip.ParsePrefix(cidr)
		if err != nil {
			diagnostics.AddError("CIDR parsing error", fmt.Sprintf("Unable to parse CIDR: %q, %v", cidr, err))
			continue
		}
		available = append(available, n)
	}

	var usedCIDRBlocks []string
	diagnostics.Append(s.UsedCIDRBlocks.ElementsAs(ctx, &usedCIDRBlocks, false)...)

	for _, cidr := range usedCIDRBlocks {
		n, err := netip.ParsePrefix(cidr)
		if err != nil {
			diagnostics.AddError("CIDR parsing error", fmt.Sprintf("Unable to parse CIDR: %q, %v", cidr, err))
			continue
		}
		used = append(used, n)
	}
	return available, used, diagnostics
}

func (r *SubnetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subnet"
}

func (r *SubnetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Subnet resource",

		Attributes: map[string]schema.Attribute{
			"available_cidr_blocks": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Set of CIDR blocks from which to select an available subnet.",
				Required:            true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplaceIf(AvailableCIDRBlocksNoLongerContainsResourceCIDR, "Calculated CIDR block no longer falls within the available CIDR blocks, new CIDR will be calculated.", ""),
				},
			},
			"used_cidr_blocks": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Set of CIDR blocks which are already in use.",
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
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource ID, same as the calculated cidr_block.",
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

	available, used, diagnostics := data.ParseIPNets(ctx)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}

	calc, err := subnetcalculator.New(available, used)
	if err != nil {
		resp.Diagnostics.AddError("CIDR calculation error", fmt.Sprintf("Error building subnet calculator: %v", err))
		return
	}

	next, err := calc.NextAvailableSubnet(int(data.SubnetSize.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("CIDR calculation error", fmt.Sprintf("Unable to calculate next available CIDR: %v", err))
		return
	}

	// Save the calculated CIDR block into the Terraform state.
	data.CIDRBlock = types.StringValue(next.String())
	data.ID = types.StringValue(next.String())

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

	var stateData SubnetResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// When changes are made to the inputs, we don't need to recalculate the CIDR block.
	// We can keep the existing CIDR block from the existing state.
	data.ID = stateData.ID
	data.CIDRBlock = stateData.CIDRBlock

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
func AvailableCIDRBlocksNoLongerContainsResourceCIDR(ctx context.Context, req planmodifier.SetRequest, resp *setplanmodifier.RequiresReplaceIfFuncResponse) {
	var state SubnetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.CIDRBlock.ValueString() == "" {
		return
	}
	currentCIDR, err := netip.ParsePrefix(state.CIDRBlock.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("CIDR parsing error", fmt.Sprintf("Parsing existing cidr_block value of subnet resource: %v", err))
	}

	var plan SubnetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	availableCIDRs, _, diagnostics := plan.ParseIPNets(ctx)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, cidr := range availableCIDRs {
		if cidr.Contains(currentCIDR.Addr()) {
			return
		}
	}
	resp.RequiresReplace = true
}
