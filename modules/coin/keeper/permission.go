package keeper

import (
	"fmt"

	keepertypes "github.com/b2network/pulsar/keeper/types"
	"github.com/b2network/pulsar/modules/coin/types"
)

// PermissionKeeper manages module permissions for coin operations
type PermissionKeeper struct {
	keeper *Keeper
}

// NewPermissionKeeper creates a new permission keeper
func NewPermissionKeeper(k *Keeper) *PermissionKeeper {
	return &PermissionKeeper{
		keeper: k,
	}
}

// GetPermissionSet returns the complete permission set
func (k PermissionKeeper) GetPermissionSet(ctx keepertypes.Context) types.PermissionSet {
	store := k.keeper.GetKVStore(ctx)
	key := []byte("permission_set")
	
	bz := store.Get(key)
	if bz == nil {
		// Return default permissions if not found
		return types.DefaultPermissionSet()
	}
	
	var permissionSet types.PermissionSet
	if err := k.keeper.GetCodec().Unmarshal(bz, &permissionSet); err != nil {
		// Return default permissions on unmarshal error
		return types.DefaultPermissionSet()
	}
	
	return permissionSet
}

// SetPermissionSet sets the complete permission set
func (k PermissionKeeper) SetPermissionSet(ctx keepertypes.Context, permissionSet types.PermissionSet) {
	store := k.keeper.GetKVStore(ctx)
	key := []byte("permission_set")
	
	bz, err := k.keeper.GetCodec().Marshal(permissionSet)
	if err != nil {
		panic(fmt.Errorf("failed to marshal permission set: %w", err))
	}
	
	store.Set(key, bz)
}

// GetModulePermissions returns permissions for a specific module
func (k PermissionKeeper) GetModulePermissions(ctx keepertypes.Context, moduleName string) (types.ModulePermissions, bool) {
	permissionSet := k.GetPermissionSet(ctx)
	return permissionSet.GetModulePermissions(moduleName)
}

// SetModulePermissions sets permissions for a module
func (k PermissionKeeper) SetModulePermissions(ctx keepertypes.Context, mp types.ModulePermissions) error {
	// Validate permissions
	if err := mp.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid module permissions: %w", err)
	}
	
	// Get current permission set
	permissionSet := k.GetPermissionSet(ctx)
	
	// Add/update module permissions
	permissionSet.AddModulePermissions(mp)
	
	// Validate the updated set
	if err := permissionSet.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid permission set after update: %w", err)
	}
	
	// Save updated permission set
	k.SetPermissionSet(ctx, permissionSet)
	
	// Emit event
	ctx.EventManager().EmitEvent(keepertypes.Event{
		Type: types.EventTypePermissionSet,
		Attributes: []keepertypes.Attribute{
			{Key: types.AttributeKeyModule, Value: mp.ModuleName},
			{Key: types.AttributeKeyPermission, Value: fmt.Sprintf("%v", mp.Permissions)},
		},
	})
	
	return nil
}

// RemoveModulePermissions removes all permissions for a module
func (k PermissionKeeper) RemoveModulePermissions(ctx keepertypes.Context, moduleName string) error {
	// Get current permission set
	permissionSet := k.GetPermissionSet(ctx)
	
	// Remove module permissions
	permissionSet.RemoveModulePermissions(moduleName)
	
	// Save updated permission set
	k.SetPermissionSet(ctx, permissionSet)
	
	// Emit event
	ctx.EventManager().EmitEvent(keepertypes.Event{
		Type: types.EventTypePermissionRemoved,
		Attributes: []keepertypes.Attribute{
			{Key: types.AttributeKeyModule, Value: moduleName},
		},
	})
	
	return nil
}

// HasPermission checks if a module has a specific permission
func (k PermissionKeeper) HasPermission(ctx keepertypes.Context, moduleName, permission string) bool {
	permissionSet := k.GetPermissionSet(ctx)
	return permissionSet.HasPermission(moduleName, permission)
}

// GrantPermission grants a specific permission to a module
func (k PermissionKeeper) GrantPermission(ctx keepertypes.Context, moduleName, permission string) error {
	// Get current permissions for the module
	mp, found := k.GetModulePermissions(ctx, moduleName)
	if !found {
		// Create new module permissions
		mp = types.NewModulePermissions(moduleName, []string{})
	}
	
	// Check if permission already exists
	if mp.HasPermission(permission) {
		return fmt.Errorf("module %s already has permission %s", moduleName, permission)
	}
	
	// Add the permission
	mp.Permissions = append(mp.Permissions, permission)
	
	// Set the updated permissions
	return k.SetModulePermissions(ctx, mp)
}

// RevokePermission revokes a specific permission from a module
func (k PermissionKeeper) RevokePermission(ctx keepertypes.Context, moduleName, permission string) error {
	// Get current permissions for the module
	mp, found := k.GetModulePermissions(ctx, moduleName)
	if !found {
		return fmt.Errorf("module %s has no permissions", moduleName)
	}
	
	// Find and remove the permission
	newPermissions := []string{}
	permissionFound := false
	
	for _, perm := range mp.Permissions {
		if perm != permission {
			newPermissions = append(newPermissions, perm)
		} else {
			permissionFound = true
		}
	}
	
	if !permissionFound {
		return fmt.Errorf("module %s does not have permission %s", moduleName, permission)
	}
	
	// Update permissions
	mp.Permissions = newPermissions
	
	// If no permissions left, remove the module entirely
	if len(mp.Permissions) == 0 {
		return k.RemoveModulePermissions(ctx, moduleName)
	}
	
	// Set the updated permissions
	return k.SetModulePermissions(ctx, mp)
}

// ListModules returns all modules with permissions
func (k PermissionKeeper) ListModules(ctx keepertypes.Context) []string {
	permissionSet := k.GetPermissionSet(ctx)
	
	modules := make([]string, len(permissionSet.Permissions))
	for i, mp := range permissionSet.Permissions {
		modules[i] = mp.ModuleName
	}
	
	return modules
}

// InitDefaultPermissions initializes default module permissions
func (k PermissionKeeper) InitDefaultPermissions(ctx keepertypes.Context) error {
	defaultSet := types.DefaultPermissionSet()
	k.SetPermissionSet(ctx, defaultSet)
	return nil
}