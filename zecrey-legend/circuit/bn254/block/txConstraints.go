/*
 * Copyright © 2021 Zecrey Protocol
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package block

import (
	"errors"
	"github.com/consensys/gnark/std/hash/mimc"
	"github.com/zecrey-labs/zecrey-crypto/zecrey-legend/circuit/bn254/std"
	"log"
)

type TxConstraints struct {
	// tx type
	TxType Variable
	// different transactions
	RegisterZnsTxInfo     RegisterZnsTxConstraints
	DepositTxInfo         DepositTxConstraints
	DepositNftTxInfo      DepositNftTxConstraints
	TransferTxInfo        TransferTxConstraints
	SwapTxInfo            SwapTxConstraints
	AddLiquidityTxInfo    AddLiquidityTxConstraints
	RemoveLiquidityTxInfo RemoveLiquidityTxConstraints
	MintNftTxInfo         MintNftTxConstraints
	TransferNftTxInfo     TransferNftTxConstraints
	SetNftPriceTxInfo     SetNftPriceTxConstraints
	BuyNftTxInfo          BuyNftTxConstraints
	WithdrawTxInfo        WithdrawTxConstraints
	WithdrawNftTxInfo     WithdrawNftTxConstraints
	// signature
	Signature SignatureConstraints
	// account root before
	AccountRootBefore Variable
	// account before info, size is 4
	AccountsInfoBefore [NbAccountsPerTx]std.AccountConstraints
	// before account asset merkle proof
	MerkleProofsAccountAssetsBefore       [NbAccountsPerTx][NbAccountAssetsPerAccount][AssetMerkleLevels]Variable
	MerkleProofsHelperAccountAssetsBefore [NbAccountsPerTx][NbAccountAssetsPerAccount][AssetMerkleHelperLevels]Variable
	// before account liquidity merkle proof
	MerkleProofsAccountLiquidityBefore       [NbAccountsPerTx][LiquidityMerkleLevels]Variable
	MerkleProofsHelperAccountLiquidityBefore [NbAccountsPerTx][LiquidityMerkleHelperLevels]Variable
	// before account nft tree merkle proof
	MerkleProofsAccountNftBefore       [NbAccountsPerTx][NftMerkleLevels]Variable
	MerkleProofsHelperAccountNftBefore [NbAccountsPerTx][NftMerkleHelperLevels]Variable
	// before account merkle proof
	MerkleProofsAccountBefore       [NbAccountsPerTx][AccountMerkleLevels]Variable
	MerkleProofsHelperAccountBefore [NbAccountsPerTx][AccountMerkleHelperLevels]Variable
	// account root after
	AccountRootAfter Variable
}

func (circuit TxConstraints) Define(api API) error {
	// mimc
	hFunc, err := mimc.NewMiMC(api)
	if err != nil {
		return err
	}

	err = VerifyTransaction(api, circuit, hFunc, NilHash)
	if err != nil {
		return err
	}
	return nil
}

func VerifyTransaction(
	api API,
	tx TxConstraints,
	hFunc MiMC,
	nilHash Variable,
) error {
	// compute tx type
	isEmptyTx := api.IsZero(api.Sub(tx.TxType, TxTypeEmptyTx))
	isRegisterZnsTx := api.IsZero(api.Sub(tx.TxType, TxTypeRegisterZns))
	isDepositTx := api.IsZero(api.Sub(tx.TxType, TxTypeDeposit))
	isDepositNftTx := api.IsZero(api.Sub(tx.TxType, TxTypeDepositNft))
	isTransferTx := api.IsZero(api.Sub(tx.TxType, TxTypeTransfer))
	isSwapTx := api.IsZero(api.Sub(tx.TxType, TxTypeSwap))
	isAddLiquidityTx := api.IsZero(api.Sub(tx.TxType, TxTypeAddLiquidity))
	isRemoveLiquidityTx := api.IsZero(api.Sub(tx.TxType, TxTypeRemoveLiquidity))
	isWithdrawTx := api.IsZero(api.Sub(tx.TxType, TxTypeWithdraw))
	isMintNftTx := api.IsZero(api.Sub(tx.TxType, TxTypeMintNft))
	isTransferNftTx := api.IsZero(api.Sub(tx.TxType, TxTypeTransferNft))
	isSetNftPriceTx := api.IsZero(api.Sub(tx.TxType, TxTypeSetNftPrice))
	isBuyNftTx := api.IsZero(api.Sub(tx.TxType, TxTypeBuyNft))
	isWithdrawNftTx := api.IsZero(api.Sub(tx.TxType, TxTypeWithdrawNft))

	// no need to verify signature transaction
	notNoSignatureTx := api.IsZero(api.Or(isEmptyTx, api.Or(api.Or(isRegisterZnsTx, isDepositTx), isDepositNftTx)))

	// get hash value from tx based on tx type
	// transfer tx
	hashVal := std.ComputeHashFromTransferTx(tx.TransferTxInfo, tx.AccountsInfoBefore[0].Nonce, hFunc)
	// swap tx
	hashValCheck := std.ComputeHashFromSwapTx(tx.SwapTxInfo, tx.AccountsInfoBefore[0].Nonce, hFunc)
	hashVal = api.Select(isSwapTx, hashValCheck, hashVal)
	// add liquidity tx
	hashValCheck = std.ComputeHashFromAddLiquidityTx(tx.AddLiquidityTxInfo, tx.AccountsInfoBefore[0].Nonce, hFunc)
	hashVal = api.Select(isAddLiquidityTx, hashValCheck, hashVal)
	// remove liquidity tx
	hashValCheck = std.ComputeHashFromRemoveLiquidityTx(tx.RemoveLiquidityTxInfo, tx.AccountsInfoBefore[0].Nonce, hFunc)
	hashVal = api.Select(isRemoveLiquidityTx, hashValCheck, hashVal)
	// withdraw tx
	hashValCheck = std.ComputeHashFromWithdrawTx(tx.WithdrawTxInfo, tx.AccountsInfoBefore[0].Nonce, hFunc)
	hashVal = api.Select(isWithdrawTx, hashValCheck, hashVal)
	// mint nft tx
	hashValCheck = std.ComputeHashFromMintNftTx(tx.MintNftTxInfo, tx.AccountsInfoBefore[0].Nonce, hFunc)
	hashVal = api.Select(isMintNftTx, hashValCheck, hashVal)
	// transfer nft tx
	hashValCheck = std.ComputeHashFromTransferNftTx(tx.TransferNftTxInfo, tx.AccountsInfoBefore[0].Nonce, hFunc)
	hashVal = api.Select(isTransferNftTx, hashValCheck, hashVal)
	// set nft price tx
	hashValCheck = std.ComputeHashFromSetNftPriceTx(tx.SetNftPriceTxInfo, tx.AccountsInfoBefore[0].Nonce, hFunc)
	hashVal = api.Select(isSetNftPriceTx, hashValCheck, hashVal)
	// buy nft tx
	hashValCheck = std.ComputeHashFromBuyNftTx(tx.BuyNftTxInfo, tx.AccountsInfoBefore[0].Nonce, hFunc)
	hashVal = api.Select(isBuyNftTx, hashValCheck, hashVal)
	// withdraw nft tx
	hashValCheck = std.ComputeHashFromWithdrawNftTx(tx.WithdrawNftTxInfo, tx.AccountsInfoBefore[0].Nonce, hFunc)
	hashVal = api.Select(isWithdrawNftTx, hashValCheck, hashVal)
	hFunc.Reset()
	// verify signature
	err := std.VerifyEddsaSig(
		notNoSignatureTx,
		api,
		hFunc,
		hashVal,
		tx.AccountsInfoBefore[0].AccountPk,
		tx.Signature,
	)
	if err != nil {
		log.Println("[VerifyTx] invalid signature:", err)
		return err
	}

	// verify transactions
	std.VerifyDepositTx(api, isDepositTx, tx.DepositTxInfo, tx.AccountsInfoBefore)
	std.VerifyDepositNftTx(api, isDepositNftTx, nilHash, tx.DepositNftTxInfo, tx.AccountsInfoBefore)
	std.VerifyTransferTx(api, isTransferTx, nilHash, tx.TransferTxInfo, tx.AccountsInfoBefore)
	std.VerifySwapTx(api, isSwapTx, tx.SwapTxInfo, tx.AccountsInfoBefore)
	std.VerifyAddLiquidityTx(api, isAddLiquidityTx, tx.AddLiquidityTxInfo, tx.AccountsInfoBefore)
	std.VerifyRemoveLiquidityTx(api, isRemoveLiquidityTx, tx.RemoveLiquidityTxInfo, tx.AccountsInfoBefore)
	std.VerifyWithdrawTx(api, isWithdrawTx, tx.WithdrawTxInfo, tx.AccountsInfoBefore)
	std.VerifyMintNftTx(api, isMintNftTx, nilHash, tx.MintNftTxInfo, tx.AccountsInfoBefore)
	std.VerifyTransferNftTx(api, isTransferNftTx, nilHash, tx.TransferNftTxInfo, tx.AccountsInfoBefore)
	std.VerifySetNftPriceTx(api, isSetNftPriceTx, tx.SetNftPriceTxInfo, tx.AccountsInfoBefore)
	std.VerifyBuyNftTx(api, isBuyNftTx, nilHash, tx.BuyNftTxInfo, tx.AccountsInfoBefore)
	std.VerifyWithdrawNftTx(api, isWithdrawNftTx, nilHash, tx.WithdrawNftTxInfo, tx.AccountsInfoBefore)
	// deposit
	deltas := GetAccountDeltasFromDeposit(api, tx.DepositTxInfo)
	// generic transfer
	deltasCheck := GetAccountDeltasFromTransfer(api, tx.TransferTxInfo)
	deltas = SelectDeltas(api, isTransferTx, deltasCheck, deltas)
	// swap
	deltasCheck = GetAccountDeltasFromSwap(api, tx.SwapTxInfo)
	deltas = SelectDeltas(api, isSwapTx, deltasCheck, deltas)
	// add liquidity
	deltasCheck = GetAccountDeltasFromAddLiquidity(api, tx.AddLiquidityTxInfo)
	deltas = SelectDeltas(api, isAddLiquidityTx, deltasCheck, deltas)
	// remove liquidity
	deltasCheck = GetAccountDeltasFromRemoveLiquidity(api, tx.RemoveLiquidityTxInfo)
	deltas = SelectDeltas(api, isRemoveLiquidityTx, deltasCheck, deltas)
	// withdraw
	deltasCheck = GetAccountDeltasFromWithdraw(api, tx.WithdrawTxInfo)
	deltas = SelectDeltas(api, isWithdrawTx, deltasCheck, deltas)
	// deposit nft
	nftDeltas := GetAccountDeltasFromDepositNft(api, tx.DepositNftTxInfo)
	// mint nft
	deltasCheck, nftDeltasCheck := GetAccountDeltasFromMintNft(api, tx.MintNftTxInfo)
	deltas = SelectDeltas(api, isMintNftTx, deltasCheck, deltas)
	nftDeltas = SelectNftDeltas(api, isMintNftTx, nftDeltasCheck, nftDeltas)
	// transfer nft
	deltasCheck, nftDeltasCheck = GetAccountDeltasFromTransferNft(api, tx.TransferNftTxInfo)
	deltas = SelectDeltas(api, isTransferNftTx, deltasCheck, deltas)
	nftDeltas = SelectNftDeltas(api, isTransferNftTx, nftDeltasCheck, nftDeltas)
	// set nft price
	deltasCheck, nftDeltasCheck = GetAccountDeltasFromSetNftPrice(api, tx.SetNftPriceTxInfo)
	deltas = SelectDeltas(api, isSetNftPriceTx, deltasCheck, deltas)
	nftDeltas = SelectNftDeltas(api, isSetNftPriceTx, nftDeltasCheck, nftDeltas)
	// buy nft
	deltasCheck, nftDeltasCheck = GetAccountDeltasFromBuyNft(api, tx.BuyNftTxInfo)
	deltas = SelectDeltas(api, isBuyNftTx, deltasCheck, deltas)
	nftDeltas = SelectNftDeltas(api, isBuyNftTx, nftDeltasCheck, nftDeltas)
	// withdraw nft
	deltasCheck, nftDeltasCheck = GetAccountDeltasFromWithdrawNft(api, tx.WithdrawNftTxInfo)
	deltas = SelectDeltas(api, isWithdrawNftTx, deltasCheck, deltas)
	nftDeltas = SelectNftDeltas(api, isWithdrawNftTx, nftDeltasCheck, nftDeltas)
	// update accounts
	AccountsInfoAfter := UpdateAccounts(api, tx.AccountsInfoBefore, deltas, nftDeltas)

	NewAccountRoot := tx.MerkleProofsAccountBefore[0][AccountMerkleLevels-1]
	for i := 0; i < NbAccountsPerTx; i++ {
		var (
			NewAccountAssetsRoot    = tx.MerkleProofsAccountAssetsBefore[i][0][AssetMerkleLevels-1]
			NewAccountLiquidityRoot = tx.MerkleProofsAccountLiquidityBefore[i][AssetMerkleLevels-1]
			NewAccountNftRoot       = tx.MerkleProofsAccountNftBefore[i][NftMerkleLevels-1]
		)
		notFirstAccount := api.IsZero(api.IsZero(i))
		tx.MerkleProofsAccountBefore[i][AccountMerkleLevels-1] = api.Select(
			notFirstAccount,
			NewAccountRoot,
			tx.MerkleProofsAccountBefore[i][AccountMerkleLevels-1],
		)
		// verify account asset node hash
		/*
			Index    Variable
			Balance  Variable
			AssetAId Variable
			AssetBId Variable
			AssetA   Variable
			AssetB   Variable
			LpAmount Variable
		*/
		for j := 0; j < NbAccountAssetsPerAccount; j++ {
			notFirst := api.IsZero(api.IsZero(j))
			tx.MerkleProofsAccountAssetsBefore[i][j][AssetMerkleLevels-1] = api.Select(
				notFirst, NewAccountAssetsRoot,
				tx.MerkleProofsAccountAssetsBefore[i][j][AssetMerkleLevels-1],
			)
			hFunc.Reset()
			hFunc.Write(
				tx.AccountsInfoBefore[i].AssetsInfo[j].AssetId,
				tx.AccountsInfoBefore[i].AssetsInfo[j].Balance,
			)
			assetNodeHash := hFunc.Sum()
			notNilAssetRoot := api.IsZero(api.IsZero(api.Sub(tx.AccountsInfoBefore[i].AccountAssetsRoot, nilHash)))
			std.IsVariableEqual(api, notNilAssetRoot, tx.MerkleProofsAccountAssetsBefore[i][j][0], assetNodeHash)
			// verify account asset merkle proof
			hFunc.Reset()
			std.VerifyMerkleProof(
				api,
				notNilAssetRoot,
				hFunc,
				tx.AccountsInfoBefore[i].AccountAssetsRoot,
				tx.MerkleProofsAccountAssetsBefore[i][j][:],
				tx.MerkleProofsHelperAccountAssetsBefore[i][j][:],
			)
			hFunc.Reset()
			hFunc.Write(
				AccountsInfoAfter[i].AssetsInfo[j].AssetId,
				AccountsInfoAfter[i].AssetsInfo[j].Balance,
			)
			assetNodeHash = hFunc.Sum()
			hFunc.Reset()
			// update assetNode hash
			tx.MerkleProofsAccountAssetsBefore[i][j][0] = assetNodeHash
			// update merkle proof
			NewAccountAssetsRoot = std.UpdateMerkleProof(api, hFunc, tx.MerkleProofsAccountAssetsBefore[i][j][:], tx.MerkleProofsHelperAccountAssetsBefore[i][j][:])
		}
		// verify account liquidity node hash
		hFunc.Reset()
		hFunc.Write(
			tx.AccountsInfoBefore[i].LiquidityInfo.PairIndex,
			tx.AccountsInfoBefore[i].LiquidityInfo.AssetAId,
			tx.AccountsInfoBefore[i].LiquidityInfo.AssetAAmount,
			tx.AccountsInfoBefore[i].LiquidityInfo.AssetBId,
			tx.AccountsInfoBefore[i].LiquidityInfo.AssetBAmount,
			tx.AccountsInfoBefore[i].LiquidityInfo.LpAmount,
		)
		liquidityNodeHash := hFunc.Sum()
		isLiquidityTx := api.Or(isSwapTx, api.Or(isAddLiquidityTx, isRemoveLiquidityTx))
		notNilLiquidityRootAndIsLiquidityTx := api.And(api.IsZero(api.IsZero(api.Sub(tx.AccountsInfoBefore[i].AccountLiquidityRoot, nilHash))), isLiquidityTx)
		std.IsVariableEqual(api, notNilLiquidityRootAndIsLiquidityTx, tx.MerkleProofsAccountLiquidityBefore[i][0], liquidityNodeHash)
		// verify account liquidity merkle proof
		hFunc.Reset()
		std.VerifyMerkleProof(
			api,
			notNilLiquidityRootAndIsLiquidityTx,
			hFunc,
			tx.AccountsInfoBefore[i].AccountNftRoot,
			tx.MerkleProofsAccountLiquidityBefore[i][:],
			tx.MerkleProofsHelperAccountLiquidityBefore[i][:],
		)
		hFunc.Reset()
		hFunc.Write(
			AccountsInfoAfter[i].LiquidityInfo.PairIndex,
			AccountsInfoAfter[i].LiquidityInfo.AssetAId,
			AccountsInfoAfter[i].LiquidityInfo.AssetAAmount,
			AccountsInfoAfter[i].LiquidityInfo.AssetBId,
			AccountsInfoAfter[i].LiquidityInfo.AssetBAmount,
			AccountsInfoAfter[i].LiquidityInfo.LpAmount,
		)
		liquidityNodeHash = hFunc.Sum()
		hFunc.Reset()
		// update assetNode hash
		tx.MerkleProofsAccountLiquidityBefore[i][0] = liquidityNodeHash
		// update merkle proof
		NewAccountLiquidityRoot = std.UpdateMerkleProof(api, hFunc, tx.MerkleProofsAccountLiquidityBefore[i][:], tx.MerkleProofsHelperAccountLiquidityBefore[i][:])
		// verify account nft node hash
		/*
			NftIndex       Variable
			CreatorIndex   Variable
			NftContentHash Variable
			AssetId        Variable
			AssetAmount    Variable
			NftL1Address      Variable
			NftL1TokenId      Variable
		*/
		hFunc.Reset()
		hFunc.Write(
			tx.AccountsInfoBefore[i].NftInfo.NftAssetId,
			tx.AccountsInfoBefore[i].NftInfo.NftIndex,
			tx.AccountsInfoBefore[i].NftInfo.CreatorIndex,
			tx.AccountsInfoBefore[i].NftInfo.NftContentHash,
			tx.AccountsInfoBefore[i].NftInfo.AssetId,
			tx.AccountsInfoBefore[i].NftInfo.AssetAmount,
			tx.AccountsInfoBefore[i].NftInfo.NftL1Address,
			tx.AccountsInfoBefore[i].NftInfo.NftL1TokenId,
		)
		nftNodeHash := hFunc.Sum()
		isNftTxs := api.Or(isDepositNftTx, api.Or(isMintNftTx, api.Or(isTransferNftTx, api.Or(isSetNftPriceTx, api.Or(isBuyNftTx, isWithdrawNftTx)))))
		notNilNftRootAndIsNftTxs := api.And(api.IsZero(api.IsZero(api.Sub(tx.AccountsInfoBefore[i].AccountNftRoot, nilHash))), isNftTxs)
		std.IsVariableEqual(api, notNilNftRootAndIsNftTxs, tx.MerkleProofsAccountNftBefore[i][0], nftNodeHash)
		// verify account nft merkle proof
		hFunc.Reset()
		std.VerifyMerkleProof(
			api,
			notNilNftRootAndIsNftTxs,
			hFunc,
			tx.AccountsInfoBefore[i].AccountNftRoot,
			tx.MerkleProofsAccountNftBefore[i][:],
			tx.MerkleProofsHelperAccountNftBefore[i][:],
		)
		hFunc.Reset()
		hFunc.Write(
			AccountsInfoAfter[i].NftInfo.NftAssetId,
			AccountsInfoAfter[i].NftInfo.NftIndex,
			AccountsInfoAfter[i].NftInfo.CreatorIndex,
			AccountsInfoAfter[i].NftInfo.NftContentHash,
			AccountsInfoAfter[i].NftInfo.AssetId,
			AccountsInfoAfter[i].NftInfo.AssetAmount,
			AccountsInfoAfter[i].NftInfo.NftL1Address,
			AccountsInfoAfter[i].NftInfo.NftL1TokenId,
		)
		nftNodeHash = hFunc.Sum()
		hFunc.Reset()
		// update assetNode hash
		tx.MerkleProofsAccountNftBefore[i][0] = nftNodeHash
		// update merkle proof
		NewAccountNftRoot = std.UpdateMerkleProof(api, hFunc, tx.MerkleProofsAccountNftBefore[i][:], tx.MerkleProofsHelperAccountNftBefore[i][:])
		// verify account node hash
		/*
			BuyerAccountIndex      Variable
			AccountName       Variable
			AccountPk         eddsa.PublicKey
			Nonce             Variable
			StateRoot         Variable
			AccountAssetsRoot Variable
			AccountNftRoot    Variable
		*/
		hFunc.Reset()
		hFunc.Write(
			tx.AccountsInfoBefore[i].AccountIndex,
			tx.AccountsInfoBefore[i].AccountName,
			tx.AccountsInfoBefore[i].AccountPk.A.X,
			tx.AccountsInfoBefore[i].AccountPk.A.Y,
			tx.AccountsInfoBefore[i].Nonce,
			tx.AccountsInfoBefore[i].AccountAssetsRoot,
			tx.AccountsInfoBefore[i].AccountLiquidityRoot,
			tx.AccountsInfoBefore[i].AccountNftRoot,
		)
		accountNodeHash := hFunc.Sum()
		notNilAccountRoot := api.IsZero(api.IsZero(api.Sub(tx.AccountRootBefore, nilHash)))
		std.IsVariableEqual(api, notNilAccountRoot, tx.MerkleProofsAccountBefore[i][0], accountNodeHash)
		// verify account merkle proof
		hFunc.Reset()
		std.VerifyMerkleProof(
			api,
			notNilAccountRoot,
			hFunc,
			tx.AccountRootBefore,
			tx.MerkleProofsAccountBefore[i][:],
			tx.MerkleProofsHelperAccountBefore[i][:],
		)
		hFunc.Reset()
		hFunc.Write(
			AccountsInfoAfter[i].AccountIndex,
			AccountsInfoAfter[i].AccountName,
			AccountsInfoAfter[i].AccountPk.A.X,
			AccountsInfoAfter[i].AccountPk.A.Y,
			AccountsInfoAfter[i].Nonce,
			NewAccountAssetsRoot,
			NewAccountLiquidityRoot,
			NewAccountNftRoot,
		)
		accountNodeHash = hFunc.Sum()
		hFunc.Reset()
		// update account node hash
		tx.MerkleProofsAccountBefore[i][0] = accountNodeHash
		// update merkle proof
		NewAccountRoot = std.UpdateMerkleProof(api, hFunc, tx.MerkleProofsAccountBefore[i][:], tx.MerkleProofsHelperAccountBefore[i][:])
	}

	return nil
}

func SetTxWitness(oTx *Tx) (witness TxConstraints, err error) {
	witness.RegisterZnsTxInfo = std.EmptyRegisterZnsTxWitness()
	witness.DepositTxInfo = std.EmptyDepositTxWitness()
	witness.DepositNftTxInfo = std.EmptyDepositNftTxWitness()
	witness.TransferTxInfo = std.EmptyTransferTxWitness()
	witness.SwapTxInfo = std.EmptySwapTxWitness()
	witness.AddLiquidityTxInfo = std.EmptyAddLiquidityTxWitness()
	witness.RemoveLiquidityTxInfo = std.EmptyRemoveLiquidityTxWitness()
	witness.MintNftTxInfo = std.EmptyMintNftTxWitness()
	witness.TransferNftTxInfo = std.EmptyTransferNftTxWitness()
	witness.SetNftPriceTxInfo = std.EmptySetNftPriceTxWitness()
	witness.BuyNftTxInfo = std.EmptyBuyNftTxWitness()
	witness.WithdrawTxInfo = std.EmptyWithdrawTxWitness()
	witness.WithdrawNftTxInfo = std.EmptyWithdrawNftTxWitness()
	switch oTx.TxType {
	case TxTypeEmptyTx:
		break
	case TxTypeRegisterZns:
		witness.RegisterZnsTxInfo = std.SetRegisterZnsTxWitness(oTx.RegisterZnsTxInfo)
		break
	case TxTypeDeposit:
		witness.DepositTxInfo = std.SetDepositTxWitness(oTx.DepositTxInfo)
		break
	case TxTypeDepositNft:
		witness.DepositNftTxInfo = std.SetDepositNftTxWitness(oTx.DepositNftTxInfo)
		break
	case TxTypeTransfer:
		witness.TransferTxInfo = std.SetTransferTxWitness(oTx.TransferTxInfo)
		break
	case TxTypeSwap:
		witness.SwapTxInfo = std.SetSwapTxWitness(oTx.SwapTxInfo)
		break
	case TxTypeAddLiquidity:
		witness.AddLiquidityTxInfo = std.SetAddLiquidityTxWitness(oTx.AddLiquidityTxInfo)
		break
	case TxTypeRemoveLiquidity:
		witness.RemoveLiquidityTxInfo = std.SetRemoveLiquidityTxWitness(oTx.RemoveLiquidityTxInfo)
		break
	case TxTypeWithdraw:
		witness.WithdrawTxInfo = std.SetWithdrawTxWitness(oTx.WithdrawTxInfo)
		break
	case TxTypeMintNft:
		witness.MintNftTxInfo = std.SetMintNftTxWitness(oTx.MintNftTxInfo)
		break
	case TxTypeTransferNft:
		witness.TransferNftTxInfo = std.SetTransferNftTxWitness(oTx.TransferNftTxInfo)
		break
	case TxTypeSetNftPrice:
		witness.SetNftPriceTxInfo = std.SetSetNftPriceTxWitness(oTx.SetNftPriceTxInfo)
		break
	case TxTypeBuyNft:
		witness.BuyNftTxInfo = std.SetBuyNftTxWitness(oTx.BuyNftTxInfo)
		break
	case TxTypeWithdrawNft:
		witness.WithdrawNftTxInfo = std.SetWithdrawNftTxWitness(oTx.WithdrawNftTxInfo)
		break
	default:
		log.Println("[SetTxWitness] invalid oTx type")
		return witness, errors.New("[SetTxWitness] invalid oTx type")
	}
	// set common account & merkle parts
	// account root before
	witness.AccountRootBefore = oTx.AccountRootBefore
	// account root after
	witness.AccountRootAfter = oTx.AccountRootAfter
	// account before info, size is 4
	for i := 0; i < NbAccountsPerTx; i++ {
		// accounts info before
		witness.AccountsInfoBefore[i], err = std.SetAccountWitness(oTx.AccountsInfoBefore[i])
		if err != nil {
			log.Println("[SetTxWitness] err info:", err)
			return witness, err
		}
		for j := 0; j < NbAccountAssetsPerAccount; j++ {
			for k := 0; k < AssetMerkleLevels; k++ {
				if k != AssetMerkleHelperLevels {
					// account assets before
					witness.MerkleProofsHelperAccountAssetsBefore[i][j][k] = oTx.MerkleProofsHelperAccountAssetsBefore[i][j][k]
					// liquidity asset before
					witness.MerkleProofsHelperAccountLiquidityBefore[i][j] = oTx.MerkleProofsHelperAccountLiquidityBefore[i][j]
				}
				// account assets before
				witness.MerkleProofsAccountAssetsBefore[i][j][k] = oTx.MerkleProofsAccountAssetsBefore[i][j][k]
			}
		}
		for j := 0; j < NftMerkleLevels; j++ {
			if j != NftMerkleHelperLevels {
				// nft assets before
				witness.MerkleProofsHelperAccountNftBefore[i][j] = oTx.MerkleProofsHelperAccountNftBefore[i][j]
			}
			// liquidity asset before
			witness.MerkleProofsAccountLiquidityBefore[i][j] = oTx.MerkleProofsAccountLiquidityBefore[i][j]
			// nft assets before
			witness.MerkleProofsAccountNftBefore[i][j] = oTx.MerkleProofsAccountNftBefore[i][j]
		}
		for j := 0; j < AccountMerkleLevels; j++ {
			if j != AccountMerkleHelperLevels {
				// account before
				witness.MerkleProofsHelperAccountBefore[i][j] = oTx.MerkleProofsHelperAccountBefore[i][j]
			}
			// account before
			witness.MerkleProofsAccountBefore[i][j] = oTx.MerkleProofsAccountBefore[i][j]
		}
	}
	return witness, nil
}
