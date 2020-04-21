package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	hmCommon "github.com/maticnetwork/heimdall/common"
	"github.com/maticnetwork/heimdall/types"
)

//
// Fee token
//

// MsgTopup - high level transaction of the fee coin module
type MsgTopup struct {
	FromAddress types.HeimdallAddress `json:"from_address"`
	ID          types.ValidatorID     `json:"id"`
	TxHash      types.HeimdallHash    `json:"tx_hash"`
	LogIndex    uint64                `json:"log_index"`
}

var _ sdk.Msg = MsgTopup{}

// NewMsgTopup - construct arbitrary multi-in, multi-out send msg.
func NewMsgTopup(
	fromAddr types.HeimdallAddress,
	id uint64,
	txhash types.HeimdallHash,
	logIndex uint64,
) MsgTopup {
	return MsgTopup{
		FromAddress: fromAddr,
		ID:          types.NewValidatorID(id),
		TxHash:      txhash,
		LogIndex:    logIndex,
	}
}

// Route Implements Msg.
func (msg MsgTopup) Route() string {
	return RouterKey
}

// Type Implements Msg.
func (msg MsgTopup) Type() string {
	return "deposit"
}

// ValidateBasic Implements Msg.
func (msg MsgTopup) ValidateBasic() sdk.Error {
	if msg.FromAddress.Empty() {
		return sdk.ErrInvalidAddress("missing sender address")
	}

	if msg.ID <= 0 {
		return hmCommon.ErrInvalidMsg(hmCommon.DefaultCodespace, "Invalid validator ID %v", msg.ID)
	}

	if msg.FromAddress.Empty() {
		return hmCommon.ErrInvalidMsg(hmCommon.DefaultCodespace, "Invalid proposer %v", msg.FromAddress.String())
	}

	return nil
}

// GetSignBytes Implements Msg.
func (msg MsgTopup) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners Implements Msg.
func (msg MsgTopup) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{types.HeimdallAddressToAccAddress(msg.FromAddress)}
}

// GetTxHash Returns tx hash
func (msg MsgTopup) GetTxHash() types.HeimdallHash {
	return msg.TxHash
}

// GetLogIndex Returns log index
func (msg MsgTopup) GetLogIndex() uint64 {
	return msg.LogIndex
}

//
// Fee token withdrawal
//

// MsgWithdrawFee - high level transaction of the fee coin withdrawal module
type MsgWithdrawFee struct {
	ValidatorAddress types.HeimdallAddress `json:"from_address"`
	Amount           sdk.Int               `json:"amount"`
}

var _ sdk.Msg = MsgWithdrawFee{}

// NewMsgWithdrawFee - construct arbitrary fee withdraw msg
func NewMsgWithdrawFee(
	fromAddr types.HeimdallAddress,
	amount sdk.Int,
) MsgWithdrawFee {
	return MsgWithdrawFee{
		ValidatorAddress: fromAddr,
		Amount:           amount,
	}
}

// Route Implements Msg.
func (msg MsgWithdrawFee) Route() string {
	return RouterKey
}

// Type Implements Msg.
func (msg MsgWithdrawFee) Type() string {
	return "withdraw"
}

// ValidateBasic Implements Msg.
func (msg MsgWithdrawFee) ValidateBasic() sdk.Error {
	if msg.ValidatorAddress.Empty() {
		return sdk.ErrInvalidAddress("missing sender address")
	}

	if msg.ValidatorAddress.Empty() {
		return hmCommon.ErrInvalidMsg(hmCommon.DefaultCodespace, "Invalid proposer %v", msg.ValidatorAddress.String())
	}

	return nil
}

// GetSignBytes Implements Msg.
func (msg MsgWithdrawFee) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners Implements Msg.
func (msg MsgWithdrawFee) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{types.HeimdallAddressToAccAddress(msg.ValidatorAddress)}
}
