package stellar

import (
	"github.com/stellar/go-stellar-sdk/txnbuild"
	"github.com/stellar/go-stellar-sdk/xdr"
)

// operationTypeNames maps XDR operation types to human-readable names.
// Only sponsorable operations are included.
var operationTypeNames = map[xdr.OperationType]string{
	xdr.OperationTypeCreateAccount:                 "CREATE_ACCOUNT",
	xdr.OperationTypeChangeTrust:                   "CHANGE_TRUST",
	xdr.OperationTypeManageSellOffer:               "MANAGE_SELL_OFFER",
	xdr.OperationTypeManageBuyOffer:                "MANAGE_BUY_OFFER",
	xdr.OperationTypeSetOptions:                    "SET_OPTIONS",
	xdr.OperationTypeManageData:                    "MANAGE_DATA",
	xdr.OperationTypeCreateClaimableBalance:        "CREATE_CLAIMABLE_BALANCE",
	xdr.OperationTypeBeginSponsoringFutureReserves: "BEGIN_SPONSORING_FUTURE_RESERVES",
	xdr.OperationTypeEndSponsoringFutureReserves:   "END_SPONSORING_FUTURE_RESERVES",
}

// OperationTypeName returns the string name for an XDR operation type.
func OperationTypeName(opType xdr.OperationType) (string, bool) {
	name, ok := operationTypeNames[opType]
	return name, ok
}

// isStructuralOp returns true for BEGIN/END_SPONSORING_FUTURE_RESERVES operations,
// which are structural operations that control sponsoring blocks.
func isStructuralOp(opType xdr.OperationType) bool {
	return opType == xdr.OperationTypeBeginSponsoringFutureReserves ||
		opType == xdr.OperationTypeEndSponsoringFutureReserves
}

// isXLMTransfer checks if an operation attempts to transfer native XLM.
// These are unconditionally rejected.
func isXLMTransfer(op txnbuild.Operation) bool {
	switch o := op.(type) {
	case *txnbuild.Payment:
		return o.Asset.IsNative()
	case *txnbuild.PathPaymentStrictSend:
		return o.SendAsset.IsNative() || o.DestAsset.IsNative()
	case *txnbuild.PathPaymentStrictReceive:
		return o.SendAsset.IsNative() || o.DestAsset.IsNative()
	case *txnbuild.AccountMerge:
		return true
	case *txnbuild.Inflation:
		return true
	case *txnbuild.Clawback:
		return o.Asset.IsNative()
	default:
		return false
	}
}

// reservesForOperation returns how many base reserves a sponsored operation
// will lock in the sponsor account. Returns 0 for operations that don't
// create new ledger entries (e.g. updates or deletions).
func reservesForOperation(op txnbuild.Operation) int {
	switch o := op.(type) {
	case *txnbuild.CreateAccount:
		// New account requires 2 base reserves
		return 2
	case *txnbuild.ChangeTrust:
		// Adding/changing a trustline locks 1 reserve; removing (limit "0") frees it
		if o.Limit == "0" {
			return 0
		}
		return 1
	case *txnbuild.ManageSellOffer:
		// New offer (OfferID 0) locks 1 reserve; update/delete does not
		if o.OfferID == 0 {
			return 1
		}
		return 0
	case *txnbuild.ManageBuyOffer:
		if o.OfferID == 0 {
			return 1
		}
		return 0
	case *txnbuild.SetOptions:
		// Adding a signer locks 1 reserve
		if o.Signer != nil {
			return 1
		}
		return 0
	case *txnbuild.ManageData:
		// Setting a value creates a data entry (1 reserve); nil value deletes it
		if o.Value != nil {
			return 1
		}
		return 0
	case *txnbuild.CreateClaimableBalance:
		return 1
	default:
		return 0
	}
}

// SupportedOperations returns the list of sponsorable operation type names
// (excluding structural BEGIN/END_SPONSORING operations).
func SupportedOperations() []string {
	return []string{
		"CREATE_ACCOUNT",
		"CHANGE_TRUST",
		"MANAGE_SELL_OFFER",
		"MANAGE_BUY_OFFER",
		"SET_OPTIONS",
		"MANAGE_DATA",
		"CREATE_CLAIMABLE_BALANCE",
	}
}
