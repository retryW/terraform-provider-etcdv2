package provider

import (
	"context"
	"fmt"

	clientv2 "go.etcd.io/etcd/client/v2"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &KeyValueResource{}
	_ resource.ResourceWithConfigure   = &KeyValueResource{}
	_ resource.ResourceWithImportState = &KeyValueResource{}
)

func NewKeyValueResource() resource.Resource {
	return &KeyValueResource{}
}

// KeyValueResource defines the resource implementation.
type KeyValueResource struct {
	client *clientv2.Client
}

// KeyValueResourceModel describes the resource data model.
type KeyValueResourceModel struct {
	Key           types.String `tfsdk:"key"`
	Value         types.String `tfsdk:"value"`
	ModifiedIndex types.Int64  `tfsdk:"modified_index"`
}

func (r *KeyValueResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_keyvalue"
}

func (r *KeyValueResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "etcdv2 Key-value resource",
		Attributes: map[string]schema.Attribute{
			"key": schema.StringAttribute{
				MarkdownDescription: "The unique location of this resource (e.g. '/foo/bar')",
				Required:            true,
				Computed:            false,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "The data stored in this resource",
				Required:            true,
				Computed:            false,
			},
			"modified_index": schema.Int64Attribute{
				MarkdownDescription: "The index at which this resource was last modified",
				Computed:            true,
			},
		},
	}
}

func (r *KeyValueResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*clientv2.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected clientv2.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *KeyValueResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data KeyValueResourceModel

	// Read Terraform plan data into the model.
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	// If we fail to retrieve the plan data, we don't want to continue.
	if resp.Diagnostics.HasError() {
		return
	}

	// Retrieve KeyAPI from client.
	kapi := clientv2.NewKeysAPI(*r.client)

	keyvalue, err := kapi.Create(context.Background(), data.Key.ValueString(), data.Value.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create etcd keyvalue",
			err.Error(),
		)
		return
	}

	data.Value = types.StringValue(keyvalue.Node.Value)
	data.ModifiedIndex = types.Int64Value(int64(keyvalue.Node.ModifiedIndex))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KeyValueResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KeyValueResourceModel

	// Read Terraform prior state data into the model.
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Retrieve KeyAPI from client.
	kapi := clientv2.NewKeysAPI(*r.client)

	keyvalue, err := kapi.Get(context.Background(), data.Key.ValueString(), nil)
	if err != nil {
		if !clientv2.IsKeyNotFound(err) {
			resp.Diagnostics.AddError(
				"Unable to Read etcd keyvalue",
				err.Error(),
			)
			return
		}
		// Key was not found, remove from state
		resp.State.RemoveResource(ctx)
		return
	}
	data.Value = types.StringValue(keyvalue.Node.Value)
	data.ModifiedIndex = types.Int64Value(int64(keyvalue.Node.ModifiedIndex))

	// Save updated data into Terraform state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KeyValueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data KeyValueResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Retrieve KeyAPI from client.
	kapi := clientv2.NewKeysAPI(*r.client)

	keyvalue, err := kapi.Set(context.Background(), data.Key.ValueString(), data.Value.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update etcd keyvalue",
			err.Error(),
		)
		return
	}

	data.Value = types.StringValue(keyvalue.Node.Value)
	data.ModifiedIndex = types.Int64Value(int64(keyvalue.Node.ModifiedIndex))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KeyValueResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KeyValueResourceModel

	// Read Terraform prior state data into the model.
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Retrieve KeyAPI from client.
	kapi := clientv2.NewKeysAPI(*r.client)

	_, err := kapi.Delete(context.Background(), data.Key.ValueString(), nil)
	if err != nil {
		if !clientv2.IsKeyNotFound(err) {
			resp.Diagnostics.AddError(
				"Error when trying to Delete etcd keyvalue",
				err.Error(),
			)
			return
		}
		// Key was not found, which is seen as a success
	}
}

func (r *KeyValueResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute.
	resource.ImportStatePassthroughID(ctx, path.Root("key"), req, resp)
}
