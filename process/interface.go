package process

import (
	"bytes"

	"github.com/ElrondNetwork/elastic-indexer-go/data"
	"github.com/ElrondNetwork/elrond-go/core/statistics"
	nodeData "github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/indexer"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/elastic/go-elasticsearch/v7/esapi"
)

// DatabaseClientHandler defines the actions that a component that handles requests should do
type DatabaseClientHandler interface {
	DoRequest(req *esapi.IndexRequest) error
	DoBulkRequest(buff *bytes.Buffer, index string) error
	DoBulkRemove(index string, hashes []string) error
	DoMultiGet(hashes []string, index string) (objectsMap, error)

	CheckAndCreateIndex(index string) error
	CheckAndCreateAlias(alias string, index string) error
	CheckAndCreateTemplate(templateName string, template *bytes.Buffer) error
	CheckAndCreatePolicy(policyName string, policy *bytes.Buffer) error

	IsInterfaceNil() bool
}

// DBAccountHandler defines the actions that an accounts handler should do
type DBAccountHandler interface {
	GetAccounts(alteredAccounts data.AlteredAccountsHandler) ([]*data.Account, []*data.AccountESDT)
	PrepareRegularAccountsMap(accounts []*data.Account) map[string]*data.AccountInfo
	PrepareAccountsMapESDT(accounts []*data.AccountESDT, timestamp uint64) (map[string]*data.AccountInfo, []*data.TokenInfo)
	PrepareAccountsHistory(timestamp uint64, accounts map[string]*data.AccountInfo) map[string]*data.AccountBalanceHistory

	SerializeAccountsHistory(accounts map[string]*data.AccountBalanceHistory) ([]*bytes.Buffer, error)
	SerializeAccounts(accounts map[string]*data.AccountInfo, areESDTAccounts bool) ([]*bytes.Buffer, error)
	SerializeNFTCreateInfo(tokensInfo []*data.TokenInfo) ([]*bytes.Buffer, error)
}

// DBBlockHandler defines the actions that a block handler should do
type DBBlockHandler interface {
	PrepareBlockForDB(header nodeData.HeaderHandler, signersIndexes []uint64, body *block.Body, notarizedHeadersHashes []string, sizeTxs int) (*data.Block, error)
	ComputeHeaderHash(header nodeData.HeaderHandler) ([]byte, error)

	SerializeEpochInfoData(header nodeData.HeaderHandler) (*bytes.Buffer, error)
	SerializeBlock(elasticBlock *data.Block) (*bytes.Buffer, error)
}

// DBTransactionsHandler defines the actions that a transactions handler should do
type DBTransactionsHandler interface {
	PrepareTransactionsForDatabase(
		body *block.Body,
		header nodeData.HeaderHandler,
		pool *indexer.Pool,
	) *data.PreparedResults
	GetRewardsTxsHashesHexEncoded(header nodeData.HeaderHandler, body *block.Body) []string
	SetTxLogProcessor(logProcessor process.TransactionLogProcessorDatabase)

	SerializeReceipts(receipts []*data.Receipt) ([]*bytes.Buffer, error)
	SerializeTransactions(transactions []*data.Transaction, selfShardID uint32, mbsHashInDB map[string]bool) ([]*bytes.Buffer, error)
	SerializeScResults(scResults []*data.ScResult) ([]*bytes.Buffer, error)
	SerializeDeploysData(deploys []*data.ScDeployInfo) ([]*bytes.Buffer, error)
	SerializeTokens(tokens []*data.TokenInfo) ([]*bytes.Buffer, error)
}

// DBMiniblocksHandler defines the actions that a miniblocks handler should do
type DBMiniblocksHandler interface {
	PrepareDBMiniblocks(header nodeData.HeaderHandler, body *block.Body) []*data.Miniblock
	GetMiniblocksHashesHexEncoded(header nodeData.HeaderHandler, body *block.Body) []string

	SerializeBulkMiniBlocks(bulkMbs []*data.Miniblock, mbsInDB map[string]bool) *bytes.Buffer
}

// DBStatisticsHandler defines the actions that a database statistics handler should do
type DBStatisticsHandler interface {
	PrepareStatistics(tpsBenchmark statistics.TPSBenchmark) (*data.TPS, []*data.TPS, error)

	SerializeStatistics(genInfo *data.TPS, shardsInfo []*data.TPS, index string) (*bytes.Buffer, error)
	SerializeRoundsInfo(roundsInfo []*data.RoundInfo) *bytes.Buffer
}

// DBValidatorsHandler defines the actions that a validators handler should do
type DBValidatorsHandler interface {
	PrepareValidatorsPublicKeys(shardValidatorsPubKeys [][]byte) *data.ValidatorsPublicKeys
	SerializeValidatorsPubKeys(validatorsPubKeys *data.ValidatorsPublicKeys) (*bytes.Buffer, error)
	SerializeValidatorsRating(index string, validatorsRatingInfo []*data.ValidatorRatingInfo) ([]*bytes.Buffer, error)
}
