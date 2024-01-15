package provider

import (
	"context"

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
	cfg *clientv2.Config
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

func (d *keyValueDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	var data keyValueDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	//var kApi clientv2.httpKeysAPI
	//kApi = *d.kApi

	client, err := clientv2.New(*d.cfg)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create etcdv2 API client",
			"An unexpected error occurred when creating the etcdv2 API client.\n\n"+
				"etcdv2 Client Error: "+err.Error(),
		)

		return
	}

	kApi := clientv2.NewKeysAPI(client)

	keyvalue, err := kApi.Get(context.Background(), data.Key.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read etcd keyvalue",
			err.Error(),
		)
		return
	}

	data.Value = types.StringValue(keyvalue.Node.Value)
	data.ModifiedIndex = types.Int64Value(int64(keyvalue.Node.ModifiedIndex))

	//keyValueState := keyValueModel{
	//	Key:         types.StringValue(keyvalue.Node.Key),
	//	Value:       types.StringValue(keyvalue.Node.Value),
	//	LastUpdated: types.StringValue(string(keyvalue.Node.ModifiedIndex)),
	//}

	diags := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

func (d *keyValueDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	cfg := req.ProviderData.(*clientv2.Config)

	d.cfg = cfg
}
