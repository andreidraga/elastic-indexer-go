module github.com/ElrondNetwork/elastic-indexer-go

go 1.15

require (
	github.com/ElrondNetwork/elrond-go v1.2.2-0.20210614184354-99ee4465ab37
	github.com/ElrondNetwork/elrond-go-logger v1.0.4
	github.com/elastic/go-elasticsearch/v7 v7.12.0
	github.com/stretchr/testify v1.7.0
)

replace github.com/ElrondNetwork/arwen-wasm-vm/v1_3 v1.3.16 => github.com/ElrondNetwork/arwen-wasm-vm v1.3.16
