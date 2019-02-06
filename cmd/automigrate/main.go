package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/jinzhu/gorm"
	"github.com/notegio/openrelay/common"
	dbModule "github.com/notegio/openrelay/db"
	poolModule "github.com/notegio/openrelay/pool"
	"github.com/notegio/openrelay/types"
)

const terms = `In signing this statement and using OpenRelay, I agree to abide by all terms outlined in the OpenRelay Terms of Use.

As a required condition before I am permitted to trade on OpenRelay, I explicitly acknowledge:

1. OpenRelay is a U.S. company not registered as an exchange with the U.S. Securities and Exchange Commission, and
2. OpenRelay is not exempt from registration requirements under any valid exemption,

And I agree not use OpenRelay's services to trade:

1. any asset that the SEC has declared a security, or
2. any asset that I have (or should have) reason to believe could be classified as a security, or
3. any asset intended to induce another to trade by means of deception or fraud, including but not limited to assets named or marketed to look like a different asset of greater value.
4. any asset that may violate any other law or regulation of the United States, including state and local laws and regulations.

I understand that if I am discovered to be in (intentional or accidental) violation of these terms, OpenRelay may take any action necessary to maintain lawful operations, Up to and Including (but not limited to):

1. Removing my orders from the order book,
2. Temporarily or permanently banning me or my accounts from access to OpenRelay,
3. Reporting my actions and any available identifying information to any relevant investigatory or enforcement authority, or
4. Seeking any appropriate legal or equitable remedy that may be available to OpenRelay resulting from any violation of these terms.`

func main() {
	db, err := dbModule.GetDB(os.Args[1], os.Args[2])
	if err != nil {
		log.Fatalf("Could not open database connection: %v", err.Error())
	}
	// Migrate tables
	if err := db.AutoMigrate(&dbModule.AssetProxy{}).Error; err != nil {
		log.Fatalf("Error migrating asset proxy table: %v", err.Error())
	}
	if err := db.AutoMigrate(&dbModule.Asset{}).Error; err != nil {
		log.Fatalf("Error migrating asset table: %v", err.Error())
	}
	if err := db.AutoMigrate(&dbModule.AssetPair{}).Error; err != nil {
		log.Fatalf("Error migrating asset pair table: %v", err.Error())
	}
	if err := db.AutoMigrate(&dbModule.Order{}).Error; err != nil {
		log.Fatalf("Error migrating order table: %v", err.Error())
	}
	if err := db.AutoMigrate(&dbModule.Cancellation{}).Error; err != nil {
		log.Fatalf("Error migrating cancellation table: %v", err.Error())
	}
	if err := db.AutoMigrate(&dbModule.Exchange{}).Error; err != nil {
		log.Fatalf("Error migrating exchange table: %v", err.Error())
	}
	if err := db.AutoMigrate(&dbModule.Terms{}).Error; err != nil {
		log.Fatalf("Error migrating terms table: %v", err.Error())
	}
	if err := db.AutoMigrate(&dbModule.TermsSig{}).Error; err != nil {
		log.Fatalf("Error migrating term_sigs table: %v", err.Error())
	}
	if err := db.AutoMigrate(&dbModule.HashMask{}).Error; err != nil {
		log.Fatalf("Error migrating hash_masks table: %v", err.Error())
	}
	if err := db.AutoMigrate(&poolModule.Pool{}).Error; err != nil {
		log.Fatalf("Error migrating pools table: %v", err.Error())
	}
	// Fill tables with initial data
	fillExchanges(db)
	fillAssetProxies(db)
	fillAssets(db)
	fillAssetPairs(db)
	fillTerms(db)
	// Add indexex
	if err := db.Model(&dbModule.Order{}).AddIndex("idx_order_maker_asset_taker_asset_data", "maker_asset_data", "taker_asset_data").Error; err != nil {
		log.Fatalf("Error adding order table index: %v", err.Error())
	}
	if err := db.Model(&dbModule.AssetProxy{}).AddForeignKey("exchange_address", "exchanges(address)", "RESTRICT", "CASCADE").Error; err != nil {
		log.Fatalf("Error adding asset proxy table foreign key: %v", err.Error())
	}
	if err := db.Model(&dbModule.Asset{}).AddForeignKey("proxy_id", "asset_proxies(id)", "RESTRICT", "CASCADE").Error; err != nil {
		log.Fatalf("Error adding asset table foreign key: %v", err.Error())
	}
	if err := db.Model(&dbModule.AssetPair{}).AddForeignKey("asset_symbol_a", "assets(symbol)", "RESTRICT", "CASCADE").Error; err != nil {
		log.Fatalf("Error adding asset pair table foreign key: %v", err.Error())
	}
	if err := db.Model(&dbModule.AssetPair{}).AddForeignKey("asset_symbol_b", "assets(symbol)", "RESTRICT", "CASCADE").Error; err != nil {
		log.Fatalf("Error adding asset pair table foreign key: %v", err.Error())
	}

	poolHash := sha3.NewKeccak256()
	poolHash.Write([]byte(""))

	pool := &poolModule.Pool{
		SearchTerms:     "",
		Expiration:      1744733652,
		Nonce:           0,
		FeeShare:        "1000000000000000000",
		ID:              poolHash.Sum(nil),
		SenderAddresses: types.NetworkAddressMap{},
		FilterAddresses: types.NetworkAddressMap{},
	}

	err = db.Model(&poolModule.Pool{}).Where("id = ?", pool.ID).Assign(pool).FirstOrCreate(pool).Error

	for _, credString := range os.Args[3:] {
		creds := strings.Split(credString, ";")
		if len(creds) != 3 {
			log.Printf("Malformed credential string: %v", credString)
			continue
		}
		username, passwordURI, permissions := creds[0], creds[1], creds[2]
		password := common.GetSecret(passwordURI)
		if dialect := db.Dialect().GetName(); dialect == "postgres" {
			// I don't like using string formatting instead of paramterization, but I
			// don't know of a way to parameterize the username in this statement. It
			// should still be fairly safe, because if you're able to execute this
			// command you already have administrative database access.
			if err = db.Exec(fmt.Sprintf("CREATE USER %v WITH PASSWORD '%v'", username, password)).Error; err != nil {
				log.Printf(err.Error())
			}
			for _, permission := range strings.Split(permissions, ",") {
				permArray := strings.Split(permission, ".")
				if len(permArray) != 2 {
					log.Printf("Malformed permission string '$v'", permission)
					continue
				}
				table, permission := permArray[0], permArray[1]
				// I don't like using string formatting instead of paramterization, but I
				// don't know of a way to parameterize the elements in this statement. It
				// should still be fairly safe, because if you're able to execute this
				// command you already have administrative database access.
				if err = db.Exec(fmt.Sprintf("GRANT %v ON TABLE %v TO %v", permission, table, username)).Error; err != nil {
					log.Printf(err.Error())
				}
			}
			if err = db.Exec(fmt.Sprintf("GRANT USAGE, SELECT on ALL SEQUENCES in SCHEMA public to %v", username)).Error; err != nil {
				log.Printf(err.Error())
			}
		} else if dialect == "mysql" {
			if err := db.Exec(fmt.Sprintf("CREATE USER '%v' IDENTIFIED BY '%v'", username, password)).Error; err != nil {
				log.Printf(err.Error())
			}
			result := make(map[string]string)
			if err := db.Exec("SELECT DATABASE()").Row().Scan(result); err != nil {
				log.Printf(err.Error())
			}
			log.Printf("'%v'", result)
			databaseName := result["DATABASE()"]
			log.Printf("Database name: %v", databaseName)
			for _, permission := range strings.Split(permissions, ",") {
				permArray := strings.Split(permission, ".")
				if len(permArray) != 2 {
					log.Printf("Malformed permission string '$v'", permission)
					continue
				}
				table, permission := permArray[0], permArray[1]
				// I don't like using string formatting instead of paramterization, but I
				// don't know of a way to parameterize the elements in this statement. It
				// should still be fairly safe, because if you're able to execute this
				// command you already have administrative database access.
				if err = db.Exec(fmt.Sprintf("GRANT %v ON %v.%v TO '%v'", permission, databaseName, table, username)).Error; err != nil {
					log.Printf(err.Error())
				}
			}
			if err := db.Exec("FLUSH PRIVILEGES;").Error; err != nil {
				log.Printf(err.Error())
			}
		}
		log.Printf("Created '%v'", credString)
	}
}

func fillExchanges(db *gorm.DB) {

	var result *gorm.DB

	mainnetAddress, _ := common.HexToAddress("0x4f833a24e1f95d70f028921e27040ca56e09ab0b")
	result = db.Model(&dbModule.Exchange{}).Create(&dbModule.Exchange{Network: 1, Address: mainnetAddress})
	if result.Error != nil {
		log.Fatalf("Error adding exchange with address %s: %v", mainnetAddress.String(), result.Error.Error())
	}

	ropstenAddress, _ := common.HexToAddress("0x4530c0483a1633c7a1c97d2c53721caff2caaaaf")
	result = db.Model(&dbModule.Exchange{}).Create(&dbModule.Exchange{Network: 3, Address: ropstenAddress})
	if result.Error != nil {
		log.Fatalf("Error adding exchange with address %s: %v", ropstenAddress.String(), result.Error.Error())
	}

	rinkebyAddress, _ := common.HexToAddress("0x25d26d6a7c86a250b7af3893b46b519388389bbe")
	result = db.Model(&dbModule.Exchange{}).Create(&dbModule.Exchange{Network: 4, Address: rinkebyAddress})
	if result.Error != nil {
		log.Fatalf("Error adding exchange with address %s: %v", rinkebyAddress.String(), result.Error.Error())
	}

	kovanAddress, _ := common.HexToAddress("0x35dd2932454449b14cee11a94d3674a936d5d7b2")
	result = db.Model(&dbModule.Exchange{}).Create(&dbModule.Exchange{Network: 42, Address: kovanAddress})
	if result.Error != nil {
		log.Fatalf("Error adding exchange with address %s: %v", kovanAddress.String(), result.Error.Error())
	}

	ganacheAddress, _ := common.HexToAddress("0x48bacb9266a570d521063ef5dd96e61686dbe788")
	result = db.Model(&dbModule.Exchange{}).Create(&dbModule.Exchange{Network: 50, Address: ganacheAddress})
	if result.Error != nil {
		log.Fatalf("Error adding exchange with address %s: %v", ganacheAddress.String(), result.Error.Error())
	}
}

func fillAssetProxies(db *gorm.DB) {

	erc20ProxyID, _ := hex.DecodeString("f47261b0")
	erc721ProxyID, _ := hex.DecodeString("02571792")
	roboDexProxyID, _ := hex.DecodeString("343adc23")

	exchangeAddress, _ := common.HexToAddress("0x25d26d6a7c86a250b7af3893b46b519388389bbe")

	var result *gorm.DB

	erc20ProxyAddress, _ := common.HexToAddress("0xb8350417b8ff3431c90a290c856ddfaa72b7ac02")
	result = db.Model(&dbModule.AssetProxy{}).Create(&dbModule.AssetProxy{
		ID:              erc20ProxyID,
		Name:            "ERC20",
		Address:         erc20ProxyAddress,
		ExchangeAddress: exchangeAddress,
		ZeroEx:          false,
	})
	if result.Error != nil {
		log.Fatalf("Error adding asset proxy with address %s: %v", erc20ProxyAddress.String(), result.Error.Error())
	}

	erc721ProxyAddress, _ := common.HexToAddress("0x0fcb30cdb4a799d6109d3d80fabb528d3d642780")
	result = db.Model(&dbModule.AssetProxy{}).Create(&dbModule.AssetProxy{
		ID:              erc721ProxyID,
		Name:            "ERC721",
		Address:         erc721ProxyAddress,
		ExchangeAddress: exchangeAddress,
		ZeroEx:          false,
	})
	if result.Error != nil {
		log.Fatalf("Error adding asset proxy with address %s: %v", erc721ProxyAddress.String(), result.Error.Error())
	}

	roboDexProxyAddress, _ := common.HexToAddress("0xa0719d553d326c34b135d120f2bb366843aa635c")
	result = db.Model(&dbModule.AssetProxy{}).Create(&dbModule.AssetProxy{
		ID:              roboDexProxyID,
		Name:            "RoboDEX",
		Address:         roboDexProxyAddress,
		ExchangeAddress: exchangeAddress,
		ZeroEx:          false,
	})
	if result.Error != nil {
		log.Fatalf("Error adding asset proxy with address %s: %v", roboDexProxyAddress.String(), result.Error.Error())
	}
}

func fillAssets(db *gorm.DB) {

	minUint256, _ := new(big.Int).SetString("0", 10)
	maxUint256, _ := new(big.Int).SetString("115792089237316195423570985008687907853269984665640564039457584007913129639935", 10)

	erc20ProxyID, _ := hex.DecodeString("f47261b0")
	//erc721ProxyID, _ := hex.DecodeString("02571792")
	roboDexProxyID, _ := hex.DecodeString("343adc23")

	var result *gorm.DB

	wethAssetAddress, _ := common.HexToAddress("0x7acac581a8ca077f1a4547a165983d1ee4dca168")
	wethAssetData, _ := common.HexToAssetData("0xf47261b00000000000000000000000007acac581a8ca077f1a4547a165983d1ee4dca168")
	result = db.Model(&dbModule.Asset{}).Create(&dbModule.Asset{
		Symbol:         "WETH",
		Name:           "WETH",
		Address:        wethAssetAddress,
		Decimals:       18,
		ProxyID:        erc20ProxyID,
		Data:           &wethAssetData,
		Precision:      6,
		MinTradeAmount: common.BigToUint256(minUint256),
		MaxTradeAmount: common.BigToUint256(maxUint256),
		ZeroEx:         false,
		Active:         true,
		Quote:          false,
	})
	if result.Error != nil {
		log.Fatalf("Error adding asset with address %s: %v", wethAssetAddress.String(), result.Error.Error())
	}

	rdxAssetAddress, _ := common.HexToAddress("0x95f69a06397211699fd86a98b4cd3bc3aa7599dd")
	rdxAssetData, _ := common.HexToAssetData("0x5d388e1700000000000000000000000095f69a06397211699fd86a98b4cd3bc3aa7599dd0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000700000000000000000000000000000000000000000000000000000000000000ff")
	result = db.Model(&dbModule.Asset{}).Create(&dbModule.Asset{
		Symbol:         "RDX",
		Name:           "RoboDEX Token",
		Address:        rdxAssetAddress,
		Decimals:       18,
		ProxyID:        roboDexProxyID,
		Data:           &rdxAssetData,
		Precision:      6,
		MinTradeAmount: common.BigToUint256(minUint256),
		MaxTradeAmount: common.BigToUint256(maxUint256),
		ZeroEx:         false,
		Active:         true,
		Quote:          false,
	})
	if result.Error != nil {
		log.Fatalf("Error adding asset with address %s: %v", rdxAssetAddress.String(), result.Error.Error())
	}

	weth9AssetAddress, _ := common.HexToAddress("0xc778417e063141139fce010982780140aa0cd5ab")
	weth9AssetData, _ := common.HexToAssetData("0xf47261b0000000000000000000000000c778417e063141139fce010982780140aa0cd5ab")
	result = db.Model(&dbModule.Asset{}).Create(&dbModule.Asset{
		Symbol:         "WETH9",
		Name:           "WETH9",
		Address:        weth9AssetAddress,
		Decimals:       18,
		ProxyID:        erc20ProxyID,
		Data:           &weth9AssetData,
		Precision:      6,
		MinTradeAmount: common.BigToUint256(minUint256),
		MaxTradeAmount: common.BigToUint256(maxUint256),
		ZeroEx:         true,
		Active:         true,
		Quote:          false,
	})
	if result.Error != nil {
		log.Fatalf("Error adding asset with address %s: %v", weth9AssetAddress.String(), result.Error.Error())
	}

	zrxAssetAddress, _ := common.HexToAddress("0x2727e688b8fd40b198cd5fe6e408e00494a06f07")
	zrxAssetData, _ := common.HexToAssetData("0xf47261b00000000000000000000000002727e688b8fd40b198cd5fe6e408e00494a06f07")
	result = db.Model(&dbModule.Asset{}).Create(&dbModule.Asset{
		Symbol:         "ZRX",
		Name:           "ZRX",
		Address:        zrxAssetAddress,
		Decimals:       18,
		ProxyID:        erc20ProxyID,
		Data:           &zrxAssetData,
		Precision:      6,
		MinTradeAmount: common.BigToUint256(minUint256),
		MaxTradeAmount: common.BigToUint256(maxUint256),
		ZeroEx:         true,
		Active:         true,
		Quote:          false,
	})
	if result.Error != nil {
		log.Fatalf("Error adding asset with address %s: %v", zrxAssetAddress.String(), result.Error.Error())
	}
}

func fillAssetPairs(db *gorm.DB) {

	var result *gorm.DB

	result = db.Model(&dbModule.AssetPair{}).Create(&dbModule.AssetPair{
		AssetSymbolA: "RDX",
		AssetSymbolB: "WETH",
		Active:       true,
	})
	if result.Error != nil {
		log.Fatalf("Error adding asset pair: %v", result.Error.Error())
	}

	result = db.Model(&dbModule.AssetPair{}).Create(&dbModule.AssetPair{
		AssetSymbolA: "WETH9",
		AssetSymbolB: "RDX",
		Active:       true,
	})
	if result.Error != nil {
		log.Fatalf("Error adding asset pair: %v", result.Error.Error())
	}

	result = db.Model(&dbModule.AssetPair{}).Create(&dbModule.AssetPair{
		AssetSymbolA: "ZRX",
		AssetSymbolB: "RDX",
		Active:       true,
	})
	if result.Error != nil {
		log.Fatalf("Error adding asset pair: %v", result.Error.Error())
	}

	result = db.Model(&dbModule.AssetPair{}).Create(&dbModule.AssetPair{
		AssetSymbolA: "ZRX",
		AssetSymbolB: "WETH",
		Active:       true,
	})
	if result.Error != nil {
		log.Fatalf("Error adding asset pair: %v", result.Error.Error())
	}
}

func fillTerms(db *gorm.DB) {
	if db.Model(&dbModule.Terms{}).First(&dbModule.Terms{}).RecordNotFound() {
		if err := dbModule.NewTermsManager(db).UpdateTerms("en", terms); err != nil {
			log.Fatalf("Error adding terms: %v", err.Error())
		}
	}
}
