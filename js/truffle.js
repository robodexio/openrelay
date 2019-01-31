const PrivateKeyProvider = require('truffle-privatekey-provider')

module.exports = {
  networks: {
    main: {
      provider: () => new PrivateKeyProvider(process.env.ETHEREUM_DEPLOYER_PRIVATE_KEY, process.env.ETHEREUM_URL),
      network_id: "*"
    },
    development: {
      host: "localhost",
      port: 8546,
      network_id: "*" // Match any network id
    },
    testnet: {
      host: "ethnode",
      port: 8545,
      network_id: "*"
    },
    parity: {
      host: "172.17.0.4",
      port: 8545,
      network_id: "*" // Match any network id
    }
  }
};
