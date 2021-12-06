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

package transactions

import (
	"errors"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/algebra/twistededwards"
	"github.com/consensys/gnark/std/hash/mimc"
	"log"
	"zecrey-crypto/hash/bn254/zmimc"
	"zecrey-crypto/zecrey/circuit/bn254/std"
)

type TxConstraints struct {
	// tx type
	TxType Variable
	// deposit info
	DepositTxInfo DepositOrLockTxConstraints
	// lock info
	LockTxInfo DepositOrLockTxConstraints
	// unlock proof
	UnlockProof UnlockProofConstraints
	// transfer proof
	TransferProof TransferProofConstraints
	// swap proof
	SwapProof SwapProofConstraints
	// add liquidity proof
	AddLiquidityProof AddLiquidityProofConstraints
	// remove liquidity proof
	RemoveLiquidityProof RemoveLiquidityProofConstraints
	// withdraw proof
	WithdrawProof WithdrawProofConstraints
	// common verification part
	// range proofs
	RangeProofs [MaxRangeProofCount]CtRangeProofConstraints
	// account root before
	AccountRootBefore Variable
	// account before info, size is 4
	AccountsInfoBefore [NbAccountsPerTx]AccountConstraints
	// before account merkle proof
	MerkleProofsAccountBefore       [NbAccountsPerTx][]Variable
	MerkleProofsHelperAccountBefore [NbAccountsPerTx][]Variable
	// before account asset merkle proof
	MerkleProofsAccountAssetsBefore       [NbAccountsPerTx][][]Variable
	MerkleProofsHelperAccountAssetsBefore [NbAccountsPerTx][][]Variable
	// before account asset lock merkle proof
	MerkleProofsAccountLockedAssetsBefore       [NbAccountsPerTx][]Variable
	MerkleProofsHelperAccountLockedAssetsBefore [NbAccountsPerTx][]Variable
	// before account liquidity merkle proof
	MerkleProofsAccountLiquidityBefore       [NbAccountsPerTx][]Variable
	MerkleProofsHelperAccountLiquidityBefore [NbAccountsPerTx][]Variable
	// account root after
	AccountRootAfter Variable
	// account after info, size is 4
	AccountsInfoAfter [NbAccountsPerTx]AccountConstraints
	// after account merkle proof
	MerkleProofsAccountAfter       [NbAccountsPerTx][]Variable
	MerkleProofsHelperAccountAfter [NbAccountsPerTx][]Variable
	// after account asset merkle proof
	MerkleProofsAccountAssetsAfter       [NbAccountsPerTx][]Variable
	MerkleProofsHelperAccountAssetsAfter [NbAccountsPerTx][]Variable
	// after account asset lock merkle proof
	MerkleProofsAccountLockedAssetsAfter       [NbAccountsPerTx][]Variable
	MerkleProofsHelperAccountLockedAssetsAfter [NbAccountsPerTx][]Variable
	// after account liquidity merkle proof
	MerkleProofsAccountLiquidityAfter       [NbAccountsPerTx][]Variable
	MerkleProofsHelperAccountLiquidityAfter [NbAccountsPerTx][]Variable
}

func (circuit TxConstraints) Define(curveID ecc.ID, api frontend.API) error {
	// get edwards curve params
	params, err := twistededwards.NewEdCurve(curveID)
	if err != nil {
		return err
	}

	// mimc
	hFunc, err := mimc.NewMiMC(zmimc.SEED, curveID, api)
	if err != nil {
		return err
	}

	// TODO verify H: need to optimize
	H := Point{
		X: api.Constant(std.HX),
		Y: api.Constant(std.HY),
	}
	tool := std.NewEccTool(api, params)
	VerifyTransaction(tool, api, circuit, hFunc, H)

	return nil
}

func VerifyTransaction(
	tool *std.EccTool,
	api API,
	tx TxConstraints,
	hFunc MiMC,
	h Point,
) {
	// txType constants
	txTypeNoop := api.Constant(uint64(TxTypeNoop))
	txTypeDeposit := api.Constant(uint64(TxTypeDeposit))
	txTypeLock := api.Constant(uint64(TxTypeLock))
	txTypeUnlock := api.Constant(uint64(TxTypeUnlock))
	txTypeTransfer := api.Constant(uint64(TxTypeTransfer))
	txTypeSwap := api.Constant(uint64(TxTypeSwap))
	txTypeAddLiquidity := api.Constant(uint64(TxTypeAddLiquidity))
	txTypeRemoveLiquidity := api.Constant(uint64(TxTypeRemoveLiquidity))
	txTypeWithdraw := api.Constant(uint64(TxTypeWithdraw))

	// compute tx type
	isNoopTx := api.IsZero(api.Sub(tx.TxType, txTypeNoop))
	isDepositTx := api.IsZero(api.Sub(tx.TxType, txTypeDeposit))
	tx.DepositTxInfo.IsEnabled = isDepositTx
	isLockTx := api.IsZero(api.Sub(tx.TxType, txTypeLock))
	tx.LockTxInfo.IsEnabled = isLockTx
	isUnlockTx := api.IsZero(api.Sub(tx.TxType, txTypeUnlock))
	tx.UnlockProof.IsEnabled = isUnlockTx
	isTransferTx := api.IsZero(api.Sub(tx.TxType, txTypeTransfer))
	tx.TransferProof.IsEnabled = isTransferTx
	isSwapTx := api.IsZero(api.Sub(tx.TxType, txTypeSwap))
	tx.SwapProof.IsEnabled = isSwapTx
	isAddLiquidityTx := api.IsZero(api.Sub(tx.TxType, txTypeAddLiquidity))
	tx.AddLiquidityProof.IsEnabled = isAddLiquidityTx
	isRemoveLiquidityTx := api.IsZero(api.Sub(tx.TxType, txTypeRemoveLiquidity))
	tx.RemoveLiquidityProof.IsEnabled = isRemoveLiquidityTx
	isWithdrawTx := api.IsZero(api.Sub(tx.TxType, txTypeWithdraw))
	tx.WithdrawProof.IsEnabled = isWithdrawTx

	isCheckAccount := api.IsZero(isNoopTx)
	// verify range proofs
	for i, rangeProof := range tx.RangeProofs {
		// set range proof is true
		isNoRangeTx := api.Or(isDepositTx, isLockTx)
		isEnabled := api.IsZero(isNoRangeTx)
		rangeProof.IsEnabled = isEnabled
		std.VerifyCtRangeProof(tool, api, rangeProof, hFunc)
		hFunc.Reset()
		tx.TransferProof.SubProofs[i].Y = rangeProof.A
	}
	// set T or Y
	// unlock proof
	tx.UnlockProof.T_fee = tx.RangeProofs[0].A
	// transfer proof
	for i := 0; i < NbTransferCount; i++ {
		tx.TransferProof.SubProofs[i].T = tx.RangeProofs[i].A
	}
	// swap proof
	tx.SwapProof.T_uA = tx.RangeProofs[0].A
	tx.SwapProof.T_fee = tx.RangeProofs[1].A
	// add liquidity proof
	tx.AddLiquidityProof.T_uA = tx.RangeProofs[0].A
	tx.AddLiquidityProof.T_uB = tx.RangeProofs[1].A
	tx.AddLiquidityProof.T_fee = tx.RangeProofs[2].A
	// remove liquidity proof
	tx.RemoveLiquidityProof.T_uLP = tx.RangeProofs[0].A
	tx.RemoveLiquidityProof.T_fee = tx.RangeProofs[1].A
	// withdraw proof
	tx.WithdrawProof.T = tx.RangeProofs[0].A
	tx.WithdrawProof.T_fee = tx.RangeProofs[1].A

	// verify account before
	for i := 0; i < NbAccountsPerTx; i++ {
		// verify accounts before & after params
		std.IsVariableEqual(api, isCheckAccount, tx.AccountsInfoBefore[i].AccountIndex, tx.AccountsInfoAfter[i].AccountIndex)
		std.IsVariableEqual(api, isCheckAccount, tx.AccountsInfoBefore[i].AccountName, tx.AccountsInfoAfter[i].AccountName)
		std.IsPointEqual(api, isCheckAccount, tx.AccountsInfoBefore[i].AccountPk, tx.AccountsInfoAfter[i].AccountPk)
		// check state root
		hFunc.Write(
			tx.AccountsInfoBefore[i].AccountAssetsRoot,
			tx.AccountsInfoBefore[i].AccountLockedAssetsRoot,
			tx.AccountsInfoBefore[i].AccountLiquidityRoot,
		)
		stateRootCheck := hFunc.Sum()
		std.IsVariableEqual(api, isCheckAccount, stateRootCheck, tx.AccountsInfoBefore[i].StateRoot)
		hFunc.Reset()
		// check account hash
		hFunc.Write(
			tx.AccountsInfoBefore[i].AccountIndex,
			tx.AccountsInfoBefore[i].AccountName,
		)
		std.WritePointIntoBuf(&hFunc, tx.AccountsInfoBefore[i].AccountPk)
		hFunc.Write(tx.AccountsInfoBefore[i].StateRoot)
		accountHashCheck := hFunc.Sum()
		std.IsVariableEqual(api, isCheckAccount, accountHashCheck, tx.MerkleProofsAccountBefore[i][0])
		hFunc.Reset()
		// verify account asset root
		for j := 0; j < NbAccountAssetsPerAccount; j++ {
			std.VerifyMerkleProof(
				api, isCheckAccount, hFunc,
				tx.AccountsInfoBefore[i].AccountAssetsRoot,
				tx.MerkleProofsAccountAssetsBefore[i][j], tx.MerkleProofsHelperAccountAssetsBefore[i][j])
			hFunc.Reset()
			// verify account asset before & after params
			std.IsVariableEqual(
				api, isCheckAccount,
				tx.AccountsInfoBefore[i].AssetsInfo[j].AssetId, tx.AccountsInfoAfter[i].AssetsInfo[j].AssetId)
		}
		// verify account locked asset root
		std.VerifyMerkleProof(
			api, isCheckAccount, hFunc,
			tx.AccountsInfoBefore[i].AccountLockedAssetsRoot,
			tx.MerkleProofsAccountLockedAssetsBefore[i], tx.MerkleProofsHelperAccountLockedAssetsBefore[i])
		hFunc.Reset()
		// verify account locked asset before & after params
		std.IsVariableEqual(
			api, isCheckAccount,
			tx.AccountsInfoBefore[i].LockedAssetInfo.ChainId, tx.AccountsInfoAfter[i].LockedAssetInfo.ChainId)
		std.IsVariableEqual(
			api, isCheckAccount,
			tx.AccountsInfoBefore[i].LockedAssetInfo.AssetId, tx.AccountsInfoAfter[i].LockedAssetInfo.AssetId)
		// verify account liquidity root
		std.VerifyMerkleProof(
			api, isCheckAccount, hFunc,
			tx.AccountsInfoBefore[i].AccountLiquidityRoot,
			tx.MerkleProofsAccountLiquidityBefore[i], tx.MerkleProofsHelperAccountLiquidityBefore[i])
		hFunc.Reset()
		// verify account liquidity before & after params
		std.IsVariableEqual(
			api, isCheckAccount,
			tx.AccountsInfoBefore[i].LiquidityInfo.PairIndex, tx.AccountsInfoAfter[i].LiquidityInfo.PairIndex)
		// verify account root
		std.VerifyMerkleProof(
			api, isCheckAccount, hFunc,
			tx.AccountRootBefore,
			tx.MerkleProofsAccountBefore[i], tx.MerkleProofsHelperAccountBefore[i])
		hFunc.Reset()
	}

	// verify proofs
	var (
		c, cCheck               Variable
		pkProofs, pkProofsCheck [MaxRangeProofCount]std.CommonPkProof
		tProofs, tProofsCheck   [MaxRangeProofCount]std.CommonTProof
	)

	// verify unlock proof
	// set public data
	// set account info
	tx.UnlockProof.ChainId = tx.AccountsInfoBefore[0].LockedAssetInfo.ChainId
	tx.UnlockProof.AssetId = tx.AccountsInfoBefore[0].LockedAssetInfo.AssetId
	tx.UnlockProof.Pk = tx.AccountsInfoBefore[0].AccountPk
	tx.UnlockProof.Balance = tx.AccountsInfoBefore[0].LockedAssetInfo.LockedAmount
	// fee info
	tx.UnlockProof.C_fee = tx.AccountsInfoBefore[0].AssetsInfo[1].BalanceEnc
	tx.UnlockProof.GasFeeAssetId = tx.AccountsInfoBefore[0].AssetsInfo[1].AssetId
	c, pkProofs, tProofs = std.VerifyUnlockProof(tool, api, tx.UnlockProof, hFunc, h)
	hFunc.Reset()

	// verify transfer proof
	// set public data
	// set account info
	tx.TransferProof.AssetId = tx.AccountsInfoBefore[0].AssetsInfo[0].AssetId
	for i := 0; i < NbTransferCount; i++ {
		tx.TransferProof.SubProofs[i].Pk = tx.AccountsInfoBefore[i].AccountPk
		tx.TransferProof.SubProofs[i].C = tx.AccountsInfoBefore[i].AssetsInfo[0].BalanceEnc
	}
	cCheck, pkProofsCheck, tProofsCheck = std.VerifyTransferProof(tool, api, tx.TransferProof, hFunc, h)
	hFunc.Reset()
	c, pkProofs, tProofs = SelectCommonPart(api, isTransferTx, cCheck, c, pkProofsCheck, pkProofs, tProofsCheck, tProofs)

	// verify swap proof
	// set public data
	// set account info
	tx.SwapProof.C_uA = tx.AccountsInfoBefore[0].AssetsInfo[0].BalanceEnc
	tx.SwapProof.Pk_u = tx.AccountsInfoBefore[0].AccountPk
	tx.SwapProof.AssetAId = tx.AccountsInfoBefore[1].LiquidityInfo.AssetAId
	tx.SwapProof.AssetBId = tx.AccountsInfoBefore[1].LiquidityInfo.AssetBId
	tx.SwapProof.Pk_pool = tx.AccountsInfoBefore[1].AccountPk
	tx.SwapProof.B_poolA = tx.AccountsInfoBefore[1].LiquidityInfo.AssetA
	tx.SwapProof.B_poolB = tx.AccountsInfoBefore[1].LiquidityInfo.AssetB
	tx.SwapProof.Pk_treasury = tx.AccountsInfoBefore[2].AccountPk
	// fee info
	tx.SwapProof.C_fee = tx.AccountsInfoBefore[0].AssetsInfo[2].BalanceEnc
	tx.SwapProof.GasFeeAssetId = tx.AccountsInfoBefore[0].AssetsInfo[2].AssetId
	cCheck, pkProofsCheck, tProofsCheck = std.VerifySwapProof(tool, api, tx.SwapProof, hFunc, h)
	hFunc.Reset()
	c, pkProofs, tProofs = SelectCommonPart(api, isSwapTx, cCheck, c, pkProofsCheck, pkProofs, tProofsCheck, tProofs)

	// verify add liquidity proof
	// set public data
	// set account info
	tx.AddLiquidityProof.C_uA = tx.AccountsInfoBefore[0].AssetsInfo[0].BalanceEnc
	tx.AddLiquidityProof.C_uB = tx.AccountsInfoBefore[0].AssetsInfo[1].BalanceEnc
	tx.AddLiquidityProof.Pk_u = tx.AccountsInfoBefore[0].AccountPk
	tx.AddLiquidityProof.AssetAId = tx.AccountsInfoBefore[1].LiquidityInfo.AssetAId
	tx.AddLiquidityProof.AssetBId = tx.AccountsInfoBefore[1].LiquidityInfo.AssetBId
	tx.AddLiquidityProof.Pk_pool = tx.AccountsInfoBefore[1].AccountPk
	tx.AddLiquidityProof.B_poolA = tx.AccountsInfoBefore[1].LiquidityInfo.AssetA
	tx.AddLiquidityProof.B_poolB = tx.AccountsInfoBefore[1].LiquidityInfo.AssetB
	// fee info
	tx.AddLiquidityProof.C_fee = tx.AccountsInfoBefore[0].AssetsInfo[2].BalanceEnc
	tx.AddLiquidityProof.GasFeeAssetId = tx.AccountsInfoBefore[0].AssetsInfo[2].AssetId
	cCheck, pkProofsCheck, tProofsCheck = std.VerifyAddLiquidityProof(tool, api, tx.AddLiquidityProof, hFunc, h)
	hFunc.Reset()
	c, pkProofs, tProofs = SelectCommonPart(api, isAddLiquidityTx, cCheck, c, pkProofsCheck, pkProofs, tProofsCheck, tProofs)

	// verify remove liquidity proof
	// set public data
	// set account info
	tx.RemoveLiquidityProof.C_u_LP = tx.AccountsInfoBefore[0].LiquidityInfo.LpEnc
	tx.RemoveLiquidityProof.Pk_u = tx.AccountsInfoBefore[0].AccountPk
	tx.RemoveLiquidityProof.AssetAId = tx.AccountsInfoBefore[1].LiquidityInfo.AssetAId
	tx.RemoveLiquidityProof.AssetBId = tx.AccountsInfoBefore[1].LiquidityInfo.AssetBId
	tx.RemoveLiquidityProof.Pk_pool = tx.AccountsInfoBefore[1].AccountPk
	tx.RemoveLiquidityProof.B_pool_A = tx.AccountsInfoBefore[1].LiquidityInfo.AssetA
	tx.RemoveLiquidityProof.B_pool_B = tx.AccountsInfoBefore[1].LiquidityInfo.AssetB
	// fee info
	tx.RemoveLiquidityProof.C_fee = tx.AccountsInfoBefore[0].AssetsInfo[2].BalanceEnc
	tx.RemoveLiquidityProof.GasFeeAssetId = tx.AccountsInfoBefore[0].AssetsInfo[2].AssetId
	cCheck, pkProofsCheck, tProofsCheck = std.VerifyRemoveLiquidityProof(tool, api, tx.RemoveLiquidityProof, hFunc, h)
	hFunc.Reset()
	c, pkProofs, tProofs = SelectCommonPart(api, isRemoveLiquidityTx, cCheck, c, pkProofsCheck, pkProofs, tProofsCheck, tProofs)

	// verify withdraw proof
	// set public data
	// set account info
	tx.WithdrawProof.AssetId = tx.AccountsInfoBefore[0].AssetsInfo[0].AssetId
	tx.WithdrawProof.Pk = tx.AccountsInfoBefore[0].AccountPk
	// fee info
	tx.WithdrawProof.C_fee = tx.AccountsInfoBefore[0].AssetsInfo[1].BalanceEnc
	tx.WithdrawProof.GasFeeAssetId = tx.AccountsInfoBefore[0].AssetsInfo[1].AssetId
	cCheck, pkProofsCheck, tProofsCheck = std.VerifyWithdrawProof(tool, api, tx.WithdrawProof, hFunc, h)
	hFunc.Reset()
	c, pkProofs, tProofs = SelectCommonPart(api, isWithdrawTx, cCheck, c, pkProofsCheck, pkProofs, tProofsCheck, tProofs)
	enabled := api.Constant(1)
	for i := 0; i < MaxRangeProofCount; i++ {
		// pk proof
		l1 := tool.ScalarBaseMul(pkProofs[i].Z_sk_u)
		r1 := tool.Add(pkProofs[i].A_pk_u, tool.ScalarMul(pkProofs[i].Pk_u, c))
		std.IsPointEqual(api, enabled, l1, r1)
		// T proof
		// Verify T(C_R - C_R^{\star})^{-1} = (C_L - C_L^{\star})^{-sk^{-1}} g^{\bar{r}}
		l2 := tool.Add(tool.ScalarBaseMul(tProofs[i].Z_bar_r), tool.ScalarMul(tProofs[i].C_PrimeNeg.CL, pkProofs[i].Z_sk_uInv))
		r2 := tool.Add(tProofs[i].A_T_C_RPrimeInv, tool.ScalarMul(tool.Add(tProofs[i].T, tProofs[i].C_PrimeNeg.CR), c))
		std.IsPointEqual(api, enabled, l2, r2)
	}

	// check if the after account info is correct


	// verify account after
	for i := 0; i < NbAccountsPerTx; i++ {
		// check state root
		hFunc.Write(
			tx.AccountsInfoAfter[i].AccountAssetsRoot,
			tx.AccountsInfoAfter[i].AccountLockedAssetsRoot,
			tx.AccountsInfoAfter[i].AccountLiquidityRoot,
		)
		stateRootCheck := hFunc.Sum()
		std.IsVariableEqual(api, isCheckAccount, stateRootCheck, tx.AccountsInfoAfter[i].StateRoot)
		hFunc.Reset()
		// check account hash
		hFunc.Write(
			tx.AccountsInfoAfter[i].AccountIndex,
			tx.AccountsInfoAfter[i].AccountName,
		)
		std.WritePointIntoBuf(&hFunc, tx.AccountsInfoAfter[i].AccountPk)
		hFunc.Write(tx.AccountsInfoAfter[i].StateRoot)
		accountHashCheck := hFunc.Sum()
		std.IsVariableEqual(api, isCheckAccount, accountHashCheck, tx.MerkleProofsAccountAfter[i][0])
		hFunc.Reset()
		// verify account asset root
		std.VerifyMerkleProof(
			api, isCheckAccount, hFunc,
			tx.AccountsInfoAfter[i].AccountAssetsRoot,
			tx.MerkleProofsAccountAssetsAfter[i], tx.MerkleProofsHelperAccountAssetsAfter[i])
		hFunc.Reset()
		// verify account locked asset root
		std.VerifyMerkleProof(
			api, isCheckAccount, hFunc,
			tx.AccountsInfoAfter[i].AccountLockedAssetsRoot,
			tx.MerkleProofsAccountLockedAssetsAfter[i], tx.MerkleProofsHelperAccountLockedAssetsAfter[i])
		hFunc.Reset()
		// verify account liquidity root
		std.VerifyMerkleProof(
			api, isCheckAccount, hFunc,
			tx.AccountsInfoAfter[i].AccountLiquidityRoot,
			tx.MerkleProofsAccountLiquidityAfter[i], tx.MerkleProofsHelperAccountLiquidityAfter[i])
		hFunc.Reset()
		// verify account root
		std.VerifyMerkleProof(
			api, isCheckAccount, hFunc,
			tx.AccountRootAfter,
			tx.MerkleProofsAccountAfter[i], tx.MerkleProofsHelperAccountAfter[i])
		hFunc.Reset()
	}

}

func SetTxWitness(oTx *Tx) (witness TxConstraints, err error) {
	oproof := oTx.OProof
	txType := oTx.TxType
	isEnabled := true
	switch txType {
	case TxTypeNoop:
		// set witness
		witness.TxType.Assign(uint64(txType))
		witness.DepositTxInfo = std.SetEmptyDepositOrLockWitness()
		witness.LockTxInfo = std.SetEmptyDepositOrLockWitness()
		witness.UnlockProof = std.SetEmptyUnlockProofWitness()
		witness.TransferProof = std.SetEmptyTransferProofWitness()
		witness.SwapProof = std.SetEmptySwapProofWitness()
		witness.AddLiquidityProof = std.SetEmptyAddLiquidityProofWitness()
		witness.RemoveLiquidityProof = std.SetEmptyRemoveLiquidityProofWitness()
		witness.WithdrawProof = std.SetEmptyWithdrawProofWitness()
		witness.RangeProofs[0] = std.SetEmptyCtRangeProofWitness()
		witness.RangeProofs[1] = std.SetEmptyCtRangeProofWitness()
		witness.RangeProofs[2] = std.SetEmptyCtRangeProofWitness()
		break
	case TxTypeDeposit:
		// convert to special proof
		tx, b := oproof.(*DepositOrLockTx)
		if !b {
			log.Println("[SetTxWitness] unable to convert proof to special type")
			return witness, errors.New("[SetTxWitness] unable to convert proof to special type")
		}
		// set witness
		witness.TxType.Assign(uint64(txType))
		witness.DepositTxInfo, err = std.SetDepositOrLockWitness(tx, isEnabled)
		if err != nil {
			return witness, err
		}
		witness.UnlockProof = std.SetEmptyUnlockProofWitness()
		witness.TransferProof = std.SetEmptyTransferProofWitness()
		witness.SwapProof = std.SetEmptySwapProofWitness()
		witness.AddLiquidityProof = std.SetEmptyAddLiquidityProofWitness()
		witness.RemoveLiquidityProof = std.SetEmptyRemoveLiquidityProofWitness()
		witness.WithdrawProof = std.SetEmptyWithdrawProofWitness()
		witness.RangeProofs[0] = std.SetEmptyCtRangeProofWitness()
		witness.RangeProofs[1] = std.SetEmptyCtRangeProofWitness()
		witness.RangeProofs[2] = std.SetEmptyCtRangeProofWitness()
		break
	case TxTypeLock:
		// convert to special proof
		tx, b := oproof.(*DepositOrLockTx)
		if !b {
			log.Println("[SetTxWitness] unable to convert proof to special type")
			return witness, errors.New("[SetTxWitness] unable to convert proof to special type")
		}
		// set witness
		witness.TxType.Assign(uint64(txType))
		witness.DepositTxInfo = std.SetEmptyDepositOrLockWitness()
		witness.LockTxInfo, err = std.SetDepositOrLockWitness(tx, isEnabled)
		if err != nil {
			return witness, err
		}
		witness.UnlockProof = std.SetEmptyUnlockProofWitness()
		witness.TransferProof = std.SetEmptyTransferProofWitness()
		witness.SwapProof = std.SetEmptySwapProofWitness()
		witness.AddLiquidityProof = std.SetEmptyAddLiquidityProofWitness()
		witness.RemoveLiquidityProof = std.SetEmptyRemoveLiquidityProofWitness()
		witness.WithdrawProof = std.SetEmptyWithdrawProofWitness()
		witness.RangeProofs[0] = std.SetEmptyCtRangeProofWitness()
		witness.RangeProofs[1] = std.SetEmptyCtRangeProofWitness()
		witness.RangeProofs[2] = std.SetEmptyCtRangeProofWitness()
		break
	case TxTypeTransfer:
		// convert to special proof
		proof, b := oproof.(*TransferProof)
		if !b {
			log.Println("[SetTxWitness] unable to convert proof to special type")
			return witness, errors.New("[SetTxWitness] unable to convert proof to special type")
		}
		proofConstraints, err := std.SetTransferProofWitness(proof, isEnabled)
		if err != nil {
			return witness, err
		}
		// set witness
		witness.TxType.Assign(uint64(txType))
		witness.DepositTxInfo = std.SetEmptyDepositOrLockWitness()
		witness.LockTxInfo = std.SetEmptyDepositOrLockWitness()
		witness.UnlockProof = std.SetEmptyUnlockProofWitness()
		witness.TransferProof = proofConstraints
		for i, subProof := range proof.SubProofs {
			witness.RangeProofs[i], err = std.SetCtRangeProofWitness(subProof.BStarRangeProof, isEnabled)
			if err != nil {
				return witness, err
			}
		}
		witness.SwapProof = std.SetEmptySwapProofWitness()
		witness.AddLiquidityProof = std.SetEmptyAddLiquidityProofWitness()
		witness.RemoveLiquidityProof = std.SetEmptyRemoveLiquidityProofWitness()
		witness.WithdrawProof = std.SetEmptyWithdrawProofWitness()
		break
	case TxTypeSwap:
		proof, b := oproof.(*SwapProof)
		if !b {
			return witness, errors.New("[SetTxWitness] unable to convert proof to special type")
		}
		proofConstraints, err := std.SetSwapProofWitness(proof, isEnabled)
		if err != nil {
			return witness, err
		}
		// set witness
		witness.TxType.Assign(uint64(txType))
		witness.DepositTxInfo = std.SetEmptyDepositOrLockWitness()
		witness.LockTxInfo = std.SetEmptyDepositOrLockWitness()
		witness.UnlockProof = std.SetEmptyUnlockProofWitness()
		witness.TransferProof = std.SetEmptyTransferProofWitness()
		witness.SwapProof = proofConstraints
		witness.RangeProofs[0], err = std.SetCtRangeProofWitness(proof.ARangeProof, isEnabled)
		if err != nil {
			return witness, err
		}
		witness.RangeProofs[1], err = std.SetCtRangeProofWitness(proof.GasFeePrimeRangeProof, isEnabled)
		if err != nil {
			return witness, err
		}
		witness.RangeProofs[2] = witness.RangeProofs[1]
		witness.AddLiquidityProof = std.SetEmptyAddLiquidityProofWitness()
		witness.RemoveLiquidityProof = std.SetEmptyRemoveLiquidityProofWitness()
		witness.WithdrawProof = std.SetEmptyWithdrawProofWitness()
		break
	case TxTypeAddLiquidity:
		proof, b := oproof.(*AddLiquidityProof)
		if !b {
			return witness, errors.New("[SetTxWitness] unable to convert proof to special type")
		}
		proofConstraints, err := std.SetAddLiquidityProofWitness(proof, isEnabled)
		if err != nil {
			return witness, err
		}
		// set witness
		witness.TxType.Assign(uint64(txType))
		witness.DepositTxInfo = std.SetEmptyDepositOrLockWitness()
		witness.LockTxInfo = std.SetEmptyDepositOrLockWitness()
		witness.UnlockProof = std.SetEmptyUnlockProofWitness()
		witness.TransferProof = std.SetEmptyTransferProofWitness()
		witness.SwapProof = std.SetEmptySwapProofWitness()
		witness.AddLiquidityProof = proofConstraints
		witness.RangeProofs[0], err = std.SetCtRangeProofWitness(proof.ARangeProof, isEnabled)
		if err != nil {
			return witness, err
		}
		witness.RangeProofs[1], err = std.SetCtRangeProofWitness(proof.BRangeProof, isEnabled)
		if err != nil {
			return witness, err
		}
		witness.RangeProofs[2], err = std.SetCtRangeProofWitness(proof.GasFeePrimeRangeProof, isEnabled)
		if err != nil {
			return witness, err
		}
		witness.RemoveLiquidityProof = std.SetEmptyRemoveLiquidityProofWitness()
		witness.WithdrawProof = std.SetEmptyWithdrawProofWitness()
		break
	case TxTypeRemoveLiquidity:
		proof, b := oproof.(*RemoveLiquidityProof)
		if !b {
			return witness, errors.New("[SetTxWitness] unable to convert proof to special type")
		}
		proofConstraints, err := std.SetRemoveLiquidityProofWitness(proof, isEnabled)
		if err != nil {
			return witness, err
		}
		// set witness
		witness.TxType.Assign(uint64(txType))
		witness.DepositTxInfo = std.SetEmptyDepositOrLockWitness()
		witness.LockTxInfo = std.SetEmptyDepositOrLockWitness()
		witness.UnlockProof = std.SetEmptyUnlockProofWitness()
		witness.TransferProof = std.SetEmptyTransferProofWitness()
		witness.SwapProof = std.SetEmptySwapProofWitness()
		witness.AddLiquidityProof = std.SetEmptyAddLiquidityProofWitness()
		witness.RemoveLiquidityProof = proofConstraints
		witness.RangeProofs[0], err = std.SetCtRangeProofWitness(proof.LPRangeProof, isEnabled)
		if err != nil {
			return witness, err
		}
		witness.RangeProofs[1], err = std.SetCtRangeProofWitness(proof.GasFeePrimeRangeProof, isEnabled)
		if err != nil {
			return witness, err
		}
		witness.RangeProofs[2] = witness.RangeProofs[0]
		witness.WithdrawProof = std.SetEmptyWithdrawProofWitness()
		break
	case TxTypeUnlock:
		proof, b := oproof.(*UnlockProof)
		if !b {
			return witness, errors.New("[SetTxWitness] unable to convert proof to special type")
		}
		proofConstraints, err := std.SetUnlockProofWitness(proof, isEnabled)
		if err != nil {
			return witness, err
		}
		// set witness
		witness.TxType.Assign(uint64(txType))
		witness.DepositTxInfo = std.SetEmptyDepositOrLockWitness()
		witness.LockTxInfo = std.SetEmptyDepositOrLockWitness()
		witness.UnlockProof = proofConstraints
		witness.TransferProof = std.SetEmptyTransferProofWitness()
		witness.SwapProof = std.SetEmptySwapProofWitness()
		witness.AddLiquidityProof = std.SetEmptyAddLiquidityProofWitness()
		witness.RemoveLiquidityProof = std.SetEmptyRemoveLiquidityProofWitness()
		witness.WithdrawProof = std.SetEmptyWithdrawProofWitness()
		witness.RangeProofs[0], err = std.SetCtRangeProofWitness(proof.GasFeePrimeRangeProof, isEnabled)
		if err != nil {
			return witness, err
		}
		witness.RangeProofs[1] = witness.RangeProofs[0]
		witness.RangeProofs[2] = witness.RangeProofs[0]
		break
	case TxTypeWithdraw:
		proof, b := oproof.(*WithdrawProof)
		if !b {
			return witness, errors.New("[SetTxWitness] unable to convert proof to special type")
		}
		proofConstraints, err := std.SetWithdrawProofWitness(proof, isEnabled)
		if err != nil {
			return witness, err
		}
		// set witness
		witness.TxType.Assign(uint64(txType))
		witness.DepositTxInfo = std.SetEmptyDepositOrLockWitness()
		witness.LockTxInfo = std.SetEmptyDepositOrLockWitness()
		witness.UnlockProof = std.SetEmptyUnlockProofWitness()
		witness.TransferProof = std.SetEmptyTransferProofWitness()
		witness.SwapProof = std.SetEmptySwapProofWitness()
		witness.AddLiquidityProof = std.SetEmptyAddLiquidityProofWitness()
		witness.RemoveLiquidityProof = std.SetEmptyRemoveLiquidityProofWitness()
		witness.WithdrawProof = proofConstraints
		witness.RangeProofs[0], err = std.SetCtRangeProofWitness(proof.BPrimeRangeProof, isEnabled)
		if err != nil {
			return witness, err
		}
		witness.RangeProofs[1], err = std.SetCtRangeProofWitness(proof.GasFeePrimeRangeProof, isEnabled)
		if err != nil {
			return witness, err
		}
		witness.RangeProofs[2] = witness.RangeProofs[0]
		break
	default:
		log.Println("[SetTxWitness] invalid tx type")
		return witness, errors.New("[SetTxWitness] invalid tx type")
	}
	// set common account & merkle parts
	// account root before
	witness.AccountRootBefore.Assign(oTx.AccountRootBefore)
	// account before info, size is 4
	for i := 0; i < NbAccountsPerTx; i++ {
		witness.AccountsInfoBefore[i], err = SetAccountWitness(oTx.AccountsInfoBefore[i])
		if err != nil {
			log.Println("[SetTxWitness] err info:", err)
			return witness, err
		}
	}
	// before account asset merkle proof
	for i := 0; i < NbAccountsPerTx; i++ {
		for j := 0; j < NbAccountAssetsPerAccount; j++ {
			witness.MerkleProofsAccountAssetsBefore[i][j] =
				std.SetMerkleProofsWitness(oTx.MerkleProofsAccountAssetsBefore[i][j][:], AssetMerkleLevels)
			witness.MerkleProofsHelperAccountAssetsBefore[i][j] =
				std.SetMerkleProofsHelperWitness(oTx.MerkleProofsHelperAccountAssetsBefore[i][j][:], AssetMerkleHelperLevels)
		}
	}
	// before account asset lock merkle proof
	for i := 0; i < NbAccountsPerTx; i++ {
		witness.MerkleProofsAccountLockedAssetsBefore[i] =
			std.SetMerkleProofsWitness(oTx.MerkleProofsAccountLockedAssetsBefore[i][:], LockedAssetMerkleLevels)
		witness.MerkleProofsHelperAccountLockedAssetsBefore[i] =
			std.SetMerkleProofsHelperWitness(oTx.MerkleProofsHelperAccountLockedAssetsBefore[i][:], LockedAssetMerkleHelperLevels)
	}
	// before account liquidity merkle proof
	for i := 0; i < NbAccountsPerTx; i++ {
		witness.MerkleProofsAccountLiquidityBefore[i] =
			std.SetMerkleProofsWitness(oTx.MerkleProofsAccountLiquidityBefore[i][:], LiquidityMerkleLevels)
		witness.MerkleProofsHelperAccountLiquidityBefore[i] =
			std.SetMerkleProofsHelperWitness(oTx.MerkleProofsHelperAccountLiquidityBefore[i][:], LiquidityMerkleHelperLevels)
	}
	// before account merkle proof
	for i := 0; i < NbAccountsPerTx; i++ {
		witness.MerkleProofsAccountBefore[i] =
			std.SetMerkleProofsWitness(oTx.MerkleProofsAccountBefore[i][:], AccountMerkleLevels)
		witness.MerkleProofsHelperAccountBefore[i] =
			std.SetMerkleProofsHelperWitness(oTx.MerkleProofsHelperAccountBefore[i][:], AccountMerkleHelperLevels)
	}
	// account root after
	witness.AccountRootAfter.Assign(oTx.AccountRootAfter)
	// account after info, size is 4
	for i := 0; i < NbAccountsPerTx; i++ {
		witness.AccountsInfoAfter[i], err = SetAccountWitness(oTx.AccountsInfoAfter[i])
		if err != nil {
			log.Println("[SetTxWitness] err info:", err)
			return witness, err
		}
	}
	// after account asset merkle proof
	for i := 0; i < NbAccountsPerTx; i++ {
		witness.MerkleProofsAccountAssetsAfter[i] =
			std.SetMerkleProofsWitness(oTx.MerkleProofsAccountAssetsAfter[i][:], AssetMerkleLevels)
		witness.MerkleProofsHelperAccountAssetsAfter[i] =
			std.SetMerkleProofsHelperWitness(oTx.MerkleProofsHelperAccountAssetsAfter[i][:], AssetMerkleHelperLevels)
	}
	// after account asset lock merkle proof
	for i := 0; i < NbAccountsPerTx; i++ {
		witness.MerkleProofsAccountLockedAssetsAfter[i] =
			std.SetMerkleProofsWitness(oTx.MerkleProofsAccountLockedAssetsAfter[i][:], LockedAssetMerkleLevels)
		witness.MerkleProofsHelperAccountLockedAssetsAfter[i] =
			std.SetMerkleProofsHelperWitness(oTx.MerkleProofsHelperAccountLockedAssetsAfter[i][:], LockedAssetMerkleHelperLevels)
	}
	// after account liquidity merkle proof
	for i := 0; i < NbAccountsPerTx; i++ {
		witness.MerkleProofsAccountLiquidityAfter[i] =
			std.SetMerkleProofsWitness(oTx.MerkleProofsAccountLiquidityAfter[i][:], LiquidityMerkleLevels)
		witness.MerkleProofsHelperAccountLiquidityAfter[i] =
			std.SetMerkleProofsHelperWitness(oTx.MerkleProofsHelperAccountLiquidityAfter[i][:], LiquidityMerkleHelperLevels)
	}
	// after account merkle proof
	for i := 0; i < NbAccountsPerTx; i++ {
		witness.MerkleProofsAccountAfter[i] =
			std.SetMerkleProofsWitness(oTx.MerkleProofsAccountAfter[i][:], AccountMerkleLevels)
		witness.MerkleProofsHelperAccountAfter[i] =
			std.SetMerkleProofsHelperWitness(oTx.MerkleProofsHelperAccountAfter[i][:], AccountMerkleHelperLevels)
	}
	return witness, nil
}
