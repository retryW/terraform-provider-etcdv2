package provider

import (
	"context"
	"fmt"

	clientv2 "go.etcd.io/etcd/client/v2"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &roleResource{}
	_ resource.ResourceWithImportState = &roleResource{}
)

func NewRoleResource() resource.Resource {
	return &roleResource{}
}

type roleResource struct {
	client *clientv2.Client
}

type roleResourceModel struct {
	Name        types.String              `tfsdk:"name"`
	Permissions []permissionResourceModel `tfsdk:"permissions"`
}

type permissionResourceModel struct {
	KeyPath types.String `tfsdk:"key_path"`
	Read    types.Bool   `tfsdk:"read"`
	Write   types.Bool   `tfsdk:"write"`
}

func (r *roleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *roleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an etcdv2 role.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the role.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permissions": schema.ListNestedAttribute{
				Description: "List of permissions for the role.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key_path": schema.StringAttribute{
							Description: "The key path for the permission.",
							Required:    true,
						},
						"read": schema.BoolAttribute{
							Description: "Whether read access is granted.",
							Required:    true,
						},
						"write": schema.BoolAttribute{
							Description: "Whether write access is granted.",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func (r *roleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*clientv2.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	authRoleAPI := clientv2.NewAuthRoleAPI(*r.client)

	// Create role
	err := authRoleAPI.AddRole(ctx, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating role",
			fmt.Sprintf("Could not create role %s: %s", plan.Name.ValueString(), err),
		)
		return
	}

	// Grant permissions
	if len(plan.Permissions) > 0 {
		for _, perm := range plan.Permissions {
			var permType clientv2.PermissionType
			read := perm.Read.ValueBool()
			write := perm.Write.ValueBool()

			if read && write {
				permType = clientv2.ReadWritePermission
			} else if read {
				permType = clientv2.ReadPermission
			} else if write {
				permType = clientv2.WritePermission
			} else {
				continue // Skip if neither read nor write
			}

			_, err := authRoleAPI.GrantRoleKV(ctx, plan.Name.ValueString(), []string{perm.KeyPath.ValueString()}, permType)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error granting permission to role",
					fmt.Sprintf("Could not grant permission on %s to role %s: %s", perm.KeyPath.ValueString(), plan.Name.ValueString(), err),
				)
				return
			}
		}
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	authRoleAPI := clientv2.NewAuthRoleAPI(*r.client)

	// Get role details
	role, err := authRoleAPI.GetRole(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading role",
			fmt.Sprintf("Could not read role %s: %s", state.Name.ValueString(), err),
		)
		return
	}

	// Update permissions
	state.Permissions = make([]permissionResourceModel, 0)
	for _, perm := range role.Permissions.KV.Read {
		state.Permissions = append(state.Permissions, permissionResourceModel{
			KeyPath: types.StringValue(perm),
			Read:    types.BoolValue(true),
			Write:   types.BoolValue(false),
		})
	}

	for _, perm := range role.Permissions.KV.Write {
		// Check if this path already has read permission
		found := false
		for i, existingPerm := range state.Permissions {
			if existingPerm.KeyPath.ValueString() == perm {
				state.Permissions[i].Write = types.BoolValue(true)
				found = true
				break
			}
		}
		if !found {
			state.Permissions = append(state.Permissions, permissionResourceModel{
				KeyPath: types.StringValue(perm),
				Read:    types.BoolValue(false),
				Write:   types.BoolValue(true),
			})
		}
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan roleResourceModel
	var state roleResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	authRoleAPI := clientv2.NewAuthRoleAPI(*r.client)

	// Revoke all existing permissions
	if len(state.Permissions) > 0 {
		for _, perm := range state.Permissions {
			var permType clientv2.PermissionType
			read := perm.Read.ValueBool()
			write := perm.Write.ValueBool()

			if read && write {
				permType = clientv2.ReadWritePermission
			} else if read {
				permType = clientv2.ReadPermission
			} else if write {
				permType = clientv2.WritePermission
			} else {
				continue
			}

			_, err := authRoleAPI.RevokeRoleKV(ctx, plan.Name.ValueString(), []string{perm.KeyPath.ValueString()}, permType)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error revoking permission from role",
					fmt.Sprintf("Could not revoke permission on %s from role %s: %s", perm.KeyPath.ValueString(), plan.Name.ValueString(), err),
				)
				return
			}
		}
	}

	// Grant new permissions
	if len(plan.Permissions) > 0 {
		for _, perm := range plan.Permissions {
			var permType clientv2.PermissionType
			read := perm.Read.ValueBool()
			write := perm.Write.ValueBool()

			if read && write {
				permType = clientv2.ReadWritePermission
			} else if read {
				permType = clientv2.ReadPermission
			} else if write {
				permType = clientv2.WritePermission
			} else {
				continue
			}

			_, err := authRoleAPI.GrantRoleKV(ctx, plan.Name.ValueString(), []string{perm.KeyPath.ValueString()}, permType)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error granting permission to role",
					fmt.Sprintf("Could not grant permission on %s to role %s: %s", perm.KeyPath.ValueString(), plan.Name.ValueString(), err),
				)
				return
			}
		}
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	authRoleAPI := clientv2.NewAuthRoleAPI(*r.client)

	err := authRoleAPI.RemoveRole(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting role",
			fmt.Sprintf("Could not delete role %s: %s", state.Name.ValueString(), err),
		)
		return
	}
}

func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by role name
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
