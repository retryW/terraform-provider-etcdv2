// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"
	"time"

	clientv2 "go.etcd.io/etcd/client/v2"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure implementation satisfies various provider interfaces.
var _ provider.Provider = &etcdv2Provider{}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &etcdv2Provider{
			version: version,
		}
	}
}

// etcdv2Provider defines the provider implementation.
type etcdv2Provider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type etcdv2ProviderModel struct {
	Host     types.String `tfsdk:"host"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Timeout  types.Int64  `tfsdk:"timeout"`
}

func (p *etcdv2Provider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "etcdv2"
	resp.Version = p.version
}

func (p *etcdv2Provider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				MarkdownDescription: "The host address of your etcd server",
				Optional:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The username used for authentication",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password used for authentication",
				Optional:            true,
				Sensitive:           true,
			},
			"timeout": schema.Int64Attribute{
				MarkdownDescription: "Maximum header timeout",
				Optional:            true,
			},
		},
	}
}

func (p *etcdv2Provider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config etcdv2ProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	// Configuration values are now available.
	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown etcd API Host",
			"The provider cannot create the ectdv2 API client as there is an unknown host value. ",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	host := os.Getenv("ETCDV2_HOST")
	username := os.Getenv("ETCDV2_USERNAME")
	password := os.Getenv("ETCDV2_PASSWORD")

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	}

	if !config.Password.IsNull() {
		password = config.Password.ValueString()
	}

	var timeoutSec int64 = 1
	if !config.Timeout.IsNull() {
		timeoutSec = config.Timeout.ValueInt64()
	}
	if timeoutSec <= 0 {
		resp.Diagnostics.AddAttributeError(path.Root("timeout"), "Invalid timeout", "timeout must be > 0")
		return
	}
	headerTimeout := time.Duration(timeoutSec) * time.Second

	if host == "" {
		resp.Diagnostics.AddError(
			"No host detected.",
			"Ensure a host value is set either via ENV or Config",
		)
		return
	}

	var cfg clientv2.Config

	if (username != "") && (password != "") {
		cfg = clientv2.Config{
			Endpoints:               []string{host},
			Transport:               clientv2.DefaultTransport,
			HeaderTimeoutPerRequest: headerTimeout,
			Username:                username,
			Password:                password,
		}
	} else {
		cfg = clientv2.Config{
			Endpoints:               []string{host},
			Transport:               clientv2.DefaultTransport,
			HeaderTimeoutPerRequest: headerTimeout,
		}
	}

	etcdClient, err := clientv2.New(cfg)
	if err != nil {
		resp.Diagnostics.AddError(
			"An unexpected error occurred when creating the etcd client",
			"Error: "+err.Error(),
		)
	}

	kapi := clientv2.NewKeysAPI(etcdClient)
	_, err = kapi.Get(context.Background(), "/", &clientv2.GetOptions{})
	if err != nil {
		if !clientv2.IsKeyNotFound(err) {
			tflog.Warn(ctx, "Could not test etcd connection", map[string]any{
				"Error": err.Error(),
			})
		}
	}

	resp.DataSourceData = &etcdClient
	resp.ResourceData = &etcdClient

	tflog.Info(ctx, "Configured etcd client", map[string]any{
		"host":     host,
		"username": username != "",
		"success":  true,
	})
}

func (p *etcdv2Provider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewKeyValueResource,
		NewRoleResource,
		NewUserResource,
	}
}

func (p *etcdv2Provider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewKeyValueDataSource,
	}
}
