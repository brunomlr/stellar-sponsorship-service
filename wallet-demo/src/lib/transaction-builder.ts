import * as StellarSdk from "@stellar/stellar-sdk";

function getHorizonUrl(): string {
  const custom =
    typeof window !== "undefined"
      ? process.env.NEXT_PUBLIC_HORIZON_URL
      : undefined;
  if (custom) return custom;
  const network = process.env.NEXT_PUBLIC_STELLAR_NETWORK || "testnet";
  return network === "mainnet"
    ? "https://horizon.stellar.org"
    : "https://horizon-testnet.stellar.org";
}

function getNetworkPassphrase(): string {
  const network = process.env.NEXT_PUBLIC_STELLAR_NETWORK || "testnet";
  return network === "mainnet"
    ? StellarSdk.Networks.PUBLIC
    : StellarSdk.Networks.TESTNET;
}

/**
 * Builds an unsigned transaction that adds a trustline with sponsored reserves.
 *
 * Structure (matching verifier.go expectations):
 *   Source: userPublicKey
 *   Op 1: BEGIN_SPONSORING_FUTURE_RESERVES (source: sponsorPublicKey, sponsoredID: userPublicKey)
 *   Op 2: CHANGE_TRUST (source: userPublicKey, asset: assetCode/issuer)
 *   Op 3: END_SPONSORING_FUTURE_RESERVES (source: userPublicKey)
 */
export async function buildAddTrustlineTransaction(
  userPublicKey: string,
  sponsorPublicKey: string,
  assetCode: string,
  assetIssuer: string
): Promise<string> {
  const server = new StellarSdk.Horizon.Server(getHorizonUrl());
  const account = await server.loadAccount(userPublicKey);

  const asset = new StellarSdk.Asset(assetCode, assetIssuer);

  const tx = new StellarSdk.TransactionBuilder(account, {
    fee: StellarSdk.BASE_FEE,
    networkPassphrase: getNetworkPassphrase(),
  })
    .addOperation(
      StellarSdk.Operation.beginSponsoringFutureReserves({
        source: sponsorPublicKey,
        sponsoredId: userPublicKey,
      })
    )
    .addOperation(
      StellarSdk.Operation.changeTrust({
        source: userPublicKey,
        asset: asset,
      })
    )
    .addOperation(
      StellarSdk.Operation.endSponsoringFutureReserves({
        source: userPublicKey,
      })
    )
    .setTimeout(300)
    .build();

  return tx.toXDR();
}

/**
 * Builds an unsigned transaction that creates a new Stellar account with sponsored reserves.
 *
 * Structure:
 *   Source: userPublicKey (the existing funded account)
 *   Op 1: BEGIN_SPONSORING_FUTURE_RESERVES (source: sponsorPublicKey, sponsoredID: newAccountPublicKey)
 *   Op 2: CREATE_ACCOUNT (source: userPublicKey, dest: newAccountPublicKey, startingBalance: "0")
 *   Op 3: END_SPONSORING_FUTURE_RESERVES (source: newAccountPublicKey)
 *
 * Note: The new account doesn't need to sign — Stellar protocol allows END_SPONSORING
 * from an account being created in the same transaction.
 */
export async function buildCreateAccountTransaction(
  userPublicKey: string,
  sponsorPublicKey: string,
  newAccountPublicKey: string
): Promise<string> {
  const server = new StellarSdk.Horizon.Server(getHorizonUrl());
  const account = await server.loadAccount(userPublicKey);

  const tx = new StellarSdk.TransactionBuilder(account, {
    fee: StellarSdk.BASE_FEE,
    networkPassphrase: getNetworkPassphrase(),
  })
    .addOperation(
      StellarSdk.Operation.beginSponsoringFutureReserves({
        source: sponsorPublicKey,
        sponsoredId: newAccountPublicKey,
      })
    )
    .addOperation(
      StellarSdk.Operation.createAccount({
        source: userPublicKey,
        destination: newAccountPublicKey,
        startingBalance: "0",
      })
    )
    .addOperation(
      StellarSdk.Operation.endSponsoringFutureReserves({
        source: newAccountPublicKey,
      })
    )
    .setTimeout(300)
    .build();

  return tx.toXDR();
}

/**
 * Submits a fully-signed transaction XDR to the Stellar network via Horizon.
 */
export async function submitTransaction(signedXdr: string) {
  const server = new StellarSdk.Horizon.Server(getHorizonUrl());
  const tx = StellarSdk.TransactionBuilder.fromXDR(
    signedXdr,
    getNetworkPassphrase()
  );
  return server.submitTransaction(tx);
}
