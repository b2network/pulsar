package types

const (
	// ModuleName defines the module name
	ModuleName = "coin"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName
)

// Store prefixes
var (
	SupplyPrefix        = []byte{0x00} // Prefix for coin supply storage
	MetadataPrefix      = []byte{0x01} // Prefix for denom metadata storage
	PermissionsPrefix   = []byte{0x02} // Prefix for module permissions storage
)

// Store keys
func SupplyKey(denom string) []byte {
	return append(SupplyPrefix, []byte(denom)...)
}

func MetadataKey(denom string) []byte {
	return append(MetadataPrefix, []byte(denom)...)
}

func PermissionKey(moduleName string) []byte {
	return append(PermissionsPrefix, []byte(moduleName)...)
}