package allowance

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	// "github.com/notegio/openrelay/funds"
	"log"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/notegio/openrelay/channels"
	"github.com/notegio/openrelay/db"
	"github.com/notegio/openrelay/exchangecontract"
	"github.com/notegio/openrelay/monitor/blocks"
	"github.com/notegio/openrelay/types"
)

type allowanceBlockConsumer struct {
	bitDexProxyAddress *big.Int
	tokenProxyAddress  *big.Int
	approvalTopic      *big.Int
	feeTokenAddress    string // Needed for the SpendRecord,
	logFilter          ethereum.LogFilterer
	publisher          channels.Publisher
}

func (consumer *allowanceBlockConsumer) Consume(delivery channels.Delivery) {
	block := &blocks.MiniBlock{}
	err := json.Unmarshal([]byte(delivery.Payload()), block)
	if err != nil {
		log.Printf("Error parsing payload: %v\n", err.Error())
	}
	if coreTypes.BloomLookup(block.Bloom, consumer.approvalTopic) && coreTypes.BloomLookup(block.Bloom, common.BigToHash(consumer.tokenProxyAddress)) {
		log.Printf("Block %#x bloom filter indicates approval event for %#x", block.Hash, consumer.tokenProxyAddress)
		query := ethereum.FilterQuery{
			FromBlock: block.Number,
			ToBlock:   block.Number,
			Addresses: nil,
			Topics: [][]common.Hash{
				[]common.Hash{common.BigToHash(consumer.approvalTopic)},
				nil,
				[]common.Hash{common.BigToHash(consumer.tokenProxyAddress)},
			},
		}
		logs, err := consumer.logFilter.FilterLogs(context.Background(), query)
		if err != nil {
			delivery.Return()
			log.Fatalf("Failed to filter logs on block %v - aborting: %v", block.Number, err.Error())
		}
		log.Printf("Found %v approval logs", len(logs))
		for _, approvalLog := range logs {
			if len(approvalLog.Topics) < 2 || len(approvalLog.Data) != 32 {
				log.Printf("Unexpected log data. Skipping.")
				continue
			}
			balance := big.NewInt(0)
			balance.SetBytes(approvalLog.Data)
			sr := &db.SpendRecord{
				TokenAddress:   strings.ToLower(approvalLog.Address.String()),
				SpenderAddress: hexutil.Encode(approvalLog.Topics[1][12:]),
				ZrxToken:       consumer.feeTokenAddress,
				Balance:        balance.String(),
			}
			msg, err := json.Marshal(sr)
			if err != nil {
				delivery.Return()
				log.Fatalf("Failed to encode SpendRecord on block %v", block.Number)
			}
			consumer.publisher.Publish(string(msg))
		}
	} else if coreTypes.BloomLookup(block.Bloom, consumer.approvalTopic) && coreTypes.BloomLookup(block.Bloom, common.BigToHash(consumer.bitDexProxyAddress)) {
		log.Printf("Block %#x bloom filter indicates approval event for %#x", block.Hash, consumer.bitDexProxyAddress)
		query := ethereum.FilterQuery{
			FromBlock: block.Number,
			ToBlock:   block.Number,
			Addresses: nil,
			Topics: [][]common.Hash{
				[]common.Hash{common.BigToHash(consumer.approvalTopic)},
				nil,
				[]common.Hash{common.BigToHash(consumer.bitDexProxyAddress)},
			},
		}
		logs, err := consumer.logFilter.FilterLogs(context.Background(), query)
		if err != nil {
			delivery.Return()
			log.Fatalf("Failed to filter logs on block %v - aborting: %v", block.Number, err.Error())
		}
		log.Printf("Found %v approval logs", len(logs))
		for _, approvalLog := range logs {
			if len(approvalLog.Topics) < 2 || len(approvalLog.Data) != 32 {
				log.Printf("Unexpected log data. Skipping.")
				continue
			}
			balance := big.NewInt(0)
			balance.SetBytes(approvalLog.Data)
			sr := &db.SpendRecord{
				TokenAddress:   strings.ToLower(approvalLog.Address.String()),
				SpenderAddress: hexutil.Encode(approvalLog.Topics[1][12:]),
				ZrxToken:       consumer.feeTokenAddress,
				Balance:        balance.String(),
			}
			msg, err := json.Marshal(sr)
			if err != nil {
				delivery.Return()
				log.Fatalf("Failed to encode SpendRecord on block %v", block.Number)
			}
			consumer.publisher.Publish(string(msg))
		}
	} else {
		log.Printf("Block 0x%x shows no approval events", block.Hash)
	}
	delivery.Ack()
}

func NewAllowanceBlockConsumer(bdp *big.Int, tp *big.Int, feeToken string, lf ethereum.LogFilterer, publisher channels.Publisher) channels.Consumer {
	approvalTopic := &big.Int{}
	approvalTopic.SetString("8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925", 16)
	return &allowanceBlockConsumer{bdp, tp, approvalTopic, feeToken, lf, publisher}
}

func NewRPCAllowanceBlockConsumer(rpcURL string, exchangeAddress string, publisher channels.Publisher) (channels.Consumer, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}
	exchange, err := exchangecontract.NewExchange(common.HexToAddress(exchangeAddress), client)
	if err != nil {
		log.Printf("Error intializing exchange contract '%v': '%v'", exchangeAddress, err.Error())
		return nil, err
	}
	feeTokenAssetData, err := exchange.ZRX_ASSET_DATA(nil)
	if err != nil {
		log.Printf("Error getting fee token address for exchange %v", exchangeAddress)
		return nil, err
	}
	feeTokenAsset := make(types.AssetData, len(feeTokenAssetData))
	copy(feeTokenAsset[:], feeTokenAssetData[:])
	feeTokenAddress := feeTokenAsset.Address()
	bitDexProxyAddress, err := exchange.GetAssetProxy(nil, types.BitDexProxyID)
	if err != nil {
		log.Printf("error getting bitDexProxyAddress")
		return nil, err
	}
	tokenProxyAddress, err := exchange.GetAssetProxy(nil, types.ERC20ProxyID)
	if err != nil {
		log.Printf("error getting tokenProxyAddress")
		return nil, err
	}
	return NewAllowanceBlockConsumer(bitDexProxyAddress.Big(), tokenProxyAddress.Big(), feeTokenAddress.String(), client, publisher), nil
}
