// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/geezyx/subnet-calculator/internal/subnet"
	"github.com/google/uuid"
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
var _ resource.Resource = &SubnetsResource{}
var _ resource.ResourceWithImportState = &SubnetsResource{}
var _ resource.ResourceWithConfigure = &SubnetsResource{}

func NewSubnetsResource() resource.Resource {
	return &SubnetsResource{}
}

// SubnetsResource defines the resource implementation.
type SubnetsResource struct {
	calculator *subnet.Calculator
}

// SubnetResourceModel describes the resource data model.
type SubnetsResourceModel struct {
	PoolCIDRBlocks     types.Set    `tfsdk:"pool_cidr_blocks"`
	ExistingCIDRBlocks types.Set    `tfsdk:"existing_cidr_blocks"`
	CIDRMaskLength     types.Int64  `tfsdk:"cidr_mask_length"`
	CIDRCount          types.Int64  `tfsdk:"cidr_count"`
	CIDRBlocks         types.List   `tfsdk:"cidr_blocks"`
	ID                 types.String `tfsdk:"id"`
}

func (r *SubnetsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subnets"
}

func (r *SubnetsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Subnet resource",

		Attributes: map[string]schema.Attribute{
			"pool_cidr_blocks": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Set of CIDR blocks from which to select an available subnet.",
				Required:            true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplaceIf(r.AvailableCIDRBlocksNoLongerContainsResourceCIDR, "Calculated CIDR block no longer falls within the available CIDR blocks, new CIDR will be calculated.", ""),
				},
			},
			"existing_cidr_blocks": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Set of CIDR blocks which are already in use.",
				Optional:            true,
			},
			"cidr_mask_length": schema.Int64Attribute{
				MarkdownDescription: "Network size in bits. e.g. if you wanted a /27 network, 27 would be the value here.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"cidr_count": schema.Int64Attribute{
				MarkdownDescription: "Number of CIDR blocks to provision",
				Required:            true,
			},
			"cidr_blocks": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Calculated CIDR block.",
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					UnknownValueOnCIDRCountChange(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource ID, same as the calculated cidr_block.",
				Computed:            true,
			},
		},
	}
}

func (r *SubnetsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.calculator = subnet.NewCalculator()
}

func (r *SubnetsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SubnetsResourceModel

	// Read Terraform plan data into the model.
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Load CIDR blocks into calculator.
	resp.Diagnostics.Append(r.LoadCIDRBlocks(ctx, data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cidrMaskLength := int(data.CIDRMaskLength.ValueInt64())
	var calculatedCIDRs []types.String
	for i := int64(0); i < data.CIDRCount.ValueInt64(); i++ {
		next, err := r.calculator.NextAvailableSubnet(cidrMaskLength)
		if err != nil {
			resp.Diagnostics.AddError("CIDR calculation error", fmt.Sprintf("Unable to calculate next available CIDR: %v", err))
			return
		}
		calculatedCIDRs = append(calculatedCIDRs, types.StringValue(next.String()))
	}

	// Save the calculated CIDR blocks into the Terraform state.
	val, diagnostics := types.ListValueFrom(ctx, types.StringType, calculatedCIDRs)
	resp.Diagnostics.Append(diagnostics...)
	data.CIDRBlocks = val

	// Set the ID
	id, err := uuid.NewRandom()
	if err != nil {
		resp.Diagnostics.AddError("Create ID error", fmt.Sprintf("Unable to create ID for resource: %v", err))
		return
	}
	data.ID = types.StringValue(id.String())

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SubnetsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SubnetsResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SubnetsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SubnetsResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	var state SubnetsResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	// Load CIDR blocks into calculator.
	resp.Diagnostics.Append(r.LoadCIDRBlocks(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update CIDR blocks.
	var cidrBlocks types.List
	var diagnostics diag.Diagnostics
	stateCount := state.CIDRCount.ValueInt64()
	planCount := plan.CIDRCount.ValueInt64()
	switch {
	case planCount == stateCount:
		cidrBlocks = state.CIDRBlocks
	case planCount > stateCount:
		cidrBlocks, diagnostics = r.IncreaseCIDRBlockCount(ctx, state.CIDRBlocks, int(planCount-stateCount), int(plan.CIDRMaskLength.ValueInt64()))
	case planCount < stateCount:
		cidrBlocks, diagnostics = r.DecreaseCIDRBlockCount(ctx, state.CIDRBlocks, int(stateCount-planCount))
	}
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set state values.
	plan.CIDRBlocks = cidrBlocks
	plan.ID = state.ID

	// Save updated data into Terraform state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SubnetsResource) IncreaseCIDRBlockCount(ctx context.Context, existing types.List, count, cidrMaskLength int) (types.List, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	var cidrs []types.String
	diagnostics.Append(existing.ElementsAs(ctx, &cidrs, false)...)
	for i := 0; i < count; i++ {
		next, err := r.calculator.NextAvailableSubnet(cidrMaskLength)
		if err != nil {
			diagnostics.AddError("CIDR calculation error", fmt.Sprintf("Unable to calculate next available CIDR: %v", err))
			continue
		}
		cidrs = append(cidrs, types.StringValue(next.String()))
	}
	val, d := types.ListValueFrom(ctx, types.StringType, cidrs)
	diagnostics.Append(d...)
	return val, diagnostics
}

func (r *SubnetsResource) DecreaseCIDRBlockCount(ctx context.Context, existing types.List, count int) (types.List, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	var cidrs []types.String
	diagnostics.Append(existing.ElementsAs(ctx, &cidrs, false)...)
	for i := 0; i < count; i++ {
		cidr := cidrs[len(cidrs)-1]
		n, err := netip.ParsePrefix(cidr.ValueString())
		if err != nil {
			diagnostics.AddError("CIDR parsing error", fmt.Sprintf("Unable to parse existing CIDR: %q, %v", cidr, err))
			continue
		}
		r.calculator.DeleteAllocatedPrefix(n)
		cidrs = cidrs[:len(cidrs)-1]
	}
	val, d := types.ListValueFrom(ctx, types.StringType, cidrs)
	diagnostics.Append(d...)
	return val, diagnostics
}

func (r *SubnetsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SubnetsResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *SubnetsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *SubnetsResource) LoadCIDRBlocks(ctx context.Context, s SubnetsResourceModel) diag.Diagnostics {
	var diagnostics diag.Diagnostics

	var poolCIDRBlocks []types.String
	diagnostics.Append(s.PoolCIDRBlocks.ElementsAs(ctx, &poolCIDRBlocks, false)...)

	var existingCIDRBlocks []types.String
	diagnostics.Append(s.ExistingCIDRBlocks.ElementsAs(ctx, &existingCIDRBlocks, false)...)

	var allocatedCIDRBlocks []types.String
	for _, elem := range s.CIDRBlocks.Elements() {
		cidr, ok := elem.(types.String)
		if !ok {
			diagnostics.AddError("Value conversion error", "Unable to build a value from the the list of allocated CIDR blocks.")
		}
		allocatedCIDRBlocks = append(allocatedCIDRBlocks, cidr)
	}

	for _, cidr := range poolCIDRBlocks {
		n, err := netip.ParsePrefix(cidr.ValueString())
		if err != nil {
			diagnostics.AddError("CIDR parsing error", fmt.Sprintf("Unable to parse pool CIDR %q: %v", cidr, err))
			continue
		}
		if err := r.calculator.AddPool(n); err != nil {
			diagnostics.AddError("Subnet calculator error", fmt.Sprintf("Unable to add pool CIDR %q: %v", cidr, err))
		}
	}
	for _, cidr := range existingCIDRBlocks {
		n, err := netip.ParsePrefix(cidr.ValueString())
		if err != nil {
			diagnostics.AddError("CIDR parsing error", fmt.Sprintf("Unable to parse existing CIDR: %q, %v", cidr, err))
			continue
		}
		if err := r.calculator.AddAllocatedPrefix(n); err != nil {
			diagnostics.AddError("Subnet calculator error", fmt.Sprintf("Unable to add existing CIDR %q: %v", cidr, err))
		}
	}
	for _, cidr := range allocatedCIDRBlocks {
		n, err := netip.ParsePrefix(cidr.ValueString())
		if err != nil {
			diagnostics.AddError("CIDR parsing error", fmt.Sprintf("Unable to parse calculated CIDR: %q, %v", cidr, err))
			continue
		}
		if err := r.calculator.AddAllocatedPrefix(n); err != nil {
			diagnostics.AddError("Subnet calculator error", fmt.Sprintf("Unable to add calculated CIDR %q: %v", cidr, err))
		}
	}
	return diagnostics
}

// AvailableCIDRBlocksNoLongerContainsResourceCIDR checks the existing calculated CIDR block (if it exists in the current state)
// against the list of available CIDR blocks in the configuration. If the calculated CIDR no longer belongs to one of the available
// blocks, it will require replacement.
func (r *SubnetsResource) AvailableCIDRBlocksNoLongerContainsResourceCIDR(ctx context.Context, req planmodifier.SetRequest, resp *setplanmodifier.RequiresReplaceIfFuncResponse) {
	// Plan modifier doesnt call Configure so we need to initialize our calculator.
	r.calculator = subnet.NewCalculator()

	var state SubnetsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Load state into calculator.
	resp.Diagnostics.Append(r.LoadCIDRBlocks(ctx, state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var allocatedCIDRBlocks []types.String
	for _, elem := range state.CIDRBlocks.Elements() {
		cidr, ok := elem.(types.String)
		if !ok {
			resp.Diagnostics.AddError("Value conversion error", "Unable to build a value from the the list of allocated CIDR blocks.")
		}
		allocatedCIDRBlocks = append(allocatedCIDRBlocks, cidr)
	}

	for _, cidr := range allocatedCIDRBlocks {
		n, err := netip.ParsePrefix(cidr.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("CIDR parsing error", fmt.Sprintf("Unable to parse calculated CIDR: %q, %v", cidr, err))
			continue
		}
		r.calculator.PrefixInPools(n)
	}
	resp.RequiresReplace = true
}

// UseStateForUnknown returns a plan modifier that copies a known prior state
// value into the planned value. Use this when it is known that an unconfigured
// value will remain the same after a resource update.
//
// To prevent Terraform errors, the framework automatically sets unconfigured
// and Computed attributes to an unknown value "(known after apply)" on update.
// Using this plan modifier will instead display the prior state value in the
// plan, unless a prior plan modifier adjusts the value.
func UnknownValueOnCIDRCountChange() planmodifier.List {
	return unknownValueOnCIDRCountChange{}
}

// useStateForUnknownModifier implements the plan modifier.
type unknownValueOnCIDRCountChange struct{}

// Description returns a human-readable description of the plan modifier.
func (m unknownValueOnCIDRCountChange) Description(_ context.Context) string {
	return "Once set, the value of this attribute in state will not change."
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m unknownValueOnCIDRCountChange) MarkdownDescription(_ context.Context) string {
	return "Once set, the value of this attribute in state will not change."
}

// PlanModifyList implements the plan modification logic.
func (m unknownValueOnCIDRCountChange) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	// Do nothing if there is no state value.
	if req.StateValue.IsNull() {
		return
	}

	// Read Terraform plan data into the model
	var plan SubnetsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	// Read Terraform plan data into the model
	var state SubnetsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if !plan.CIDRCount.Equal(state.CIDRCount) {
		resp.PlanValue = types.ListUnknown(types.StringType)
	}
}
