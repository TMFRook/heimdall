package types

import (
	"math/big"

	"github.com/maticnetwork/heimdall/types"
	hmTypes "github.com/maticnetwork/heimdall/types"
)

// query endpoints supported by the staking Querier
const (
	QueryCurrentValidatorSet  = "current-validator-set"
	QuerySigner               = "signer"
	QueryTotalValidatorPower  = "total-val-power"
	QueryValidator            = "validator"
	QueryValidatorStatus      = "validator-status"
	QueryProposer             = "proposer"
	QueryCurrentProposer      = "current-proposer"
	QueryProposerBonusPercent = "proposer-bonus-percent"
	QueryDividendAccount      = "dividend-account"
	QueryDividendAccountRoot  = "dividend-account-root"
	QueryAccountProof         = "dividend-account-proof"
	QueryVerifyAccountProof   = "verify-account-proof"
	QuerySlashValidator       = "slash-validator"
	QueryStakingSequence      = "staking-sequence"
)

// QuerySignerParams defines the params for querying by address
type QuerySignerParams struct {
	SignerAddress []byte `json:"signer_address"`
}

// NewQuerySignerParams creates a new instance of QuerySignerParams.
func NewQuerySignerParams(signerAddress []byte) QuerySignerParams {
	return QuerySignerParams{SignerAddress: signerAddress}
}

// QueryValidatorParams defines the params for querying val status.
type QueryValidatorParams struct {
	ValidatorID types.ValidatorID `json:"validator_id"`
}

// NewQueryValidatorParams creates a new instance of QueryValidatorParams.
func NewQueryValidatorParams(validatorID types.ValidatorID) QueryValidatorParams {
	return QueryValidatorParams{ValidatorID: validatorID}
}

// QueryDividendAccountParams defines the params for querying dividend account status.
type QueryDividendAccountParams struct {
	DividendAccountID types.DividendAccountID `json:"dividend_account_id"`
}

// NewQueryDividendAccountParams creates a new instance of QueryDividendAccountParams.
func NewQueryDividendAccountParams(dividendAccountID types.DividendAccountID) QueryDividendAccountParams {
	return QueryDividendAccountParams{DividendAccountID: dividendAccountID}
}

// QueryAccountProofParams defines the params for querying account proof.
type QueryAccountProofParams struct {
	DividendAccountID types.DividendAccountID `json:"dividend_account_id"`
}

// NewQueryAccountProofParams creates a new instance of QueryAccountProofParams.
func NewQueryAccountProofParams(dividendAccountID types.DividendAccountID) QueryAccountProofParams {
	return QueryAccountProofParams{DividendAccountID: dividendAccountID}
}

// QueryVerifyAccountProofParams defines the params for verifying account proof.
type QueryVerifyAccountProofParams struct {
	DividendAccountID types.DividendAccountID `json:"dividend_account_id"`
	AccountProof      string                  `json:"account_proof"`
}

// NewQueryVerifyAccountProofParams creates a new instance of QueryVerifyAccountProofParams.
func NewQueryVerifyAccountProofParams(dividendAccountID types.DividendAccountID, accountProof string) QueryVerifyAccountProofParams {
	return QueryVerifyAccountProofParams{DividendAccountID: dividendAccountID, AccountProof: accountProof}
}

// QueryProposerParams defines the params for querying val status.
type QueryProposerParams struct {
	Times uint64 `json:"times"`
}

// NewQueryProposerParams creates a new instance of QueryProposerParams.
func NewQueryProposerParams(times uint64) QueryProposerParams {
	return QueryProposerParams{Times: times}
}

// QueryValidatorStatusParams defines the params for querying val status.
type QueryValidatorStatusParams struct {
	SignerAddress []byte
}

// QueryStakingSequenceParams defines the params for querying an account Sequence.
type QueryStakingSequenceParams struct {
	TxHash   string
	LogIndex uint64
}

// // NewQuerySequenceParams creates a new instance of QuerySequenceParams.
// func NewQuerySequenceParams(txHash string, logIndex uint64) QueryStakingSequenceParams {
// 	return QueryStakingSequenceParams{TxHash: txHash, LogIndex: logIndex}
// }

// ValidatorSlashParams defines the params for slashing a validator
type ValidatorSlashParams struct {
	ValID       hmTypes.ValidatorID
	SlashAmount *big.Int
}

// NewQueryValidatorStatusParams creates a new instance of QueryValidatorStatusParams.
func NewQueryValidatorStatusParams(signerAddress []byte) QueryValidatorStatusParams {
	return QueryValidatorStatusParams{SignerAddress: signerAddress}
}

// NewQueryStakingSequenceParams creates a new instance of QueryStakingSequenceParams.
func NewQueryStakingSequenceParams(txHash string, logIndex uint64) QueryStakingSequenceParams {
	return QueryStakingSequenceParams{TxHash: txHash, LogIndex: logIndex}
}

// NewValidatorSlashParams creates a new instance of ValidatorSlashParams.
func NewValidatorSlashParams(validatorID hmTypes.ValidatorID, amountToSlash *big.Int) ValidatorSlashParams {
	return ValidatorSlashParams{ValID: validatorID, SlashAmount: amountToSlash}
}
