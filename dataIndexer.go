package indexer

import (
	"github.com/ElrondNetwork/elastic-indexer-go/data"
	"github.com/ElrondNetwork/elastic-indexer-go/workItems"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/statistics"
	nodeData "github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/indexer"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/epochStart/notifier"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

type dataIndexer struct {
	isNilIndexer     bool
	dispatcher       DispatcherHandler
	coordinator      sharding.NodesCoordinator
	elasticProcessor ElasticProcessor
	options          *Options
	marshalizer      marshal.Marshalizer
}

// NewDataIndexer will create a new data indexer
func NewDataIndexer(arguments ArgDataIndexer) (*dataIndexer, error) {
	err := checkIndexerArgs(arguments)
	if err != nil {
		return nil, err
	}

	dataIndexerObj := &dataIndexer{
		isNilIndexer:     false,
		dispatcher:       arguments.DataDispatcher,
		coordinator:      arguments.NodesCoordinator,
		elasticProcessor: arguments.ElasticProcessor,
		marshalizer:      arguments.Marshalizer,
		options:          arguments.Options,
	}

	if arguments.ShardCoordinator.SelfId() == core.MetachainShardId {
		arguments.EpochStartNotifier.RegisterHandler(dataIndexerObj.epochStartEventHandler())
	}

	return dataIndexerObj, nil
}

func checkIndexerArgs(arguments ArgDataIndexer) error {
	if check.IfNil(arguments.DataDispatcher) {
		return ErrNilDataDispatcher
	}
	if check.IfNil(arguments.ElasticProcessor) {
		return ErrNilElasticProcessor
	}
	if check.IfNil(arguments.NodesCoordinator) {
		return core.ErrNilNodesCoordinator
	}
	if check.IfNil(arguments.EpochStartNotifier) {
		return core.ErrNilEpochStartNotifier
	}
	if check.IfNil(arguments.Marshalizer) {
		return core.ErrNilMarshalizer
	}
	if check.IfNil(arguments.ShardCoordinator) {
		return ErrNilShardCoordinator
	}

	return nil
}

func (di *dataIndexer) epochStartEventHandler() epochStart.ActionHandler {
	subscribeHandler := notifier.NewHandlerForEpochStart(func(hdr nodeData.HeaderHandler) {
		currentEpoch := hdr.GetEpoch()
		validatorsPubKeys, err := di.coordinator.GetAllEligibleValidatorsPublicKeys(currentEpoch)
		if err != nil {
			log.Warn("GetAllEligibleValidatorPublicKeys for current epoch failed",
				"epoch", currentEpoch,
				"error", err.Error())
		}

		go di.SaveValidatorsPubKeys(validatorsPubKeys, currentEpoch)

	}, func(_ nodeData.HeaderHandler) {}, core.IndexerOrder)

	return subscribeHandler
}

// SaveBlock saves the block info in the queue to be sent to elastic
func (di *dataIndexer) SaveBlock(args *indexer.ArgsSaveBlockData) {
	wi := workItems.NewItemBlock(
		di.elasticProcessor,
		di.marshalizer,
		args,
	)
	di.dispatcher.Add(wi)
}

// Close will stop goroutine that index data in database
func (di *dataIndexer) Close() error {
	return di.dispatcher.Close()
}

// RevertIndexedBlock will remove from database block and miniblocks
func (di *dataIndexer) RevertIndexedBlock(header nodeData.HeaderHandler, body nodeData.BodyHandler) {
	wi := workItems.NewItemRemoveBlock(
		di.elasticProcessor,
		body,
		header,
	)
	di.dispatcher.Add(wi)
}

// SaveRoundsInfo will save data about a slice of rounds in elasticsearch
func (di *dataIndexer) SaveRoundsInfo(rf []*indexer.RoundInfo) {
	roundsInfo := make([]*data.RoundInfo, 0)
	for _, info := range rf {
		roundsInfo = append(roundsInfo, &data.RoundInfo{
			Index:            info.Index,
			SignersIndexes:   info.SignersIndexes,
			BlockWasProposed: info.BlockWasProposed,
			ShardId:          info.ShardId,
			Timestamp:        info.Timestamp,
		})
	}

	wi := workItems.NewItemRounds(di.elasticProcessor, roundsInfo)
	di.dispatcher.Add(wi)
}

// SaveValidatorsRating will save all validators rating info to elasticsearch
func (di *dataIndexer) SaveValidatorsRating(indexID string, validatorsRatingInfo []*indexer.ValidatorRatingInfo) {
	valRatingInfo := make([]*data.ValidatorRatingInfo, 0)
	for _, info := range validatorsRatingInfo {
		valRatingInfo = append(valRatingInfo, &data.ValidatorRatingInfo{
			PublicKey: info.PublicKey,
			Rating:    info.Rating,
		})
	}

	wi := workItems.NewItemRating(
		di.elasticProcessor,
		indexID,
		valRatingInfo,
	)
	di.dispatcher.Add(wi)
}

// SaveValidatorsPubKeys will save all validators public keys to elasticsearch
func (di *dataIndexer) SaveValidatorsPubKeys(validatorsPubKeys map[uint32][][]byte, epoch uint32) {
	wi := workItems.NewItemValidators(
		di.elasticProcessor,
		epoch,
		validatorsPubKeys,
	)
	di.dispatcher.Add(wi)
}

// UpdateTPS updates the tps and statistics into elasticsearch index
func (di *dataIndexer) UpdateTPS(tpsBenchmark statistics.TPSBenchmark) {
	if tpsBenchmark == nil {
		log.Debug("indexer: update tps called, but the tpsBenchmark is nil")
		return
	}

	wi := workItems.NewItemTpsBenchmark(di.elasticProcessor, tpsBenchmark)
	di.dispatcher.Add(wi)
}

// SaveAccounts will save the provided accounts
func (di *dataIndexer) SaveAccounts(timestamp uint64, accounts []state.UserAccountHandler) {
	wi := workItems.NewItemAccounts(di.elasticProcessor, timestamp, accounts)
	di.dispatcher.Add(wi)
}

// SetTxLogsProcessor will set tx logs processor
func (di *dataIndexer) SetTxLogsProcessor(txLogsProc process.TransactionLogProcessorDatabase) {
	di.elasticProcessor.SetTxLogsProcessor(txLogsProc)
}

// IsNilIndexer will return a bool value that signals if the indexer's implementation is a NilIndexer
func (di *dataIndexer) IsNilIndexer() bool {
	return di.isNilIndexer
}

// IsInterfaceNil returns true if there is no value under the interface
func (di *dataIndexer) IsInterfaceNil() bool {
	return di == nil
}
