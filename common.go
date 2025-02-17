package indexer

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ElrondNetwork/elastic-indexer-go/data"
	"github.com/ElrondNetwork/elastic-indexer-go/templates/noKibana"
	"github.com/ElrondNetwork/elastic-indexer-go/templates/withKibana"
	"github.com/ElrondNetwork/elrond-go-core/core"
	coreData "github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/block"
	"github.com/ElrondNetwork/elrond-go-core/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go-core/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go-core/data/transaction"
)

type objectsMap = map[string]interface{}

type commonProcessor struct {
	shardCoordinator         Coordinator
	addressPubkeyConverter   core.PubkeyConverter
	validatorPubkeyConverter core.PubkeyConverter
	txFeeCalculator          FeesProcessorHandler
}

func (cm *commonProcessor) buildTransaction(
	tx *transaction.Transaction,
	txHash []byte,
	mbHash []byte,
	mb *block.MiniBlock,
	header coreData.HeaderHandler,
	txStatus string,
) *data.Transaction {
	gasUsed := cm.txFeeCalculator.ComputeGasLimit(tx)
	fee := cm.txFeeCalculator.ComputeTxFeeBasedOnGasUsed(tx, gasUsed)

	return &data.Transaction{
		Hash:                 hex.EncodeToString(txHash),
		MBHash:               hex.EncodeToString(mbHash),
		Nonce:                tx.Nonce,
		Round:                header.GetRound(),
		Value:                tx.Value.String(),
		Receiver:             cm.addressPubkeyConverter.Encode(tx.RcvAddr),
		Sender:               cm.addressPubkeyConverter.Encode(tx.SndAddr),
		ReceiverShard:        cm.shardCoordinator.ComputeId(tx.RcvAddr),
		SenderShard:          mb.SenderShardID,
		GasPrice:             tx.GasPrice,
		GasLimit:             tx.GasLimit,
		Data:                 tx.Data,
		Signature:            hex.EncodeToString(tx.Signature),
		Timestamp:            time.Duration(header.GetTimeStamp()),
		Status:               txStatus,
		GasUsed:              gasUsed,
		Fee:                  fee.String(),
		ReceiverUserName:     tx.RcvUserName,
		SenderUserName:       tx.SndUserName,
		ReceiverAddressBytes: tx.RcvAddr,
	}
}

func (cm *commonProcessor) buildRewardTransaction(
	rTx *rewardTx.RewardTx,
	txHash []byte,
	mbHash []byte,
	mb *block.MiniBlock,
	header coreData.HeaderHandler,
	txStatus string,
) *data.Transaction {
	return &data.Transaction{
		Hash:          hex.EncodeToString(txHash),
		MBHash:        hex.EncodeToString(mbHash),
		Nonce:         0,
		Round:         rTx.Round,
		Value:         rTx.Value.String(),
		Receiver:      cm.addressPubkeyConverter.Encode(rTx.RcvAddr),
		Sender:        fmt.Sprintf("%d", core.MetachainShardId),
		ReceiverShard: mb.ReceiverShardID,
		SenderShard:   mb.SenderShardID,
		GasPrice:      0,
		GasLimit:      0,
		Data:          make([]byte, 0),
		Signature:     "",
		Timestamp:     time.Duration(header.GetTimeStamp()),
		Status:        txStatus,
	}
}

func (cm *commonProcessor) convertScResultInDatabaseScr(scHash string, sc *smartContractResult.SmartContractResult) data.ScResult {
	relayerAddr := ""
	if len(sc.RelayerAddr) > 0 {
		relayerAddr = cm.addressPubkeyConverter.Encode(sc.RelayerAddr)
	}

	return data.ScResult{
		Hash:           hex.EncodeToString([]byte(scHash)),
		Nonce:          sc.Nonce,
		GasLimit:       sc.GasLimit,
		GasPrice:       sc.GasPrice,
		Value:          sc.Value.String(),
		Sender:         cm.addressPubkeyConverter.Encode(sc.SndAddr),
		Receiver:       cm.addressPubkeyConverter.Encode(sc.RcvAddr),
		RelayerAddr:    relayerAddr,
		RelayedValue:   sc.RelayedValue.String(),
		Code:           string(sc.Code),
		Data:           sc.Data,
		PreTxHash:      hex.EncodeToString(sc.PrevTxHash),
		OriginalTxHash: hex.EncodeToString(sc.OriginalTxHash),
		CallType:       strconv.Itoa(int(sc.CallType)),
		CodeMetadata:   sc.CodeMetadata,
		ReturnMessage:  string(sc.ReturnMessage),
	}
}

func serializeBulkMiniBlocks(
	hdrShardID uint32,
	bulkMbs []*data.Miniblock,
	getAlreadyIndexedItems func(hashes []string, index string) (map[string]bool, error),
) (bytes.Buffer, map[string]bool) {
	var err error
	var buff bytes.Buffer

	mbsHashes := make([]string, len(bulkMbs))
	for idx := range bulkMbs {
		mbsHashes[idx] = bulkMbs[idx].Hash
	}

	existsInDb, err := getAlreadyIndexedItems(mbsHashes, miniblocksIndex)
	if err != nil {
		log.Warn("indexer get indexed items miniblocks",
			"error", err.Error())
		return buff, make(map[string]bool)
	}

	for _, mb := range bulkMbs {
		var meta, serializedData []byte
		if !existsInDb[mb.Hash] {
			//insert miniblock in database
			meta = []byte(fmt.Sprintf(`{ "index" : { "_id" : "%s", "_type" : "%s" } }%s`, mb.Hash, "_doc", "\n"))
			serializedData, err = json.Marshal(mb)
			if err != nil {
				log.Debug("indexer: marshal",
					"error", "could not serialize miniblock, will skip indexing",
					"mb hash", mb.Hash)
				continue
			}
		} else {
			// update miniblock
			meta = []byte(fmt.Sprintf(`{ "update" : { "_id" : "%s" } }%s`, mb.Hash, "\n"))
			if hdrShardID == mb.SenderShardID {
				// update sender block hash
				serializedData = []byte(fmt.Sprintf(`{ "doc" : { "senderBlockHash" : "%s" } }`, mb.SenderBlockHash))
			} else {
				// update receiver block hash
				serializedData = []byte(fmt.Sprintf(`{ "doc" : { "receiverBlockHash" : "%s" } }`, mb.ReceiverBlockHash))
			}
		}

		buff = prepareBufferMiniblocks(buff, meta, serializedData)
	}

	return buff, existsInDb
}

func prepareBufferMiniblocks(buff bytes.Buffer, meta, serializedData []byte) bytes.Buffer {
	// append a newline for each element
	serializedData = append(serializedData, "\n"...)
	buff.Grow(len(meta) + len(serializedData))
	_, err := buff.Write(meta)
	if err != nil {
		log.Warn("elastic search: serialize bulk miniblocks, write meta", "error", err.Error())
	}
	_, err = buff.Write(serializedData)
	if err != nil {
		log.Warn("elastic search: serialize bulk miniblocks, write serialized miniblock", "error", err.Error())
	}

	return buff
}

func serializeTransactions(
	transactions []*data.Transaction,
	selfShardID uint32,
	_ func(hashes []string, index string) (map[string]bool, error),
	mbsHashInDB map[string]bool,
) ([]bytes.Buffer, error) {
	var err error

	var buff bytes.Buffer
	buffSlice := make([]bytes.Buffer, 0)
	for _, tx := range transactions {
		isMBOfTxInDB := mbsHashInDB[tx.MBHash]
		meta, serializedData, errPrepareTx := prepareSerializedDataForATransaction(tx, selfShardID, isMBOfTxInDB)
		if errPrepareTx != nil {
			log.Warn("error preparing transaction for indexing", "tx hash", tx.Hash, "error", err)
			return nil, errPrepareTx
		}

		// append a newline for each element
		serializedData = append(serializedData, "\n"...)

		buffLenWithCurrentTx := buff.Len() + len(meta) + len(serializedData)
		if buffLenWithCurrentTx > bulkSizeThreshold && buff.Len() != 0 {
			buffSlice = append(buffSlice, buff)
			buff = bytes.Buffer{}
		}

		buff.Grow(len(meta) + len(serializedData))
		_, err = buff.Write(meta)
		if err != nil {
			log.Warn("elastic search: serialize bulk tx, write meta", "error", err.Error())
			return nil, err

		}
		_, err = buff.Write(serializedData)
		if err != nil {
			log.Warn("elastic search: serialize bulk tx, write serialized tx", "error", err.Error())
			return nil, err
		}
	}

	// check if the last buffer contains data
	if buff.Len() != 0 {
		buffSlice = append(buffSlice, buff)
	}

	return buffSlice, nil
}

func serializeAccounts(accounts map[string]*data.AccountInfo) ([]bytes.Buffer, error) {
	var err error

	var buff bytes.Buffer
	buffSlice := make([]bytes.Buffer, 0)
	for address, acc := range accounts {
		meta, serializedData, errPrepareAcc := prepareSerializedAccountInfo(address, acc)
		if len(meta) == 0 {
			log.Warn("cannot prepare serializes account info", "error", errPrepareAcc)
			return nil, err
		}

		// append a newline for each element
		serializedData = append(serializedData, "\n"...)

		buffLenWithCurrentAcc := buff.Len() + len(meta) + len(serializedData)
		if buffLenWithCurrentAcc > bulkSizeThreshold && buff.Len() != 0 {
			buffSlice = append(buffSlice, buff)
			buff = bytes.Buffer{}
		}

		buff.Grow(len(meta) + len(serializedData))
		_, err = buff.Write(meta)
		if err != nil {
			log.Warn("elastic search: serialize bulk accounts, write meta", "error", err.Error())
			return nil, err
		}
		_, err = buff.Write(serializedData)
		if err != nil {
			log.Warn("elastic search: serialize bulk accounts, write serialized account", "error", err.Error())
			return nil, err
		}
	}

	// check if the last buffer contains data
	if buff.Len() != 0 {
		buffSlice = append(buffSlice, buff)
	}

	return buffSlice, nil
}

func serializeAccountsHistory(accounts map[string]*data.AccountBalanceHistory) ([]bytes.Buffer, error) {
	var err error

	var buff bytes.Buffer
	buffSlice := make([]bytes.Buffer, 0)
	for address, acc := range accounts {
		meta, serializedData, errPrepareAcc := prepareSerializedAccountBalanceHistory(address, acc)
		if errPrepareAcc != nil {
			log.Warn("cannot prepare serializes account balance history", "error", err)
			return nil, err
		}

		// append a newline for each element
		serializedData = append(serializedData, "\n"...)

		buffLenWithCurrentAccountHistory := buff.Len() + len(meta) + len(serializedData)
		if buffLenWithCurrentAccountHistory > bulkSizeThreshold && buff.Len() != 0 {
			buffSlice = append(buffSlice, buff)
			buff = bytes.Buffer{}
		}

		buff.Grow(len(meta) + len(serializedData))
		_, err = buff.Write(meta)
		if err != nil {
			log.Warn("elastic search: serialize bulk accounts history, write meta", "error", err.Error())
			return nil, err
		}
		_, err = buff.Write(serializedData)
		if err != nil {
			log.Warn("elastic search: serialize bulk accounts history, write serialized account history", "error", err.Error())
			return nil, err
		}
	}

	// check if the last buffer contains data
	if buff.Len() != 0 {
		buffSlice = append(buffSlice, buff)
	}

	return buffSlice, nil
}

func prepareSerializedDataForATransaction(
	tx *data.Transaction,
	selfShardID uint32,
	_ bool,
) ([]byte, []byte, error) {
	metaData := []byte(fmt.Sprintf(`{"update":{"_id":"%s", "_type": "_doc"}}%s`, tx.Hash, "\n"))

	marshaledTx, err := json.Marshal(tx)
	if err != nil {
		log.Debug("indexer: marshal",
			"error", "could not serialize transaction, will skip indexing",
			"tx hash", tx.Hash)
		return nil, nil, err
	}

	if isIntraShardOrInvalid(tx, selfShardID) {
		// if transaction is intra-shard, use basic insert as data can be re-written at forks
		meta := []byte(fmt.Sprintf(`{ "index" : { "_id" : "%s", "_type" : "%s" } }%s`, tx.Hash, "_doc", "\n"))
		log.Trace("indexer tx is intra shard or invalid tx", "meta", string(meta), "marshaledTx", string(marshaledTx))

		return meta, marshaledTx, nil
	}

	if !isCrossShardDstMe(tx, selfShardID) {
		// if transaction is cross-shard and current shard ID is source, use upsert without updating anything
		serializedData :=
			[]byte(fmt.Sprintf(`{"script":{"source":"return"},"upsert":%s}`,
				string(marshaledTx)))
		log.Trace("indexer tx is on sender shard", "metaData", string(metaData), "serializedData", string(serializedData))

		return metaData, serializedData, nil
	}

	// if transaction is cross-shard and current shard ID is destination, use upsert with updating fields
	marshaledLog, err := json.Marshal(tx.Log)
	if err != nil {
		log.Debug("indexer: marshal",
			"error", "could not serialize transaction log, will skip indexing",
			"tx hash", tx.Hash)
		return nil, nil, err
	}
	scResults, err := json.Marshal(tx.SmartContractResults)
	if err != nil {
		log.Debug("indexer: marshal",
			"error", "could not serialize smart contract results, will skip indexing",
			"tx hash", tx.Hash)
		return nil, nil, err
	}

	marshaledTimestamp, err := json.Marshal(tx.Timestamp)
	if err != nil {
		log.Debug("indexer: marshal",
			"error", "could not serialize timestamp, will skip indexing",
			"tx hash", tx.Hash)
		return nil, nil, err
	}

	serializedData := []byte(fmt.Sprintf(`{"script":{"source":"`+
		`ctx._source.status = params.status;`+
		`ctx._source.miniBlockHash = params.miniBlockHash;`+
		`ctx._source.log = params.log;`+
		`ctx._source.scResults = params.scResults;`+
		`ctx._source.timestamp = params.timestamp;`+
		`ctx._source.gasUsed = params.gasUsed;`+
		`ctx._source.fee = params.fee;`+
		`","lang": "painless","params":`+
		`{"status": "%s", "miniBlockHash": "%s", "log": %s, "scResults": %s, "timestamp": %s, "gasUsed": %d, "fee": "%s"}},"upsert":%s}`,
		tx.Status, tx.MBHash, string(marshaledLog), string(scResults), string(marshaledTimestamp), tx.GasUsed, tx.Fee, string(marshaledTx)))

	log.Trace("indexer tx is on destination shard", "metaData", string(metaData), "serializedData", string(serializedData))

	return metaData, serializedData, nil
}

func isRelayedTx(tx *data.Transaction) bool {
	return strings.HasPrefix(string(tx.Data), "relayedTx") && len(tx.SmartContractResults) > 0
}

func isESDTNFTTransfer(tx *data.Transaction) bool {
	return strings.HasPrefix(string(tx.Data), core.BuiltInFunctionESDTNFTTransfer) && len(tx.SmartContractResults) > 0
}

func prepareSerializedAccountInfo(address string, account *data.AccountInfo) ([]byte, []byte, error) {
	meta := []byte(fmt.Sprintf(`{ "index" : { "_id" : "%s" } }%s`, address, "\n"))
	serializedData, err := json.Marshal(account)
	if err != nil {
		log.Debug("indexer: marshal",
			"error", "could not serialize account, will skip indexing",
			"address", address)
		return nil, nil, err
	}

	return meta, serializedData, nil
}

func prepareSerializedAccountBalanceHistory(address string, account *data.AccountBalanceHistory) ([]byte, []byte, error) {
	meta := []byte(fmt.Sprintf(`{ "index" : { "_id" : "%s" } }%s`, address, "\n"))
	serializedData, err := json.Marshal(account)
	if err != nil {
		log.Debug("indexer: marshal",
			"error", "could not serialize account history entry, will skip indexing",
			"address", address)
		return nil, nil, err
	}

	return meta, serializedData, nil
}

func isCrossShardDstMe(tx *data.Transaction, selfShardID uint32) bool {
	return tx.SenderShard != tx.ReceiverShard && tx.ReceiverShard == selfShardID
}

func isIntraShardOrInvalid(tx *data.Transaction, selfShardID uint32) bool {
	return (tx.SenderShard == tx.ReceiverShard && tx.ReceiverShard == selfShardID) || tx.Status == transaction.TxStatusInvalid.String()
}

func getDecodedResponseMultiGet(response objectsMap) map[string]bool {
	founded := make(map[string]bool)
	interfaceSlice, ok := response["docs"].([]interface{})
	if !ok {
		return founded
	}

	for _, element := range interfaceSlice {
		obj := element.(objectsMap)
		_, ok = obj["error"]
		if ok {
			continue
		}
		founded[obj["_id"].(string)] = obj["found"].(bool)
	}

	return founded
}

// GetElasticTemplatesAndPolicies will return elastic templates and policies
func GetElasticTemplatesAndPolicies(useKibana bool) (map[string]*bytes.Buffer, map[string]*bytes.Buffer, error) {
	indexTemplates := make(map[string]*bytes.Buffer)
	indexPolicies := make(map[string]*bytes.Buffer)

	if useKibana {
		indexTemplates = getTemplatesKibana()
		indexPolicies = getPolicies()

		return indexTemplates, indexPolicies, nil
	}

	indexTemplates = getTemplatesNoKibana()

	return indexTemplates, indexPolicies, nil
}

func getTemplatesKibana() map[string]*bytes.Buffer {
	indexTemplates := make(map[string]*bytes.Buffer)

	indexTemplates["opendistro"] = withKibana.OpenDistro.ToBuffer()
	indexTemplates[txIndex] = withKibana.Transactions.ToBuffer()
	indexTemplates[blockIndex] = withKibana.Blocks.ToBuffer()
	indexTemplates[miniblocksIndex] = withKibana.Miniblocks.ToBuffer()
	indexTemplates[ratingIndex] = withKibana.Rating.ToBuffer()
	indexTemplates[roundIndex] = withKibana.Rounds.ToBuffer()
	indexTemplates[validatorsIndex] = withKibana.Validators.ToBuffer()
	indexTemplates[accountsIndex] = withKibana.Accounts.ToBuffer()
	indexTemplates[accountsHistoryIndex] = withKibana.AccountsHistory.ToBuffer()

	return indexTemplates
}

func getTemplatesNoKibana() map[string]*bytes.Buffer {
	indexTemplates := make(map[string]*bytes.Buffer)

	indexTemplates["opendistro"] = noKibana.OpenDistro.ToBuffer()
	indexTemplates[txIndex] = noKibana.Transactions.ToBuffer()
	indexTemplates[blockIndex] = noKibana.Blocks.ToBuffer()
	indexTemplates[miniblocksIndex] = noKibana.Miniblocks.ToBuffer()
	indexTemplates[ratingIndex] = noKibana.Rating.ToBuffer()
	indexTemplates[roundIndex] = noKibana.Rounds.ToBuffer()
	indexTemplates[validatorsIndex] = noKibana.Validators.ToBuffer()
	indexTemplates[accountsIndex] = noKibana.Accounts.ToBuffer()
	indexTemplates[accountsHistoryIndex] = noKibana.AccountsHistory.ToBuffer()

	return indexTemplates
}

func getPolicies() map[string]*bytes.Buffer {
	indexesPolicies := make(map[string]*bytes.Buffer)

	indexesPolicies[txPolicy] = withKibana.TransactionsPolicy.ToBuffer()
	indexesPolicies[blockPolicy] = withKibana.BlocksPolicy.ToBuffer()
	indexesPolicies[miniblocksPolicy] = withKibana.MiniblocksPolicy.ToBuffer()
	indexesPolicies[ratingPolicy] = withKibana.RatingPolicy.ToBuffer()
	indexesPolicies[roundPolicy] = withKibana.RoundsPolicy.ToBuffer()
	indexesPolicies[validatorsPolicy] = withKibana.ValidatorsPolicy.ToBuffer()
	indexesPolicies[accountsHistoryPolicy] = withKibana.AccountsHistoryPolicy.ToBuffer()

	return indexesPolicies
}

func stringValueToBigInt(strValue string) *big.Int {
	value, ok := big.NewInt(0).SetString(strValue, 10)
	if !ok {
		return big.NewInt(0)
	}

	return value
}
