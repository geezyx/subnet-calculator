// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/netip"
	"strings"

	"github.com/geezyx/subnet-calculator/internal/subnet"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
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
}

// SubnetsResourceModel describes the resource data model.
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
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"cidr_blocks": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Calculated CIDR block.",
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource ID, same as the calculated cidr_blocks.",
				Computed:            true,
			},
		},
	}
}

func (r *SubnetsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
}

func (r *SubnetsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SubnetsResourceModel

	// Read Terraform plan data into the model.
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Load CIDR blocks into calculator.
	calculator := subnet.NewCalculator()
	resp.Diagnostics.Append(r.LoadCIDRBlocks(ctx, data, calculator)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cidrMaskLength := int(data.CIDRMaskLength.ValueInt64())
	var calculatedCIDRs []types.String
	var cidrStrings []string
	for i := int64(0); i < data.CIDRCount.ValueInt64(); i++ {
		next, err := calculator.NextAvailableSubnet(cidrMaskLength)
		if err != nil {
			resp.Diagnostics.AddError("CIDR calculation error", fmt.Sprintf("Unable to calculate next available CIDR: %v", err))
			return
		}
		calculatedCIDRs = append(calculatedCIDRs, types.StringValue(next.String()))
		cidrStrings = append(cidrStrings, next.String())
	}

	// Save the calculated CIDR blocks into the Terraform state.
	val, diagnostics := types.ListValueFrom(ctx, types.StringType, calculatedCIDRs)
	resp.Diagnostics.Append(diagnostics...)
	data.CIDRBlocks = val

	// Set the ID
	data.ID = types.StringValue(strings.Join(cidrStrings, ","))

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Info(ctx, "created a resource")

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
	calculator := subnet.NewCalculator()
	resp.Diagnostics.Append(r.LoadCIDRBlocks(ctx, plan, calculator)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set state values.
	plan.CIDRBlocks = state.CIDRBlocks
	plan.ID = state.ID
	tflog.Info(ctx, "updated a resource")

	// Save updated data into Terraform state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SubnetsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SubnetsResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "deleted a resource")
}

func (r *SubnetsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the CIDRs from the ID.
	var prefixes []netip.Prefix
	var calculatedCIDRs []types.String
	for _, cidr := range strings.Split(req.ID, ",") {
		p, err := netip.ParsePrefix(cidr)
		if err != nil {
			resp.Diagnostics.AddError("CIDR parsing error", fmt.Sprintf("Unable to parse CIDR from ID: %q, %v", cidr, err))
			continue
		}
		prefixes = append(prefixes, p)
		calculatedCIDRs = append(calculatedCIDRs, types.StringValue(cidr))
	}
	if len(prefixes) == 0 {
		resp.Diagnostics.AddError("Invalid ID", "ID must consist of comma-separated CIDR blocks of the same size.")
	}
	maskLength := prefixes[0].Bits()
	for _, p := range prefixes {
		if p.Bits() != maskLength {
			resp.Diagnostics.AddError("CIDR prefix lengths do not match", fmt.Sprintf("Expected all cidr masks to be the same size, but found %d and %d.", maskLength, p.Bits()))
		}
	}

	// Save the calculated CIDR blocks into the Terraform state.
	val, diagnostics := types.ListValueFrom(ctx, types.StringType, calculatedCIDRs)
	resp.Diagnostics.Append(diagnostics...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cidr_blocks"), val)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cidr_count"), types.Int64Value(int64(len(calculatedCIDRs))))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cidr_mask_length"), types.Int64Value(int64(maskLength)))...)

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	tflog.Info(ctx, "imported a resource")
}

func (r *SubnetsResource) LoadCIDRBlocks(ctx context.Context, s SubnetsResourceModel, calculator *subnet.Calculator) diag.Diagnostics {
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
		if err := calculator.AddPool(n); err != nil {
			diagnostics.AddError("Subnet calculator error", fmt.Sprintf("Unable to add pool CIDR %q: %v", cidr, err))
		}
	}
	for _, cidr := range existingCIDRBlocks {
		n, err := netip.ParsePrefix(cidr.ValueString())
		if err != nil {
			diagnostics.AddError("CIDR parsing error", fmt.Sprintf("Unable to parse existing CIDR: %q, %v", cidr, err))
			continue
		}
		if err := calculator.AddAllocatedPrefix(n); err != nil {
			diagnostics.AddError("Subnet calculator error", fmt.Sprintf("Unable to add existing CIDR %q: %v", cidr, err))
		}
	}
	for _, cidr := range allocatedCIDRBlocks {
		n, err := netip.ParsePrefix(cidr.ValueString())
		if err != nil {
			diagnostics.AddError("CIDR parsing error", fmt.Sprintf("Unable to parse calculated CIDR: %q, %v", cidr, err))
			continue
		}
		if err := calculator.AddAllocatedPrefix(n); err != nil {
			diagnostics.AddError("Subnet calculator error", fmt.Sprintf("Unable to add calculated CIDR %q: %v", cidr, err))
		}
	}
	return diagnostics
}

// AvailableCIDRBlocksNoLongerContainsResourceCIDR checks the existing calculated CIDR block (if it exists in the current state)
// against the list of available CIDR blocks in the configuration. If the calculated CIDR no longer belongs to one of the available
// blocks, it will require replacement.
func (r *SubnetsResource) AvailableCIDRBlocksNoLongerContainsResourceCIDR(ctx context.Context, req planmodifier.SetRequest, resp *setplanmodifier.RequiresReplaceIfFuncResponse) {
	calculator := subnet.NewCalculator()

	var state SubnetsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var config SubnetsResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Load state into calculator.
	resp.Diagnostics.Append(r.LoadCIDRBlocks(ctx, config, calculator)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, elem := range state.CIDRBlocks.Elements() {
		cidr, ok := elem.(types.String)
		if !ok {
			resp.Diagnostics.AddError("Value conversion error", "Unable to build a value from the the list of allocated CIDR blocks.")
		}

		n, err := netip.ParsePrefix(cidr.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("CIDR parsing error", fmt.Sprintf("Unable to parse calculated CIDR: %q, %v", cidr, err))
			continue
		}
		if !calculator.PrefixInPools(n) {
			tflog.Debug(ctx, fmt.Sprintf("Prefix %s is not in cidr blocks %v", cidr.ValueString(), config.PoolCIDRBlocks))
			resp.RequiresReplace = true
		} else {
			tflog.Debug(ctx, fmt.Sprintf("Prefix %s is still in cidr blocks %v", cidr.ValueString(), config.PoolCIDRBlocks))
		}
	}
}
