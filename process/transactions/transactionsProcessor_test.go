package transactions

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elastic-indexer-go/data"
	"github.com/ElrondNetwork/elastic-indexer-go/disabled"
	"github.com/ElrondNetwork/elastic-indexer-go/mock"
	"github.com/ElrondNetwork/elrond-go/core"
	nodeData "github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/receipt"
	"github.com/ElrondNetwork/elrond-go/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/testscommon/economicsmocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddToAlteredAddresses(t *testing.T) {
	t.Parallel()

	sender := "senderAddress"
	receiver := "receiverAddress"
	tokenIdentifier := "Test-token"
	tx := &types.Transaction{
		Sender:              sender,
		Receiver:            receiver,
		EsdtValue:           "123",
		EsdtTokenIdentifier: tokenIdentifier,
	}
	alteredAddress := make(map[string]*types.AlteredAccount)
	selfShardID := uint32(0)
	mb := &block.MiniBlock{}

	addToAlteredAddresses(tx, alteredAddress, mb, selfShardID, false)

	alterdAccount, ok := alteredAddress[receiver]
	require.True(t, ok)
	require.Equal(t, &types.AlteredAccount{
		IsESDTOperation: true,
		TokenIdentifier: tokenIdentifier,
	}, alterdAccount)

	alterdAccount, ok = alteredAddress[sender]
	require.True(t, ok)
	require.Equal(t, &types.AlteredAccount{
		IsSender:        true,
		IsESDTOperation: true,
		TokenIdentifier: tokenIdentifier,
	}, alterdAccount)
}

func TestIsSCRForSenderWithGasUsed(t *testing.T) {
	t.Parallel()

	txHash := "txHash"
	nonce := uint64(10)
	sender := "sender"

	tx := &types.Transaction{
		Hash:   txHash,
		Nonce:  nonce,
		Sender: sender,
	}
	sc := &types.ScResult{
		Data:      []byte("@6f6b@something"),
		Nonce:     nonce + 1,
		Receiver:  sender,
		PreTxHash: txHash,
	}

	require.True(t, isSCRForSenderWithRefund(sc, tx))
}

func TestPrepareTransactionsForDatabase(t *testing.T) {
	t.Parallel()

	txHash1 := []byte("txHash1")
	tx1 := &transaction.Transaction{
		GasLimit: 100,
		GasPrice: 100,
	}
	txHash2 := []byte("txHash2")
	tx2 := &transaction.Transaction{
		GasLimit: 100,
		GasPrice: 100,
	}
	txHash3 := []byte("txHash3")
	tx3 := &transaction.Transaction{}
	txHash4 := []byte("txHash4")
	tx4 := &transaction.Transaction{}
	txHash5 := []byte("txHash5")
	tx5 := &transaction.Transaction{}

	rTx1Hash := []byte("rTxHash1")
	rTx1 := &rewardTx.RewardTx{}
	rTx2Hash := []byte("rTxHash2")
	rTx2 := &rewardTx.RewardTx{}

	recHash1 := []byte("recHash1")
	rec1 := &receipt.Receipt{
		Value:  big.NewInt(100),
		TxHash: txHash1,
	}
	recHash2 := []byte("recHash2")
	rec2 := &receipt.Receipt{
		Value:  big.NewInt(200),
		TxHash: txHash2,
	}

	scHash1 := []byte("scHash1")
	scResult1 := &smartContractResult.SmartContractResult{
		OriginalTxHash: txHash1,
		PrevTxHash:     txHash1,
		GasLimit:       1,
	}
	scHash2 := []byte("scHash2")
	scResult2 := &smartContractResult.SmartContractResult{
		OriginalTxHash: txHash1,
		PrevTxHash:     txHash1,
		GasLimit:       1,
	}
	scHash3 := []byte("scHash3")
	scResult3 := &smartContractResult.SmartContractResult{
		OriginalTxHash: txHash3,
		Data:           []byte("@" + "6F6B"),
	}

	body := &block.Body{
		MiniBlocks: []*block.MiniBlock{
			{
				TxHashes: [][]byte{txHash1, txHash2, txHash3},
				Type:     block.TxBlock,
			},
			{
				TxHashes: [][]byte{txHash4},
				Type:     block.TxBlock,
			},
			{
				TxHashes: [][]byte{scHash1, scHash2},
				Type:     block.SmartContractResultBlock,
			},
			{
				TxHashes: [][]byte{scHash3},
				Type:     block.SmartContractResultBlock,
			},
			{
				TxHashes: [][]byte{recHash1, recHash2},
				Type:     block.ReceiptBlock,
			},
			{
				TxHashes: [][]byte{rTx1Hash, rTx2Hash},
				Type:     block.RewardsBlock,
			},
			{
				TxHashes: [][]byte{txHash5},
				Type:     block.InvalidBlock,
			},
		},
	}
	header := &block.Header{}

	pool := &types.Pool{
		Txs: map[string]data.TransactionHandler{
			string(txHash1): tx1,
			string(txHash2): tx2,
			string(txHash3): tx3,
			string(txHash4): tx4,
		},
		Scrs: map[string]data.TransactionHandler{
			string(scHash1): scResult1,
			string(scHash2): scResult2,
			string(scHash3): scResult3,
		},
		Rewards: map[string]data.TransactionHandler{
			string(rTx1Hash): rTx1,
			string(rTx2Hash): rTx2,
		},
		Invalid: map[string]data.TransactionHandler{
			string(txHash5): tx5,
		},
		Receipts: map[string]data.TransactionHandler{
			string(recHash1): rec1,
			string(recHash2): rec2,
		},
	}

	txDbProc := NewTransactionsProcessor(
		&mock.PubkeyConverterMock{},
		&economicsmocks.EconomicsHandlerStub{},
		false,
		&mock.ShardCoordinatorMock{},
		false,
		disabled.NewNilTxLogsProcessor(),
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
	)

	results := txDbProc.PrepareTransactionsForDatabase(body, header, pool)
	assert.Equal(t, 7, len(results.Transactions))

}

func TestPrepareTxLog(t *testing.T) {
	t.Parallel()

	txDbProc := NewTransactionsProcessor(
		&mock.PubkeyConverterMock{},
		&economicsmocks.EconomicsHandlerStub{},
		false,
		&mock.ShardCoordinatorMock{},
		false,
		disabled.NewNilTxLogsProcessor(),
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
	)

	scAddr := []byte("addr")
	addr := []byte("addr")
	identifier := []byte("id")
	top1, top2 := []byte("t1"), []byte("t2")
	dt := []byte("dt")
	txLog := &transaction.Log{
		Address: scAddr,
		Events: []*transaction.Event{
			{
				Address:    addr,
				Identifier: identifier,
				Topics:     [][]byte{top1, top2},
				Data:       dt,
			},
		},
	}
	expectedTxLog := data.TxLog{
		Address: txDbProc.addressPubkeyConverter.Encode(scAddr),
		Events: []data.Event{
			{
				Address:    hex.EncodeToString(addr),
				Identifier: hex.EncodeToString(identifier),
				Topics:     []string{hex.EncodeToString(top1), hex.EncodeToString(top2)},
				Data:       hex.EncodeToString(dt),
			},
		},
	}

	dbTxLog := txDbProc.prepareTxLog(txLog)
	assert.Equal(t, expectedTxLog, dbTxLog)
}

func TestRelayedTransactions(t *testing.T) {
	t.Parallel()

	txHash1 := []byte("txHash1")
	tx1 := &transaction.Transaction{
		GasLimit: 100,
		GasPrice: 100,
		Data:     []byte("relayedTx@blablabllablalba"),
	}

	scHash1 := []byte("scHash1")
	scResult1 := &smartContractResult.SmartContractResult{
		OriginalTxHash: txHash1,
		PrevTxHash:     txHash1,
		GasLimit:       1,
	}
	scHash2 := []byte("scHash2")
	scResult2 := &smartContractResult.SmartContractResult{
		OriginalTxHash: txHash1,
		PrevTxHash:     txHash1,
		GasLimit:       1,
	}
	scHash3 := []byte("scHash3")
	scResult3 := &smartContractResult.SmartContractResult{
		OriginalTxHash: scHash1,
		Data:           []byte("@" + "6F6B"),
	}

	body := &block.Body{
		MiniBlocks: []*block.MiniBlock{
			{
				TxHashes: [][]byte{txHash1},
				Type:     block.TxBlock,
			},
			{
				TxHashes: [][]byte{scHash1, scHash2, scHash3},
				Type:     block.SmartContractResultBlock,
			},
		},
	}

	header := &block.Header{}

	pool := &indexer.Pool{
		Txs: map[string]nodeData.TransactionHandler{
			string(txHash1): tx1,
		},
		Scrs: map[string]nodeData.TransactionHandler{
			string(scHash1): scResult1,
			string(scHash2): scResult2,
			string(scHash3): scResult3,
		},
	}

	txDbProc := NewTransactionsProcessor(
		&mock.PubkeyConverterMock{},
		&economicsmocks.EconomicsHandlerStub{},
		false,
		&mock.ShardCoordinatorMock{},
		false,
		disabled.NewNilTxLogsProcessor(),
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
	)

	results := txDbProc.PrepareTransactionsForDatabase(body, header, pool)
	assert.Equal(t, 1, len(results.Transactions))
	assert.Equal(t, 3, len(results.Transactions[0].SmartContractResults))
	assert.Equal(t, transaction.TxStatusSuccess.String(), results.Transactions[0].Status)
}

func TestSetTransactionSearchOrder(t *testing.T) {
	t.Parallel()
	txHash1 := []byte("txHash1")
	tx1 := &data.Transaction{}

	txHash2 := []byte("txHash2")
	tx2 := &data.Transaction{}

	txPool := map[string]*data.Transaction{
		string(txHash1): tx1,
		string(txHash2): tx2,
	}

	txDbProc := NewTransactionsProcessor(
		&mock.PubkeyConverterMock{},
		&economicsmocks.EconomicsHandlerStub{},
		false,
		&mock.ShardCoordinatorMock{},
		false,
		disabled.NewNilTxLogsProcessor(),
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
	)

	transactions := txDbProc.setTransactionSearchOrder(txPool)
	assert.True(t, txPoolHasSearchOrder(transactions, 0))
	assert.True(t, txPoolHasSearchOrder(transactions, 1))

	transactions = txDbProc.setTransactionSearchOrder(txPool)
	assert.True(t, txPoolHasSearchOrder(transactions, 0))
	assert.True(t, txPoolHasSearchOrder(transactions, 1))

	transactions = txDbProc.setTransactionSearchOrder(txPool)
	assert.True(t, txPoolHasSearchOrder(transactions, 0))
	assert.True(t, txPoolHasSearchOrder(transactions, 1))
}

func TestAlteredAddresses(t *testing.T) {
	expectedAlteredAccounts := make(map[string]struct{})
	// addresses marked with a comment should be added to the altered addresses map

	// normal txs
	address1 := []byte("address1") // should be added
	address2 := []byte("address2")
	expectedAlteredAccounts[hex.EncodeToString(address1)] = struct{}{}
	tx1 := &transaction.Transaction{
		SndAddr: address1,
		RcvAddr: address2,
	}
	tx1Hash := []byte("tx1Hash")

	address3 := []byte("address3")
	address4 := []byte("address4") // should be added
	expectedAlteredAccounts[hex.EncodeToString(address4)] = struct{}{}
	tx2 := &transaction.Transaction{
		SndAddr: address3,
		RcvAddr: address4,
	}
	tx2Hash := []byte("tx2hash")

	txMiniBlock1 := &block.MiniBlock{
		Type:            block.TxBlock,
		TxHashes:        [][]byte{tx1Hash},
		SenderShardID:   0,
		ReceiverShardID: 1,
	}
	txMiniBlock2 := &block.MiniBlock{
		Type:            block.TxBlock,
		TxHashes:        [][]byte{tx2Hash},
		SenderShardID:   1,
		ReceiverShardID: 0,
	}

	// reward txs
	address5 := []byte("address5") // should be added
	expectedAlteredAccounts[hex.EncodeToString(address5)] = struct{}{}
	rwdTx1 := &rewardTx.RewardTx{
		RcvAddr: address5,
	}
	rwdTx1Hash := []byte("rwdTx1")

	address6 := []byte("address6")
	rwdTx2 := &rewardTx.RewardTx{
		RcvAddr: address6,
	}
	rwdTx2Hash := []byte("rwdTx2")

	rewTxMiniBlock1 := &block.MiniBlock{
		Type:            block.RewardsBlock,
		TxHashes:        [][]byte{rwdTx1Hash},
		SenderShardID:   core.MetachainShardId,
		ReceiverShardID: 0,
	}
	rewTxMiniBlock2 := &block.MiniBlock{
		Type:            block.RewardsBlock,
		TxHashes:        [][]byte{rwdTx2Hash},
		SenderShardID:   core.MetachainShardId,
		ReceiverShardID: 1,
	}

	// smart contract results
	address7 := []byte("address7") // should be added
	address8 := []byte("address8")
	expectedAlteredAccounts[hex.EncodeToString(address7)] = struct{}{}
	scr1 := &smartContractResult.SmartContractResult{
		RcvAddr: address7,
		SndAddr: address8,
	}
	scr1Hash := []byte("scr1Hash")

	address9 := []byte("address9") // should be added
	address10 := []byte("address10")
	expectedAlteredAccounts[hex.EncodeToString(address9)] = struct{}{}
	scr2 := &smartContractResult.SmartContractResult{
		RcvAddr: address9,
		SndAddr: address10,
	}
	scr2Hash := []byte("scr2Hash")

	scrMiniBlock1 := &block.MiniBlock{
		Type:            block.SmartContractResultBlock,
		TxHashes:        [][]byte{scr1Hash, scr2Hash},
		SenderShardID:   1,
		ReceiverShardID: 0,
	}

	body := &block.Body{
		MiniBlocks: []*block.MiniBlock{txMiniBlock1, txMiniBlock2, rewTxMiniBlock1, rewTxMiniBlock2, scrMiniBlock1},
	}

	hdr := &block.Header{}

	pool := &types.Pool{
		Txs: map[string]nodeData.TransactionHandler{
			string(tx1Hash): tx1,
			string(tx2Hash): tx2,
		},
		Scrs: map[string]nodeData.TransactionHandler{
			string(scr1Hash): scr1,
			string(scr2Hash): scr2,
		},
		Rewards: map[string]datnodeDataa.TransactionHandler{
			string(rwdTx1Hash): rwdTx1,
			string(rwdTx2Hash): rwdTx2,
		},
	}

	shardCoordinator := &mock.ShardCoordinatorMock{
		ComputeIdCalled: func(address []byte) uint32 {
			switch string(address) {
			case string(address1), string(address4), string(address5), string(address7), string(address9):
				return 0
			default:
				return 1
			}
		},
	}

	txDbProc := NewTransactionsProcessor(
		mock.NewPubkeyConverterMock(32),
		&economicsmocks.EconomicsHandlerStub{},
		false,
		shardCoordinator,
		false,
		disabled.NewNilTxLogsProcessor(),
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
	)

	results := txDbProc.PrepareTransactionsForDatabase(body, hdr, pool)
	require.Equal(t, len(expectedAlteredAccounts), len(results.AlteredAccounts))

	for addrActual := range results.AlteredAccounts {
		_, found := expectedAlteredAccounts[addrActual]
		if !found {
			assert.Fail(t, fmt.Sprintf("address %s not found", addrActual))
		}
	}
}

func txPoolHasSearchOrder(txPool map[string]*data.Transaction, searchOrder uint32) bool {
	for _, tx := range txPool {
		if tx.SearchOrder == searchOrder {
			return true
		}
	}

	return false
}

func TestCheckGasUsedTooMuchGasProvidedCase(t *testing.T) {
	t.Parallel()

	txHash := "txHash"
	nonce := uint64(10)
	sender := "sender"

	tx := &data.Transaction{
		Hash:   txHash,
		Nonce:  nonce,
		Sender: sender,
	}
	sc := &data.ScResult{
		Data:      []byte("@6f6b@something"),
		Nonce:     nonce + 1,
		Receiver:  sender,
		PreTxHash: txHash,
	}

	require.True(t, isSCRForSenderWithRefund(sc, tx))
}

func TestCheckGasUsedInvalidTransaction(t *testing.T) {
	t.Parallel()

	txDbProc := NewTransactionsProcessor(
		mock.NewPubkeyConverterMock(32),
		&economicsmocks.EconomicsHandlerStub{},
		false,
		&mock.ShardCoordinatorMock{},
		false,
		disabled.NewNilTxLogsProcessor(),
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
	)

	txHash1 := []byte("txHash1")
	tx1 := &transaction.Transaction{
		GasLimit: 100,
		GasPrice: 100,
	}
	recHash1 := []byte("recHash1")
	rec1 := &receipt.Receipt{
		Value:  big.NewInt(100),
		TxHash: txHash1,
	}

	body := &block.Body{
		MiniBlocks: []*block.MiniBlock{
			{
				TxHashes: [][]byte{txHash1},
				Type:     block.InvalidBlock,
			},
			{
				TxHashes: [][]byte{recHash1},
				Type:     block.ReceiptBlock,
			},
		},
	}

	header := &block.Header{}

	pool := &indexer.Pool{
		Invalid: map[string]nodeData.TransactionHandler{
			string(txHash1): tx1,
		},
		Receipts: map[string]nodeData.TransactionHandler{
			string(recHash1): rec1,
		},
	}

	results := txDbProc.PrepareTransactionsForDatabase(body, header, pool)
	require.Len(t, results.Transactions, 1)
	require.Equal(t, tx1.GasLimit, results.Transactions[0].GasUsed)
}

func TestCheckGasUsedRelayedTransaction(t *testing.T) {
	t.Parallel()

	txDbProc := NewTransactionsProcessor(
		mock.NewPubkeyConverterMock(32),
		&economicsmocks.EconomicsHandlerStub{},
		false,
		&mock.ShardCoordinatorMock{},
		false,
		disabled.NewNilTxLogsProcessor(),
		&mock.HasherMock{},
		&mock.MarshalizerMock{},
	)

	txHash1 := []byte("txHash1")
	tx1 := &transaction.Transaction{
		GasLimit: 100,
		GasPrice: 123456,
		Data:     []byte("relayedTx@1231231231239129312"),
	}
	scResHash1 := []byte("scResHash1")
	scRes1 := &smartContractResult.SmartContractResult{
		OriginalTxHash: txHash1,
	}

	body := &block.Body{
		MiniBlocks: []*block.MiniBlock{
			{
				TxHashes: [][]byte{txHash1},
				Type:     block.TxBlock,
			},
			{
				TxHashes: [][]byte{scResHash1},
				Type:     block.SmartContractResultBlock,
			},
		},
	}

	header := &block.Header{}

	pool := &indexer.Pool{
		Txs: map[string]nodeData.TransactionHandler{
			string(txHash1): tx1,
		},
		Scrs: map[string]nodeData.TransactionHandler{
			string(scResHash1): scRes1,
		},
	}

	results := txDbProc.PrepareTransactionsForDatabase(body, header, pool)
	require.Len(t, results.Transactions, 1)
	require.Equal(t, tx1.GasLimit, results.Transactions[0].GasUsed)
}
