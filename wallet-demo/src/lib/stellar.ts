import { Networks } from "@stellar/stellar-sdk";

export function getStellarNetwork(): "testnet" | "mainnet" {
  const network = process.env.NEXT_PUBLIC_STELLAR_NETWORK || "testnet";
  return network === "mainnet" ? "mainnet" : "testnet";
}

export function getNetworkPassphrase(): string {
  return getStellarNetwork() === "mainnet"
    ? Networks.PUBLIC
    : Networks.TESTNET;
}
