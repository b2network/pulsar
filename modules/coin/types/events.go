package types

// Event types for coin module
const (
	EventTypeCoinMint         = "coin_mint"
	EventTypeCoinBurn         = "coin_burn"
	EventTypeMetadataSet      = "coin_metadata_set"
	EventTypeMetadataDeleted  = "coin_metadata_deleted"
	EventTypePermissionSet    = "coin_permission_set"
	EventTypePermissionRemoved = "coin_permission_removed"
)

// Event attribute keys
const (
	AttributeKeyMinter       = "minter"
	AttributeKeyBurner       = "burner"
	AttributeKeyDenom        = "denom"
	AttributeKeyAmount       = "amount"
	AttributeKeyModule       = "module"
	AttributeKeyPermission   = "permission"
	AttributeKeySymbol       = "symbol"
	AttributeKeyName         = "name"
	AttributeKeyDisplay      = "display"
	AttributeKeyBase         = "base"
)