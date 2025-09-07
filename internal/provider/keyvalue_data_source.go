package provider

import (
	"context"
	"fmt"

	clientv2 "go.etcd.io/etcd/client/v2"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &keyValueDataSource{}
	_ datasource.DataSourceWithConfigure = &keyValueDataSource{}
)

func NewKeyValueDataSource() datasource.DataSource {
	return &keyValueDataSource{}
}

type keyValueDataSource struct {
	client *clientv2.Client
}

type keyValueDataSourceModel struct {
	Key           types.String `tfsdk:"key"`
	Value         types.String `tfsdk:"value"`
	ModifiedIndex types.Int64  `tfsdk:"modified_index"`
}

func (d *keyValueDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_keyvalue"
}

func (d *keyValueDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"key": schema.StringAttribute{
				Required: true,
				Computed: false,
			},
			"value": schema.StringAttribute{
				Computed: true,
			},
			"modified_index": schema.Int64Attribute{
				Computed: true,
			},
		},
	}
}

func (d *keyValueDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client
}

func (d *keyValueDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	var data keyValueDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	kapi := clientv2.NewKeysAPI(*d.client)

	keyvalue, err := kapi.Get(context.Background(), data.Key.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read etcd keyvalue",
			err.Error(),
		)
		return
	}

	data.Value = types.StringValue(keyvalue.Node.Value)
	data.ModifiedIndex = types.Int64Value(int64(keyvalue.Node.ModifiedIndex))

	diags := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}
