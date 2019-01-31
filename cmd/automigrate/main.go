package main

import (
	dbModule "github.com/notegio/openrelay/db"
	poolModule "github.com/notegio/openrelay/pool"
	"github.com/notegio/openrelay/common"
	"github.com/notegio/openrelay/types"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"log"
	"os"
	"fmt"
	"strings"
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
	mainnetAddress, _ := common.HexToAddress("0x4f833a24e1f95d70f028921e27040ca56e09ab0b")
	db.Where(
		&dbModule.Exchange{Network: 1},
	).FirstOrCreate(&dbModule.Exchange{Network: 1, Address: mainnetAddress})
	ropstenAddress, _ := common.HexToAddress("0x4530c0483a1633c7a1c97d2c53721caff2caaaaf")
	db.Where(
		&dbModule.Exchange{Network: 3},
	).FirstOrCreate(&dbModule.Exchange{Network: 3, Address: ropstenAddress})
	rinkebyAddress, _ := common.HexToAddress("0x22ebc052f43a88efa06379426120718170f2204e")
	db.Where(
		&dbModule.Exchange{Network: 4},
	).FirstOrCreate(&dbModule.Exchange{Network: 4, Address: rinkebyAddress})
	kovanAddress, _ := common.HexToAddress("0x35dd2932454449b14cee11a94d3674a936d5d7b2")
	db.Where(
		&dbModule.Exchange{Network: 42},
	).FirstOrCreate(&dbModule.Exchange{Network: 42, Address: kovanAddress})
	ganacheAddress, _ := common.HexToAddress("0x48bacb9266a570d521063ef5dd96e61686dbe788")
	db.Where(
		&dbModule.Exchange{Network: 50},
	).FirstOrCreate(&dbModule.Exchange{Network: 50, Address: ganacheAddress})
	if db.Model(&dbModule.Terms{}).First(&dbModule.Terms{}).RecordNotFound() {
		if err := dbModule.NewTermsManager(db).UpdateTerms("en", terms); err != nil {
			log.Fatalf("Error setting terms: %v", err.Error())
		}
	}
	if err := db.Model(&dbModule.Order{}).AddIndex("idx_order_maker_asset_taker_asset_data", "maker_asset_data", "taker_asset_data").Error; err != nil {
		log.Fatalf("Error adding token pair index: %v", err.Error())
	}

	poolHash := sha3.NewKeccak256()
	poolHash.Write([]byte(""))

	pool := &poolModule.Pool{
		SearchTerms: "",
		Expiration: 1744733652,
		Nonce: 0,
		FeeShare: "1000000000000000000",
		ID: poolHash.Sum(nil),
		SenderAddresses: types.NetworkAddressMap{},
		FilterAddresses: types.NetworkAddressMap{},
	}

	err = db.Model(&poolModule.Pool{}).Where("id = ?", pool.ID).Assign(pool).FirstOrCreate(pool).Error



	for _, credString := range(os.Args[3:]) {
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
			for _, permission := range(strings.Split(permissions, ",")) {
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
				log.Printf(err.Error());
			}
		}
		log.Printf("Created '%v'", credString)
	}
}
