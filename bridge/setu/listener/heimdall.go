package listener

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/maticnetwork/bor/core/types"
	"github.com/maticnetwork/heimdall/helper"

	sdk "github.com/cosmos/cosmos-sdk/types"

	checkpointTypes "github.com/maticnetwork/heimdall/checkpoint/types"
	clerkTypes "github.com/maticnetwork/heimdall/clerk/types"
	slashingTypes "github.com/maticnetwork/heimdall/slashing/types"
)

const (
	heimdallLastBlockKey = "heimdall-last-block" // storage key
)

// HeimdallListener - Listens to and process events from heimdall
type HeimdallListener struct {
	BaseListener
}

// NewHeimdallListener - constructor func
func NewHeimdallListener() *HeimdallListener {
	return &HeimdallListener{}
}

// Start starts new block subscription
func (hl *HeimdallListener) Start() error {
	hl.Logger.Info("Starting")

	// create cancellable context
	headerCtx, cancelHeaderProcess := context.WithCancel(context.Background())
	hl.cancelHeaderProcess = cancelHeaderProcess

	// Heimdall pollIntervall = (minimal pollInterval of rootchain and matichain)
	pollInterval := helper.GetConfig().SyncerPollInterval
	if helper.GetConfig().CheckpointerPollInterval < helper.GetConfig().SyncerPollInterval {
		pollInterval = helper.GetConfig().CheckpointerPollInterval
	}

	hl.Logger.Info("Start polling for events", "pollInterval", pollInterval)
	hl.StartPolling(headerCtx, pollInterval)
	return nil
}

// ProcessHeader -
func (hl *HeimdallListener) ProcessHeader(*types.Header) {

}

// StartPolling - starts polling for heimdall events
func (hl *HeimdallListener) StartPolling(ctx context.Context, pollInterval time.Duration) {
	// How often to fire the passed in function in second
	interval := pollInterval

	// Setup the ticket and the channel to signal
	// the ending of the interval
	ticker := time.NewTicker(interval)

	var eventTypes []string
	eventTypes = append(eventTypes, "message.action='checkpoint'")
	eventTypes = append(eventTypes, "message.action='event-record'")
	eventTypes = append(eventTypes, "message.action='tick'")
	// ADD EVENT TYPE for SLASH-LIMIT

	// start listening
	for {
		select {
		case <-ticker.C:
			fromBlock, toBlock := hl.fetchFromAndToBlock()
			if fromBlock < toBlock {
				for _, eventType := range eventTypes {
					var query []string
					query = append(query, eventType)
					query = append(query, fmt.Sprintf("tx.height>=%v", fromBlock))
					query = append(query, fmt.Sprintf("tx.height<=%v", toBlock))

					limit := 50
					for page := 1; page > 0; {
						searchResult, err := helper.QueryTxsByEvents(hl.cliCtx, query, page, limit)
						hl.Logger.Debug("Fetching new events using search query", "query", query, "page", page, "limit", limit)

						if err != nil {
							hl.Logger.Error("Error while searching events", "eventType", eventType, "error", err)
							break
						}

						for _, tx := range searchResult.Txs {
							for _, log := range tx.Logs {
								event := helper.FilterEvents(log.Events, func(et sdk.StringEvent) bool {
									return et.Type == checkpointTypes.EventTypeCheckpoint || et.Type == clerkTypes.EventTypeRecord || et.Type == slashingTypes.EventTypeTickConfirm || et.Type == slashingTypes.EventTypeSlashLimit
								})
								if event != nil {
									hl.ProcessEvent(*event, tx)
								}
							}
						}

						if len(searchResult.Txs) == limit {
							page = page + 1
						} else {
							page = 0
						}
					}
				}
				// set last block to storage
				hl.storageClient.Put([]byte(heimdallLastBlockKey), []byte(strconv.FormatUint(toBlock, 10)), nil)
			}

		case <-ctx.Done():
			hl.Logger.Info("Polling stopped")
			ticker.Stop()
			return
		}
	}
}

func (hl *HeimdallListener) fetchFromAndToBlock() (fromBlock uint64, toBlock uint64) {
	// toBlock - get latest blockheight from heimdall node
	nodeStatus, _ := helper.GetNodeStatus(hl.cliCtx)
	toBlock = uint64(nodeStatus.SyncInfo.LatestBlockHeight)

	// fromBlock - get last block from storage
	hasLastBlock, _ := hl.storageClient.Has([]byte(heimdallLastBlockKey), nil)
	if hasLastBlock {
		lastBlockBytes, err := hl.storageClient.Get([]byte(heimdallLastBlockKey), nil)
		if err != nil {
			hl.Logger.Info("Error while fetching last block bytes from storage", "error", err)
			return
		}

		if result, err := strconv.ParseUint(string(lastBlockBytes), 10, 64); err == nil {
			hl.Logger.Debug("Got last block from bridge storage", "lastBlock", result)
			fromBlock = uint64(result) + 1
		} else {
			hl.Logger.Info("Error parsing last block bytes from storage", "error", err)
			toBlock = 0
			return
		}
	}
	return
}

// ProcessEvent - process event from heimdall.
func (hl *HeimdallListener) ProcessEvent(event sdk.StringEvent, tx sdk.TxResponse) {
	hl.Logger.Info("Process received event from Heimdall", "eventType", event.Type)
	eventBytes, err := json.Marshal(event)
	if err != nil {
		hl.Logger.Error("Error while parsing event", "error", err, "eventType", event.Type)
		return
	}

	// txBytes, err := json.Marshal(tx)
	// if err != nil {
	// 	hl.Logger.Error("Error while parsing tx", "error", err, "txHash", tx.TxHash)
	// 	return
	// }

	switch event.Type {
	case clerkTypes.EventTypeRecord:
		hl.sendTask("sendDepositRecordToMatic", eventBytes, tx.Height, tx.TxHash)
	case checkpointTypes.EventTypeCheckpoint:
		hl.sendTask("sendCheckpointToRootchain", eventBytes, tx.Height, tx.TxHash)
	case slashingTypes.EventTypeSlashLimit:
		hl.sendTask("sendTickToHeimdall", eventBytes, tx.Height, tx.TxHash)
	case slashingTypes.EventTypeTickConfirm:
		hl.sendTask("sendTickToRootchain", eventBytes, tx.Height, tx.TxHash)
	default:
		hl.Logger.Info("EventType mismatch", "eventType", event.Type)
	}
}

func (hl *HeimdallListener) sendTask(taskName string, eventBytes []byte, txHeight int64, txHash string) {
	// create machinery task
	signature := &tasks.Signature{
		Name: taskName,
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: string(eventBytes),
			},
			{
				Type:  "int64",
				Value: txHeight,
			},
			{
				Type:  "string",
				Value: txHash,
			},
		},
	}
	signature.RetryCount = 3
	hl.Logger.Info("Sending task", "taskName", taskName, "currentTime", time.Now())
	// send task
	_, err := hl.queueConnector.Server.SendTask(signature)
	if err != nil {
		hl.Logger.Error("Error sending task", "taskName", taskName, "error", err)
	}
}
