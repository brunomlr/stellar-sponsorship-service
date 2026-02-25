package stellar

import (
	"net/http"
	"strconv"

	"github.com/stellar/go-stellar-sdk/txnbuild"
	"github.com/stellar/go-stellar-sdk/xdr"

	"github.com/stellar-sponsorship-service/internal/model"
)

// VerifyResult holds the outcome of transaction verification.
type VerifyResult struct {
	Valid          bool
	HTTPStatus     int
	ErrorCode      string
	ErrorMessage   string
	Operations     []string // operation type names found (excluding structural ops)
	SourceAccount  string   // transaction source account
	ReservesLocked int      // number of base reserves the transaction will lock in the sponsor
}

// Verifier validates transactions against sponsorship service rules.
type Verifier struct {
	networkPassphrase string
}

// NewVerifier creates a new transaction verifier for the given network.
func NewVerifier(networkPassphrase string) *Verifier {
	return &Verifier{networkPassphrase: networkPassphrase}
}

// Verify checks a transaction XDR against the API key's rules.
func (v *Verifier) Verify(txXDR string, apiKey *model.APIKey) VerifyResult {
	// 1. Decode XDR
	genericTx, err := txnbuild.TransactionFromXDR(txXDR)
	if err != nil {
		return rejectResult(http.StatusBadRequest, "invalid_transaction",
			"Failed to decode transaction XDR: "+err.Error())
	}

	tx, ok := genericTx.Transaction()
	if !ok {
		return rejectResult(http.StatusBadRequest, "invalid_transaction",
			"Only V1 transaction envelopes are supported (not fee bump transactions)")
	}

	// 2. Extract source account
	sourceAccount := tx.SourceAccount().AccountID
	result := VerifyResult{
		SourceAccount: sourceAccount,
	}

	// 3. Source account check — sponsor account must NEVER be the transaction source
	if sourceAccount == apiKey.SponsorAccount {
		result.ErrorCode = "sponsor_as_source"
		result.ErrorMessage = "Transaction source account matches the sponsor account — this is not allowed"
		result.HTTPStatus = http.StatusBadRequest
		return result
	}

	ops := tx.Operations()
	if len(ops) == 0 {
		return rejectResult(http.StatusBadRequest, "invalid_transaction",
			"Transaction must contain at least one operation")
	}

	// Build allowed operations set for O(1) lookup
	allowedOps := make(map[string]bool, len(apiKey.AllowedOperations))
	for _, op := range apiKey.AllowedOperations {
		allowedOps[op] = true
	}

	// Build allowed source accounts set (if configured)
	var allowedSources map[string]bool
	if len(apiKey.AllowedSourceAccounts) > 0 {
		allowedSources = make(map[string]bool, len(apiKey.AllowedSourceAccounts))
		for _, src := range apiKey.AllowedSourceAccounts {
			allowedSources[src] = true
		}
	}

	// Track sponsoring blocks for nesting validation and SponsoredID source binding.
	var sponsoredAccountStack []string
	var opNames []string
	var reservesLocked int

	// 4. Operation iteration
	for i, op := range ops {
		xdrOp, err := op.BuildXDR()
		if err != nil {
			return rejectResultWithSource(http.StatusBadRequest, "invalid_transaction",
				"Failed to build XDR for operation "+strconv.Itoa(i), sourceAccount)
		}

		opType := xdrOp.Body.Type

		// 4a. Operation source check
		opSource := getOperationSource(op, sourceAccount)

		if opSource == apiKey.SponsorAccount && opType != xdr.OperationTypeBeginSponsoringFutureReserves {
			// Sponsor account as source is only allowed for BEGIN_SPONSORING
			result.ErrorCode = "sponsor_as_source"
			result.ErrorMessage = "Operation uses the sponsor account as source — this is not allowed"
			result.HTTPStatus = http.StatusBadRequest
			return result
		}

		// 4b. Structural operations
		if opType == xdr.OperationTypeBeginSponsoringFutureReserves {
			// Validate the sponsor in BEGIN_SPONSORING matches the API key's sponsor account
			beginOp, ok := op.(*txnbuild.BeginSponsoringFutureReserves)
			if !ok {
				return rejectResultWithSource(http.StatusBadRequest, "invalid_transaction",
					"Failed to parse BEGIN_SPONSORING_FUTURE_RESERVES operation", sourceAccount)
			}

			// The source of BEGIN_SPONSORING is the sponsor — it must match our sponsor account.
			// The SponsoredID is the account being sponsored.
			beginSource := getOperationSource(op, sourceAccount)
			if beginSource != apiKey.SponsorAccount {
				result.ErrorCode = "invalid_sponsor"
				result.ErrorMessage = "BEGIN_SPONSORING_FUTURE_RESERVES source must be the sponsor account (" +
					apiKey.SponsorAccount + "), got " + beginSource
				result.HTTPStatus = http.StatusBadRequest
				result.SourceAccount = sourceAccount
				return result
			}
			if beginOp.SponsoredID == "" {
				return rejectResultWithSource(http.StatusBadRequest, "invalid_transaction",
					"BEGIN_SPONSORING_FUTURE_RESERVES missing SponsoredID", sourceAccount)
			}

			sponsoredAccountStack = append(sponsoredAccountStack, beginOp.SponsoredID)
			continue
		}

		if opType == xdr.OperationTypeEndSponsoringFutureReserves {
			if len(sponsoredAccountStack) == 0 {
				return rejectResultWithSource(http.StatusBadRequest, "invalid_transaction",
					"END_SPONSORING_FUTURE_RESERVES without matching BEGIN", sourceAccount)
			}
			sponsoredAccountStack = sponsoredAccountStack[:len(sponsoredAccountStack)-1]
			continue
		}

		// Non-structural operation from here on

		// 4c. XLM transfer check — defense in depth, runs BEFORE allowed ops check
		// so that even if an XLM-transferring op type is accidentally added to the
		// allowed list, it would still be caught here.
		if isXLMTransfer(op) {
			return rejectResultWithSource(http.StatusBadRequest, "xlm_transfer_detected",
				"Transaction attempts to transfer native XLM — this is not allowed", sourceAccount)
		}

		// 4d. Operation type check
		opName, known := OperationTypeName(opType)
		if !known {
			return rejectResultWithSource(http.StatusBadRequest, "disallowed_operation",
				"Unknown or unsupported operation type", sourceAccount)
		}
		if !allowedOps[opName] {
			result.ErrorCode = "disallowed_operation"
			result.ErrorMessage = "Operation " + opName + " is not allowed for this API key"
			result.HTTPStatus = http.StatusBadRequest
			result.SourceAccount = sourceAccount
			return result
		}

		// 4e. Sponsoring block check — non-structural ops must be inside a sponsoring block
		if len(sponsoredAccountStack) == 0 {
			result.ErrorCode = "invalid_transaction"
			result.ErrorMessage = "Operation " + opName + " must be wrapped in BEGIN_SPONSORING_FUTURE_RESERVES / END_SPONSORING_FUTURE_RESERVES"
			result.HTTPStatus = http.StatusBadRequest
			result.SourceAccount = sourceAccount
			return result
		}

		activeSponsoredID := sponsoredAccountStack[len(sponsoredAccountStack)-1]
		if opSource != activeSponsoredID {
			return rejectResultWithSource(http.StatusBadRequest, "invalid_transaction",
				"Operation source "+opSource+" does not match SponsoredID "+activeSponsoredID+
					" in active BEGIN_SPONSORING_FUTURE_RESERVES block", sourceAccount)
		}

		// 4f. Source account allowlist
		if allowedSources != nil {
			if !allowedSources[opSource] {
				result.ErrorCode = "disallowed_operation"
				result.ErrorMessage = "Operation source account " + opSource + " is not in the allowed list"
				result.HTTPStatus = http.StatusBadRequest
				result.SourceAccount = sourceAccount
				return result
			}
		}

		opNames = append(opNames, opName)
		reservesLocked += reservesForOperation(op)
	}

	// Verify all sponsoring blocks are properly closed
	if len(sponsoredAccountStack) != 0 {
		return rejectResultWithSource(http.StatusBadRequest, "invalid_transaction",
			"Unmatched BEGIN_SPONSORING_FUTURE_RESERVES — missing END", sourceAccount)
	}

	// Source account allowlist check for transaction source
	if allowedSources != nil && !allowedSources[sourceAccount] {
		return rejectResultWithSource(http.StatusBadRequest, "disallowed_operation",
			"Transaction source account "+sourceAccount+" is not in the allowed list", sourceAccount)
	}

	return VerifyResult{
		Valid:          true,
		Operations:     opNames,
		SourceAccount:  sourceAccount,
		ReservesLocked: reservesLocked,
	}
}

// getOperationSource returns the operation's explicit source account,
// or falls back to the transaction source if none is set.
func getOperationSource(op txnbuild.Operation, txSource string) string {
	src := op.GetSourceAccount()
	if src != "" {
		return src
	}
	return txSource
}

func rejectResult(httpStatus int, code, message string) VerifyResult {
	return VerifyResult{
		HTTPStatus:   httpStatus,
		ErrorCode:    code,
		ErrorMessage: message,
	}
}

func rejectResultWithSource(httpStatus int, code, message, source string) VerifyResult {
	return VerifyResult{
		HTTPStatus:    httpStatus,
		ErrorCode:     code,
		ErrorMessage:  message,
		SourceAccount: source,
	}
}
