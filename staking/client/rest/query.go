package rest

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/gorilla/mux"
	"github.com/maticnetwork/bor/common"

	"github.com/maticnetwork/heimdall/staking/types"
	hmTypes "github.com/maticnetwork/heimdall/types"
	hmRest "github.com/maticnetwork/heimdall/types/rest"
)

func registerQueryRoutes(cliCtx context.CLIContext, r *mux.Router) {
	r.HandleFunc(
		"/staking/totalpower",
		getTotalValidatorPower(cliCtx),
	).Methods("GET")
	r.HandleFunc(
		"/staking/signer/{address}",
		validatorByAddressHandlerFn(cliCtx),
	).Methods("GET")
	r.HandleFunc(
		"/staking/validator-status/{address}",
		validatorStatusByAddreesHandlerFn(cliCtx),
	).Methods("GET")
	r.HandleFunc(
		"/staking/validator/{id}",
		validatorByIDHandlerFn(cliCtx),
	).Methods("GET")
	r.HandleFunc(
		"/staking/validator-set",
		validatorSetHandlerFn(cliCtx),
	).Methods("GET")
	r.HandleFunc(
		"/staking/proposer/{times}",
		proposerHandlerFn(cliCtx),
	).Methods("GET")
	r.HandleFunc(
		"/staking/current-proposer",
		currentProposerHandlerFn(cliCtx),
	).Methods("GET")
	r.HandleFunc(
		"/staking/slash-validator",
		slashValidatorHandlerFn(cliCtx),
	).Methods("GET")
	r.HandleFunc(
		"/staking/dividend-account/{id}",
		dividendAccountByIDHandlerFn(cliCtx),
	).Methods("GET")
	r.HandleFunc(
		"/staking/dividend-account-root",
		dividendAccountRootHandlerFn(cliCtx),
	).Methods("GET")
	r.HandleFunc(
		"/staking/proposer-bonus-percent",
		proposerBonusPercentHandlerFn(cliCtx),
	).Methods("GET")
	r.HandleFunc(
		"/staking/account-proof/{id}",
		dividendAccountProofHandlerFn(cliCtx),
	).Methods("GET")
	r.HandleFunc(
		"/staking/account-proof/verify/",
		VerifyAccountProofHandlerFn(cliCtx),
	).Methods("GET")
	r.HandleFunc(
		"/staking/isoldtx",
		StakingTxStatusHandlerFn(cliCtx),
	).Methods("GET")
}

// Returns total power of current validator set
func getTotalValidatorPower(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}

		RestLogger.Debug("Fetching total validator power")
		totalPowerBytes, height, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QueryTotalValidatorPower), nil)
		if err != nil {
			hmRest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		// check content
		if ok := hmRest.ReturnNotFoundIfNoContent(w, totalPowerBytes, "total power not found"); !ok {
			return
		}

		var totalPower uint64
		if err := json.Unmarshal(totalPowerBytes, &totalPower); err != nil {
			hmRest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		result, err := json.Marshal(map[string]interface{}{"result": totalPower})
		if err != nil {
			RestLogger.Error("Error while marshalling resposne to Json", "error", err)
			hmRest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		cliCtx = cliCtx.WithHeight(height)
		rest.PostProcessResponse(w, cliCtx, result)

	}

}

// Returns validator information by signer address
func validatorByAddressHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		signerAddress := common.HexToAddress(vars["address"])

		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}

		// get query params
		queryParams, err := cliCtx.Codec.MarshalJSON(types.NewQuerySignerParams(signerAddress.Bytes()))
		if err != nil {
			hmRest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		res, height, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QuerySigner), queryParams)
		if err != nil {
			RestLogger.Error("Error while fetching signer", "Error", err.Error())
			hmRest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// error if no checkpoint found
		if ok := hmRest.ReturnNotFoundIfNoContent(w, res, "No checkpoint found"); !ok {
			return
		}

		// return result
		cliCtx = cliCtx.WithHeight(height)
		rest.PostProcessResponse(w, cliCtx, res)
	}
}

// Returns validator status information by signer address
func validatorStatusByAddreesHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		signerAddress := common.HexToAddress(vars["address"])

		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}

		// get query params
		queryParams, err := cliCtx.Codec.MarshalJSON(types.NewQuerySignerParams(signerAddress.Bytes()))
		if err != nil {
			hmRest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		statusBytes, height, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QueryValidatorStatus), queryParams)
		if err != nil {
			RestLogger.Error("Error while fetching validator status", "Error", err.Error())
			hmRest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// error if no checkpoint found
		if ok := hmRest.ReturnNotFoundIfNoContent(w, statusBytes, "No validator found"); !ok {
			return
		}

		var status bool
		if err := json.Unmarshal(statusBytes, &status); err != nil {
			hmRest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		res, err := json.Marshal(map[string]interface{}{"result": status})
		if err != nil {
			hmRest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// return result
		cliCtx = cliCtx.WithHeight(height)
		rest.PostProcessResponse(w, cliCtx, res)
	}
}

// Returns validator information by val ID
func validatorByIDHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}

		// get id
		id, ok := rest.ParseUint64OrReturnBadRequest(w, vars["id"])
		if !ok {
			return
		}

		// get query params
		queryParams, err := cliCtx.Codec.MarshalJSON(types.NewQueryValidatorParams(hmTypes.ValidatorID(id)))
		if err != nil {
			hmRest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		res, height, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QueryValidator), queryParams)
		if err != nil {
			RestLogger.Error("Error while fetching validator", "Error", err.Error())
			hmRest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// error if no checkpoint found
		if ok := hmRest.ReturnNotFoundIfNoContent(w, res, "No checkpoint found"); !ok {
			return
		}

		// return result
		cliCtx = cliCtx.WithHeight(height)
		rest.PostProcessResponse(w, cliCtx, res)
	}
}

// get current validator set
func validatorSetHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}

		res, height, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QueryCurrentValidatorSet), nil)
		if err != nil {
			RestLogger.Error("Error while fetching current validator set ", "Error", err.Error())
			hmRest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// error if no checkpoint found
		if ok := hmRest.ReturnNotFoundIfNoContent(w, res, "No checkpoint found"); !ok {
			return
		}

		// return result
		cliCtx = cliCtx.WithHeight(height)
		rest.PostProcessResponse(w, cliCtx, res)
	}
}

// get proposer for current validator set
func proposerHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}

		// get proposer times
		times, ok := rest.ParseUint64OrReturnBadRequest(w, vars["times"])
		if !ok {
			return
		}

		// get query params
		queryParams, err := cliCtx.Codec.MarshalJSON(types.NewQueryProposerParams(times))
		if err != nil {
			hmRest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		res, height, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QueryProposer), queryParams)
		if err != nil {
			RestLogger.Error("Error while fetching proposers ", "Error", err.Error())
			hmRest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// error if no checkpoint found
		if ok := hmRest.ReturnNotFoundIfNoContent(w, res, "No proposer found"); !ok {
			return
		}

		// return result
		cliCtx = cliCtx.WithHeight(height)
		rest.PostProcessResponse(w, cliCtx, res)
	}
}

// currentProposerHandlerFn get proposer for current validator set
func currentProposerHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}

		res, height, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QueryCurrentProposer), nil)
		if err != nil {
			RestLogger.Error("Error while fetching current proposer ", "Error", err.Error())
			hmRest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// error if no checkpoint found
		if ok := hmRest.ReturnNotFoundIfNoContent(w, res, "No proposer found"); !ok {
			return
		}

		// return result
		cliCtx = cliCtx.WithHeight(height)
		rest.PostProcessResponse(w, cliCtx, res)
	}
}

func slashValidatorHandlerFn(
	cliCtx context.CLIContext,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}

		params := r.URL.Query()
		valID, ok := rest.ParseUint64OrReturnBadRequest(w, params.Get("val_id"))
		if !ok {
			return
		}

		slashAmount, ok := big.NewInt(0).SetString(params.Get("slash_amount"), 10)
		if !ok {
			return
		}

		RestLogger.Info("Slashing validator - ", "valID", valID, "slashAmount", slashAmount)

		slashingParams := types.NewValidatorSlashParams(hmTypes.ValidatorID(valID), slashAmount)

		bz, err := cliCtx.Codec.MarshalJSON(slashingParams)
		if err != nil {
			RestLogger.Info("Error marshallingSlashing Params - ", "err", err)
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		res, height, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QuerySlashValidator), bz)
		if err != nil {
			RestLogger.Info("Error query data - ", "err", err)
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		RestLogger.Info("Slashed validator successfully ", "res", res)

		cliCtx.WithHeight(height)
		rest.PostProcessResponse(w, cliCtx, res)
	}
}

// Returns Dividend Account information by ID
func dividendAccountByIDHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}

		// get id
		id, ok := rest.ParseUint64OrReturnBadRequest(w, vars["id"])
		if !ok {
			return
		}

		// get query params
		queryParams, err := cliCtx.Codec.MarshalJSON(types.NewQueryDividendAccountParams(hmTypes.DividendAccountID(id)))
		if err != nil {
			hmRest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		res, height, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QueryDividendAccount), queryParams)
		if err != nil {
			RestLogger.Error("Error while fetching Dividend account", "Error", err.Error())
			hmRest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// error if no dividend account found
		if ok := hmRest.ReturnNotFoundIfNoContent(w, res, "No Dividend Account found"); !ok {
			return
		}

		// return result
		cliCtx = cliCtx.WithHeight(height)
		rest.PostProcessResponse(w, cliCtx, res)
	}
}

// dividendAccountRootHandlerFn returns genesis accountroothash
func dividendAccountRootHandlerFn(
	cliCtx context.CLIContext,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}

		res, _, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QueryDividendAccountRoot), nil)
		RestLogger.Debug("accountRootHash querier response", "res", res)

		if err != nil {
			RestLogger.Error("Error while calculating dividend AccountRoot  ", "Error", err.Error())
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// error if no checkpoint found
		if ok := hmRest.ReturnNotFoundIfNoContent(w, res, "Dividend AccountRoot not found"); !ok {
			RestLogger.Error("AccountRoot not found ", "Error", err.Error())
			return
		}

		var accountRootHash = hmTypes.BytesToHeimdallHash(res)
		RestLogger.Debug("Fetched Dividend accountRootHash ", "AccountRootHash", accountRootHash)

		result, err := json.Marshal(&accountRootHash)
		if err != nil {
			RestLogger.Error("Error while marshalling response to Json", "error", err)
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// return result

		rest.PostProcessResponse(w, cliCtx, result)
	}
}

// Returns proposer Bonus Percent information
func proposerBonusPercentHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}

		// fetch state reocrd
		res, _, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QueryProposerBonusPercent), nil)
		if err != nil {
			RestLogger.Error("Error while fetching proposer bonus percentage", "Error", err.Error())
			hmRest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		// error if no checkpoint found
		if ok := hmRest.ReturnNotFoundIfNoContent(w, res, "Proposer bonus percentage not found"); !ok {
			RestLogger.Error("Proposer bonus percentage not found ", "Error", err.Error())
			return
		}

		var _proposerBonusPercent int64
		if err := json.Unmarshal(res, &_proposerBonusPercent); err != nil {
			hmRest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		result, err := json.Marshal(_proposerBonusPercent)
		if err != nil {
			RestLogger.Error("Error while marshalling resposne to Json", "error", err)
			hmRest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		rest.PostProcessResponse(w, cliCtx, result)
	}
}

// Returns Merkle path for dividendAccountID using dividend Account Tree
func dividendAccountProofHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)

		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}

		// get id
		id, ok := rest.ParseUint64OrReturnBadRequest(w, vars["id"])
		if !ok {
			return
		}

		// get query params
		queryParams, err := cliCtx.Codec.MarshalJSON(types.NewQueryAccountProofParams(hmTypes.DividendAccountID(id)))
		if err != nil {
			hmRest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		res, height, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QueryAccountProof), queryParams)
		if err != nil {
			RestLogger.Error("Error while fetching merkle proof", "Error", err.Error())
			hmRest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// error if account proof  not found
		if ok := hmRest.ReturnNotFoundIfNoContent(w, res, "No proof found"); !ok {
			return
		}

		// return result
		cliCtx = cliCtx.WithHeight(height)
		rest.PostProcessResponse(w, cliCtx, res)
	}
}

// VerifyAccountProofHandlerFn - Returns true if given Merkle path for dividendAccountID is valid
func VerifyAccountProofHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}

		params := r.URL.Query()
		id, ok := rest.ParseUint64OrReturnBadRequest(w, params.Get("id"))
		if !ok {
			return
		}

		accountProof := params.Get("proof")

		RestLogger.Info("Verify Account Proof- ", "valID", id, "accountProof", accountProof)

		// get query params
		queryParams, err := cliCtx.Codec.MarshalJSON(types.NewQueryVerifyAccountProofParams(hmTypes.DividendAccountID(id), accountProof))
		if err != nil {
			hmRest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		res, height, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QueryVerifyAccountProof), queryParams)
		if err != nil {
			RestLogger.Error("Error while verifying merkle proof", "Error", err.Error())
			hmRest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		var accountProofStatus bool
		if err := json.Unmarshal(res, &accountProofStatus); err != nil {
			hmRest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		res, err = json.Marshal(map[string]interface{}{"result": accountProofStatus})
		if err != nil {
			hmRest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// return result
		cliCtx = cliCtx.WithHeight(height)
		rest.PostProcessResponse(w, cliCtx, res)

	}
}

// Returns staking tx status information
func StakingTxStatusHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := r.URL.Query()

		cliCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, cliCtx, r)
		if !ok {
			return
		}

		// get logIndex
		logindex, ok := rest.ParseUint64OrReturnBadRequest(w, vars.Get("logindex"))
		if !ok {
			return
		}

		txHash := vars.Get("txhash")

		if txHash == "" {
			return
		}

		// get query params
		queryParams, err := cliCtx.Codec.MarshalJSON(types.NewQueryStakingSequenceParams(txHash, logindex))
		if err != nil {
			hmRest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		seqNo, height, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QueryStakingSequence), queryParams)
		if err != nil {
			RestLogger.Error("Error while fetching staking sequence", "Error", err.Error())
			hmRest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		// error if no tx status found
		if ok := hmRest.ReturnNotFoundIfNoContent(w, seqNo, "No sequence found"); !ok {
			return
		}

		res := true

		// return result
		cliCtx = cliCtx.WithHeight(height)
		rest.PostProcessResponse(w, cliCtx, res)
	}
}
