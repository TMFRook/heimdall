package staking

import (
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/maticnetwork/bor/common"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/maticnetwork/heimdall/chainmanager"
	"github.com/maticnetwork/heimdall/helper"
	"github.com/maticnetwork/heimdall/params/subspace"
	"github.com/maticnetwork/heimdall/staking/types"
	hmTypes "github.com/maticnetwork/heimdall/types"
)

var (
	DefaultValue = []byte{0x01} // Value to store in CacheCheckpoint and CacheCheckpointACK & ValidatorSetChange Flag

	ValidatorsKey          = []byte{0x21} // prefix for each key to a validator
	ValidatorMapKey        = []byte{0x22} // prefix for each key for validator map
	CurrentValidatorSetKey = []byte{0x23} // Key to store current validator set
	StakingSequenceKey     = []byte{0x24} // prefix for each key for staking sequence map
	DividendAccountMapKey  = []byte{0x25} // prefix for each key for Dividend Account Map

)

// ModuleCommunicator manages different module interaction
type ModuleCommunicator interface {
	GetACKCount(ctx sdk.Context) uint64
	SetCoins(ctx sdk.Context, addr hmTypes.HeimdallAddress, amt sdk.Coins) sdk.Error
	GetCoins(ctx sdk.Context, addr hmTypes.HeimdallAddress) sdk.Coins
	SendCoins(ctx sdk.Context, from hmTypes.HeimdallAddress, to hmTypes.HeimdallAddress, amt sdk.Coins) sdk.Error
	CreateValiatorSigningInfo(ctx sdk.Context, valID hmTypes.ValidatorID, valSigningInfo hmTypes.ValidatorSigningInfo)
}

// Keeper stores all related data
type Keeper struct {
	cdc *codec.Codec
	// The (unexposed) keys used to access the stores from the Context.
	storeKey sdk.StoreKey
	// codespacecodespace
	codespace sdk.CodespaceType
	// param space
	paramSpace subspace.Subspace
	// chain manager keeper
	chainKeeper chainmanager.Keeper
	// module communicator
	moduleCommunicator ModuleCommunicator
}

// NewKeeper create new keeper
func NewKeeper(
	cdc *codec.Codec,
	storeKey sdk.StoreKey,
	paramSpace subspace.Subspace,
	codespace sdk.CodespaceType,
	chainKeeper chainmanager.Keeper,
	moduleCommunicator ModuleCommunicator,
) Keeper {
	keeper := Keeper{
		cdc:                cdc,
		storeKey:           storeKey,
		paramSpace:         paramSpace.WithKeyTable(types.ParamKeyTable()),
		codespace:          codespace,
		chainKeeper:        chainKeeper,
		moduleCommunicator: moduleCommunicator,
	}
	return keeper
}

// Codespace returns the codespace
func (k Keeper) Codespace() sdk.CodespaceType {
	return k.codespace
}

// Logger returns a module-specific logger
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

// GetValidatorKey drafts the validator key for addresses
func GetValidatorKey(address []byte) []byte {
	return append(ValidatorsKey, address...)
}

// GetValidatorMapKey returns validator map
func GetValidatorMapKey(address []byte) []byte {
	return append(ValidatorMapKey, address...)
}

// GetStakingSequenceKey returns staking sequence key
func GetStakingSequenceKey(sequence string) []byte {
	return append(StakingSequenceKey, []byte(sequence)...)
}

// AddValidator adds validator indexed with address
func (k *Keeper) AddValidator(ctx sdk.Context, validator hmTypes.Validator) error {
	// TODO uncomment
	//if ok:=validator.ValidateBasic(); !ok{
	//	// return error
	//}

	store := ctx.KVStore(k.storeKey)

	bz, err := hmTypes.MarshallValidator(k.cdc, validator)
	if err != nil {
		return err
	}

	// store validator with address prefixed with validator key as index
	store.Set(GetValidatorKey(validator.Signer.Bytes()), bz)
	k.Logger(ctx).Debug("Validator stored", "key", hex.EncodeToString(GetValidatorKey(validator.Signer.Bytes())), "validator", validator.String())

	// add validator to validator ID => SignerAddress map
	k.SetValidatorIDToSignerAddr(ctx, validator.ID, validator.Signer)

	return nil
}

// IsCurrentValidatorByAddress check if validator is in current validator set by signer address
func (k *Keeper) IsCurrentValidatorByAddress(ctx sdk.Context, address []byte) bool {
	// get ack count
	ackCount := k.moduleCommunicator.GetACKCount(ctx)

	// get validator info
	validator, err := k.GetValidatorInfo(ctx, address)
	if err != nil {
		return false
	}

	// check if validator is current validator
	return validator.IsCurrentValidator(ackCount)
}

// GetValidatorInfo returns validator
func (k *Keeper) GetValidatorInfo(ctx sdk.Context, address []byte) (validator hmTypes.Validator, err error) {
	store := ctx.KVStore(k.storeKey)

	// check if validator exists
	key := GetValidatorKey(address)
	if !store.Has(key) {
		return validator, errors.New("Validator not found")
	}

	// unmarshall validator and return
	validator, err = hmTypes.UnmarshallValidator(k.cdc, store.Get(key))
	if err != nil {
		return validator, err
	}

	// return true if validator
	return validator, nil
}

// GetActiveValidatorInfo returns active validator
func (k *Keeper) GetActiveValidatorInfo(ctx sdk.Context, address []byte) (validator hmTypes.Validator, err error) {
	validator, err = k.GetValidatorInfo(ctx, address)
	if err != nil {
		return validator, err
	}

	// get ack count
	ackCount := k.moduleCommunicator.GetACKCount(ctx)
	if !validator.IsCurrentValidator(ackCount) {
		return validator, errors.New("Validator is not active")
	}

	// return true if validator
	return validator, nil
}

// GetCurrentValidators returns all validators who are in validator set
func (k *Keeper) GetCurrentValidators(ctx sdk.Context) (validators []hmTypes.Validator) {
	// get ack count
	ackCount := k.moduleCommunicator.GetACKCount(ctx)

	// Get validators
	// iterate through validator list
	k.IterateValidatorsAndApplyFn(ctx, func(validator hmTypes.Validator) error {
		// check if validator is valid for current epoch
		if validator.IsCurrentValidator(ackCount) {
			// append if validator is current valdiator
			validators = append(validators, validator)
		}
		return nil
	})

	return
}

// GetSpanEligibleValidators returns current validators who are not getting deactivated in between next span
func (k *Keeper) GetSpanEligibleValidators(ctx sdk.Context) (validators []hmTypes.Validator) {
	// get ack count
	ackCount := k.moduleCommunicator.GetACKCount(ctx)

	// Get validators and iterate through validator list
	k.IterateValidatorsAndApplyFn(ctx, func(validator hmTypes.Validator) error {
		// check if validator is valid for current epoch and endEpoch is not set.
		if validator.EndEpoch == 0 && validator.IsCurrentValidator(ackCount) {
			// append if validator is current valdiator
			validators = append(validators, validator)
		}
		return nil
	})

	return
}

// GetAllValidators returns all validators
func (k *Keeper) GetAllValidators(ctx sdk.Context) (validators []*hmTypes.Validator) {
	// iterate through validators and create validator update array
	k.IterateValidatorsAndApplyFn(ctx, func(validator hmTypes.Validator) error {
		// append to list of validatorUpdates
		validators = append(validators, &validator)
		return nil
	})

	return
}

// IterateValidatorsAndApplyFn interate validators and apply the given function.
func (k *Keeper) IterateValidatorsAndApplyFn(ctx sdk.Context, f func(validator hmTypes.Validator) error) {
	store := ctx.KVStore(k.storeKey)

	// get validator iterator
	iterator := sdk.KVStorePrefixIterator(store, ValidatorsKey)
	defer iterator.Close()

	// loop through validators to get valid validators
	for ; iterator.Valid(); iterator.Next() {
		// unmarshall validator
		validator, _ := hmTypes.UnmarshallValidator(k.cdc, iterator.Value())
		// call function and return if required
		if err := f(validator); err != nil {
			return
		}
	}
}

// UpdateSigner updates validator with signer and pubkey + validator => signer map
func (k *Keeper) UpdateSigner(ctx sdk.Context, newSigner hmTypes.HeimdallAddress, newPubkey hmTypes.PubKey, prevSigner hmTypes.HeimdallAddress) error {
	// get old validator from state and make power 0
	validator, err := k.GetValidatorInfo(ctx, prevSigner.Bytes())
	if err != nil {
		k.Logger(ctx).Error("Unable to fetch valiator from store")
		return err
	}

	// copy power to reassign below
	validatorPower := validator.VotingPower
	validator.VotingPower = 0

	// update validator
	k.AddValidator(ctx, validator)

	//update signer in prev Validator
	validator.Signer = newSigner
	validator.PubKey = newPubkey
	validator.VotingPower = validatorPower

	// add updated validator to store with new key
	k.AddValidator(ctx, validator)
	return nil
}

// UpdateValidatorSetInStore adds validator set to store
func (k *Keeper) UpdateValidatorSetInStore(ctx sdk.Context, newValidatorSet hmTypes.ValidatorSet) error {
	// TODO check if we may have to delay this by 1 height to sync with tendermint validator updates
	store := ctx.KVStore(k.storeKey)

	// marshall validator set
	bz, err := k.cdc.MarshalBinaryBare(newValidatorSet)
	if err != nil {
		return err
	}

	// set validator set with CurrentValidatorSetKey as key in store
	store.Set(CurrentValidatorSetKey, bz)
	return nil
}

// GetValidatorSet returns current Validator Set from store
func (k *Keeper) GetValidatorSet(ctx sdk.Context) (validatorSet hmTypes.ValidatorSet) {
	store := ctx.KVStore(k.storeKey)
	// get current validator set from store
	bz := store.Get(CurrentValidatorSetKey)
	// unmarhsall
	k.cdc.UnmarshalBinaryBare(bz, &validatorSet)

	// return validator set
	return validatorSet
}

// IncrementAccum increments accum for validator set by n times and replace validator set in store
func (k *Keeper) IncrementAccum(ctx sdk.Context, times int) {
	// get validator set
	validatorSet := k.GetValidatorSet(ctx)

	// increment accum
	validatorSet.IncrementProposerPriority(times)

	// replace
	k.UpdateValidatorSetInStore(ctx, validatorSet)
}

// GetNextProposer returns next proposer
func (k *Keeper) GetNextProposer(ctx sdk.Context) *hmTypes.Validator {
	// get validator set
	validatorSet := k.GetValidatorSet(ctx)

	// Increment accum in copy
	copiedValidatorSet := validatorSet.CopyIncrementProposerPriority(1)

	// get signer address for next signer
	return copiedValidatorSet.GetProposer()
}

// GetCurrentProposer returns current proposer
func (k *Keeper) GetCurrentProposer(ctx sdk.Context) *hmTypes.Validator {
	// get validator set
	validatorSet := k.GetValidatorSet(ctx)

	// return get proposer
	return validatorSet.GetProposer()
}

// SetValidatorIDToSignerAddr sets mapping for validator ID to signer address
func (k *Keeper) SetValidatorIDToSignerAddr(ctx sdk.Context, valID hmTypes.ValidatorID, signerAddr hmTypes.HeimdallAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Set(GetValidatorMapKey(valID.Bytes()), signerAddr.Bytes())
}

// GetSignerFromValidatorID get signer address from validator ID
func (k *Keeper) GetSignerFromValidatorID(ctx sdk.Context, valID hmTypes.ValidatorID) (common.Address, bool) {
	store := ctx.KVStore(k.storeKey)
	key := GetValidatorMapKey(valID.Bytes())
	// check if validator address has been mapped
	if !store.Has(key) {
		return helper.ZeroAddress, false
	}
	// return address from bytes
	return common.BytesToAddress(store.Get(key)), true
}

// GetValidatorFromValID returns signer from validator ID
func (k *Keeper) GetValidatorFromValID(ctx sdk.Context, valID hmTypes.ValidatorID) (validator hmTypes.Validator, ok bool) {
	signerAddr, ok := k.GetSignerFromValidatorID(ctx, valID)
	if !ok {
		return validator, ok
	}
	// query for validator signer address
	validator, err := k.GetValidatorInfo(ctx, signerAddr.Bytes())
	if err != nil {
		return validator, false
	}
	return validator, true
}

// GetLastUpdated get last updated at for validator
func (k *Keeper) GetLastUpdated(ctx sdk.Context, valID hmTypes.ValidatorID) (updatedAt string, found bool) {
	// get validator
	validator, ok := k.GetValidatorFromValID(ctx, valID)
	if !ok {
		return "", false
	}
	return validator.LastUpdated, true
}

// GetDividendAccountMapKey returns dividend account map
func GetDividendAccountMapKey(id []byte) []byte {
	return append(DividendAccountMapKey, id...)
}

// AddDividendAccount adds DividendAccount index with DividendID
func (k *Keeper) AddDividendAccount(ctx sdk.Context, dividendAccount hmTypes.DividendAccount) error {
	store := ctx.KVStore(k.storeKey)
	// marshall dividend account
	bz, err := hmTypes.MarshallDividendAccount(k.cdc, dividendAccount)
	if err != nil {
		return err
	}

	store.Set(GetDividendAccountMapKey(dividendAccount.ID.Bytes()), bz)
	k.Logger(ctx).Debug("DividendAccount Stored", "key", hex.EncodeToString(GetDividendAccountMapKey(dividendAccount.ID.Bytes())), "dividendAccount", dividendAccount.String())
	return nil
}

// GetDividendAccountByID will return DividendAccount of valID
func (k *Keeper) GetDividendAccountByID(ctx sdk.Context, dividendID hmTypes.DividendAccountID) (dividendAccount hmTypes.DividendAccount, err error) {

	// check if dividend account exists
	if !k.CheckIfDividendAccountExists(ctx, dividendID) {
		return dividendAccount, errors.New("Dividend Account not found")
	}

	// Get DividendAccount key
	store := ctx.KVStore(k.storeKey)
	key := GetDividendAccountMapKey(dividendID.Bytes())

	// unmarshall dividend account and return
	dividendAccount, err = hmTypes.UnMarshallDividendAccount(k.cdc, store.Get(key))
	if err != nil {
		return dividendAccount, err
	}

	return dividendAccount, nil
}

// CheckIfDividendAccountExists will return true if dividend account exists
func (k *Keeper) CheckIfDividendAccountExists(ctx sdk.Context, dividendID hmTypes.DividendAccountID) (ok bool) {
	store := ctx.KVStore(k.storeKey)
	key := GetDividendAccountMapKey(dividendID.Bytes())
	if !store.Has(key) {
		return false
	}
	return true
}

// GetAllDividendAccounts returns all DividendAccountss
func (k *Keeper) GetAllDividendAccounts(ctx sdk.Context) (dividendAccounts []hmTypes.DividendAccount) {
	// iterate through dividendAccounts and create dividendAccounts update array
	k.IterateDividendAccountsByPrefixAndApplyFn(ctx, DividendAccountMapKey, func(dividendAccount hmTypes.DividendAccount) error {
		// append to list of dividendUpdates
		dividendAccounts = append(dividendAccounts, dividendAccount)
		return nil
	})

	return
}

// AddFeeToDividendAccount adds fee to dividend account for withdrawal
func (k *Keeper) AddFeeToDividendAccount(ctx sdk.Context, valID hmTypes.ValidatorID, fee *big.Int) sdk.Error {
	// Get or create dividend account
	var dividendAccount hmTypes.DividendAccount

	if k.CheckIfDividendAccountExists(ctx, hmTypes.DividendAccountID(valID)) {
		dividendAccount, _ = k.GetDividendAccountByID(ctx, hmTypes.DividendAccountID(valID))
	} else {
		dividendAccount = hmTypes.DividendAccount{
			ID:            hmTypes.DividendAccountID(valID),
			FeeAmount:     big.NewInt(0).String(),
			SlashedAmount: big.NewInt(0).String(),
		}
	}

	// update fee
	oldFee, _ := big.NewInt(0).SetString(dividendAccount.FeeAmount, 10)
	totalFee := big.NewInt(0).Add(oldFee, fee).String()
	dividendAccount.FeeAmount = totalFee

	k.Logger(ctx).Info("Dividend Account fee of validator ", "ID", dividendAccount.ID, "Fee", dividendAccount.FeeAmount)
	k.AddDividendAccount(ctx, dividendAccount)
	return nil
}

// IterateDividendAccountsByPrefixAndApplyFn iterate dividendAccounts and apply the given function.
func (k *Keeper) IterateDividendAccountsByPrefixAndApplyFn(ctx sdk.Context, prefix []byte, f func(dividendAccount hmTypes.DividendAccount) error) {
	store := ctx.KVStore(k.storeKey)

	// get validator iterator
	iterator := sdk.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	// loop through dividendAccounts
	for ; iterator.Valid(); iterator.Next() {
		// unmarshall dividendAccount
		dividendAccount, _ := hmTypes.UnMarshallDividendAccount(k.cdc, iterator.Value())
		// call function and return if required
		if err := f(dividendAccount); err != nil {
			return
		}
	}
}

// IterateCurrentValidatorsAndApplyFn iterate through current validators
func (k *Keeper) IterateCurrentValidatorsAndApplyFn(ctx sdk.Context, f func(validator *hmTypes.Validator) bool) {
	currentValidatorSet := k.GetValidatorSet(ctx)
	for _, v := range currentValidatorSet.Validators {
		if stop := f(v); stop {
			return
		}
	}
}

//
// Staking sequence
//

// SetStakingSequence sets staking sequence
func (k *Keeper) SetStakingSequence(ctx sdk.Context, sequence string) {
	store := ctx.KVStore(k.storeKey)

	store.Set(GetStakingSequenceKey(sequence), DefaultValue)
}

// HasStakingSequence checks if staking sequence already exists
func (k *Keeper) HasStakingSequence(ctx sdk.Context, sequence string) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(GetStakingSequenceKey(sequence))
}

// GetStakingSequences checks if Staking already exists
func (k *Keeper) GetStakingSequences(ctx sdk.Context) (sequences []string) {
	k.IterateStakingSequencesAndApplyFn(ctx, func(sequence string) error {
		sequences = append(sequences, sequence)
		return nil
	})
	return
}

// IterateStakingSequencesAndApplyFn interate validators and apply the given function.
func (k *Keeper) IterateStakingSequencesAndApplyFn(ctx sdk.Context, f func(sequence string) error) {
	store := ctx.KVStore(k.storeKey)

	// get sequence iterator
	iterator := sdk.KVStorePrefixIterator(store, StakingSequenceKey)
	defer iterator.Close()

	// loop through validators to get valid validators
	for ; iterator.Valid(); iterator.Next() {
		sequence := string(iterator.Key()[len(StakingSequenceKey):])

		// call function and return if required
		if err := f(sequence); err != nil {
			return
		}
	}
	return
}

// Slash a validator for an infraction committed at a known height
// Find the contributing stake at that height and burn the specified slashFactor
// of it, updating unbonding delegations & redelegations appropriately
//
// CONTRACT:
//    slashFactor is non-negative
// CONTRACT:
//    Infraction was committed equal to or less than an unbonding period in the past,
//    so all unbonding delegations and redelegations from that height are stored
// CONTRACT:
//    Slash will not slash unbonded validators (for the above reason)
// CONTRACT:
//    Infraction was committed at the current height or at a past height,
//    not at a height in the future

func (k Keeper) AddValidatorSigningInfo(ctx sdk.Context, valID hmTypes.ValidatorID, valSigningInfo hmTypes.ValidatorSigningInfo) error {
	k.moduleCommunicator.CreateValiatorSigningInfo(ctx, valID, valSigningInfo)
	return nil
}

// UpdatePower updates validator with signer and pubkey + validator => signer map
func (k *Keeper) Slash(ctx sdk.Context, valSlashingInfo hmTypes.ValidatorSlashingInfo) error {
	// get validator from state
	validator, found := k.GetValidatorFromValID(ctx, valSlashingInfo.ID)
	if !found {
		k.Logger(ctx).Error("Unable to fetch valiator from store")
		// TODO slashing - return proper error
		return nil
	}

	// calculate power after slash
	slashAmount, _ := helper.GetAmountFromString(valSlashingInfo.SlashedAmount)
	slashPower, _ := helper.GetPowerFromAmount(slashAmount)
	updatedPower := validator.VotingPower - slashPower.Int64()

	// update power and jail status
	validator.VotingPower = updatedPower
	validator.Jailed = valSlashingInfo.IsJailed

	// add updated validator to store with new key
	k.AddValidator(ctx, validator)
	return nil
}

// jail a validator
func (k Keeper) Jail(ctx sdk.Context, addr []byte) {
	// validator, err := k.GetValidatorInfo(ctx, addr)
	// if err != nil {

	// }

	// logger := k.Logger(ctx)
	// logger.Info(fmt.Sprintf("validator %s jailed", consAddr))

}

// unjail a validator
func (k Keeper) Unjail(ctx sdk.Context, valID hmTypes.ValidatorID) (ok bool) {

	// get validator from state and make jailed = false
	validator, found := k.GetValidatorFromValID(ctx, valID)
	if !found {
		k.Logger(ctx).Error("Unable to fetch valiator from store")
		return
	}

	if !validator.Jailed {
		k.Logger(ctx).Info("Already unjailed.")
		return
	}
	// unjail validator
	validator.Jailed = false

	// add updated validator to store with new key
	k.AddValidator(ctx, validator)
	return true

}
