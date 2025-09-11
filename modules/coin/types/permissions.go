package types

import "fmt"

// Permission types
const (
	PermissionMint = "mint"
	PermissionBurn = "burn"
)

// ModulePermissions represents permissions for a module
type ModulePermissions struct {
	ModuleName  string   `json:"module_name"`
	Permissions []string `json:"permissions"`
}

// NewModulePermissions creates new module permissions
func NewModulePermissions(moduleName string, permissions []string) ModulePermissions {
	return ModulePermissions{
		ModuleName:  moduleName,
		Permissions: permissions,
	}
}

// HasPermission checks if module has a specific permission
func (mp ModulePermissions) HasPermission(permission string) bool {
	for _, perm := range mp.Permissions {
		if perm == permission {
			return true
		}
	}
	return false
}

// ValidateBasic performs basic validation
func (mp ModulePermissions) ValidateBasic() error {
	if mp.ModuleName == "" {
		return fmt.Errorf("module name cannot be empty")
	}
	
	for i, perm := range mp.Permissions {
		if perm == "" {
			return fmt.Errorf("permission at index %d cannot be empty", i)
		}
		
		// Validate known permissions
		switch perm {
		case PermissionMint, PermissionBurn:
			// Valid permission
		default:
			return fmt.Errorf("unknown permission: %s", perm)
		}
	}
	
	return nil
}

// PermissionSet represents a set of module permissions
type PermissionSet struct {
	Permissions []ModulePermissions `json:"permissions"`
}

// NewPermissionSet creates a new permission set
func NewPermissionSet(permissions []ModulePermissions) PermissionSet {
	return PermissionSet{
		Permissions: permissions,
	}
}

// GetModulePermissions returns permissions for a specific module
func (ps PermissionSet) GetModulePermissions(moduleName string) (ModulePermissions, bool) {
	for _, mp := range ps.Permissions {
		if mp.ModuleName == moduleName {
			return mp, true
		}
	}
	return ModulePermissions{}, false
}

// HasPermission checks if a module has a specific permission
func (ps PermissionSet) HasPermission(moduleName, permission string) bool {
	mp, found := ps.GetModulePermissions(moduleName)
	if !found {
		return false
	}
	return mp.HasPermission(permission)
}

// AddModulePermissions adds permissions for a module
func (ps *PermissionSet) AddModulePermissions(mp ModulePermissions) {
	// Remove existing permissions for this module
	for i, existing := range ps.Permissions {
		if existing.ModuleName == mp.ModuleName {
			ps.Permissions = append(ps.Permissions[:i], ps.Permissions[i+1:]...)
			break
		}
	}
	
	// Add new permissions
	ps.Permissions = append(ps.Permissions, mp)
}

// RemoveModulePermissions removes all permissions for a module
func (ps *PermissionSet) RemoveModulePermissions(moduleName string) {
	for i, mp := range ps.Permissions {
		if mp.ModuleName == moduleName {
			ps.Permissions = append(ps.Permissions[:i], ps.Permissions[i+1:]...)
			break
		}
	}
}

// ValidateBasic performs basic validation on the permission set
func (ps PermissionSet) ValidateBasic() error {
	seen := make(map[string]bool)
	
	for i, mp := range ps.Permissions {
		if err := mp.ValidateBasic(); err != nil {
			return fmt.Errorf("invalid permissions at index %d: %w", i, err)
		}
		
		// Check for duplicate module names
		if seen[mp.ModuleName] {
			return fmt.Errorf("duplicate module permissions for %s", mp.ModuleName)
		}
		seen[mp.ModuleName] = true
	}
	
	return nil
}

// DefaultPermissionSet returns default module permissions
func DefaultPermissionSet() PermissionSet {
	return NewPermissionSet([]ModulePermissions{
		NewModulePermissions("mint", []string{PermissionMint}),
		NewModulePermissions("distribution", []string{PermissionMint}),
		NewModulePermissions("staking", []string{PermissionMint, PermissionBurn}),
		NewModulePermissions("gov", []string{PermissionBurn}),
		NewModulePermissions("slashing", []string{PermissionBurn}),
	})
}