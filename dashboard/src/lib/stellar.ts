import {
  StellarWalletsKit,
  WalletNetwork,
  allowAllModules,
  FREIGHTER_ID,
} from "@creit.tech/stellar-wallets-kit";

export function createWalletKit(network: "testnet" | "mainnet") {
  return new StellarWalletsKit({
    network:
      network === "mainnet" ? WalletNetwork.PUBLIC : WalletNetwork.TESTNET,
    selectedWalletId: FREIGHTER_ID,
    modules: allowAllModules(),
  });
}

export function getStellarNetwork(): "testnet" | "mainnet" {
  const network = process.env.NEXT_PUBLIC_STELLAR_NETWORK || "testnet";
  return network === "mainnet" ? "mainnet" : "testnet";
}

export function getNetworkPassphrase(): string {
  return getStellarNetwork() === "mainnet"
    ? WalletNetwork.PUBLIC
    : WalletNetwork.TESTNET;
}
