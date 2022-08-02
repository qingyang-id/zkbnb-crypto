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

package legendTxTypes

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"log"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
)

type WithdrawNftSegmentFormat struct {
	AccountIndex      int64  `json:"account_index"`
	NftIndex          int64  `json:"nft_index"`
	ToAddress         string `json:"to_address"`
	GasAccountIndex   int64  `json:"gas_account_index"`
	GasFeeAssetId     int64  `json:"gas_fee_asset_id"`
	GasFeeAssetAmount string `json:"gas_fee_asset_amount"`
	ExpiredAt         int64  `json:"expired_at"`
	Nonce             int64  `json:"nonce"`
}

func ConstructWithdrawNftTxInfo(sk *PrivateKey, segmentStr string) (txInfo *WithdrawNftTxInfo, err error) {
	var segmentFormat *WithdrawNftSegmentFormat
	err = json.Unmarshal([]byte(segmentStr), &segmentFormat)
	if err != nil {
		log.Println("[ConstructWithdrawNftTxInfo] err info:", err)
		return nil, err
	}
	gasFeeAmount, err := StringToBigInt(segmentFormat.GasFeeAssetAmount)
	if err != nil {
		log.Println("[ConstructBuyNftTxInfo] unable to convert string to big int:", err)
		return nil, err
	}
	gasFeeAmount, _ = CleanPackedFee(gasFeeAmount)
	txInfo = &WithdrawNftTxInfo{
		AccountIndex:      segmentFormat.AccountIndex,
		NftIndex:          segmentFormat.NftIndex,
		ToAddress:         segmentFormat.ToAddress,
		GasAccountIndex:   segmentFormat.GasAccountIndex,
		GasFeeAssetId:     segmentFormat.GasFeeAssetId,
		GasFeeAssetAmount: gasFeeAmount,
		ExpiredAt:         segmentFormat.ExpiredAt,
		Nonce:             segmentFormat.Nonce,
		Sig:               nil,
	}
	// compute call data hash
	hFunc := mimc.NewMiMC()
	// compute msg hash
	msgHash, err := ComputeWithdrawNftMsgHash(txInfo, hFunc)
	if err != nil {
		log.Println("[ConstructWithdrawNftTxInfo] unable to compute hash:", err)
		return nil, err
	}
	// compute signature
	hFunc.Reset()
	sigBytes, err := sk.Sign(msgHash, hFunc)
	if err != nil {
		log.Println("[ConstructWithdrawNftTxInfo] unable to sign:", err)
		return nil, err
	}
	txInfo.Sig = sigBytes
	return txInfo, nil
}

type WithdrawNftTxInfo struct {
	AccountIndex           int64
	CreatorAccountIndex    int64
	CreatorAccountNameHash []byte
	CreatorTreasuryRate    int64
	NftIndex               int64
	NftContentHash         []byte
	NftL1Address           string
	NftL1TokenId           *big.Int
	CollectionId           int64
	ToAddress              string
	GasAccountIndex        int64
	GasFeeAssetId          int64
	GasFeeAssetAmount      *big.Int
	ExpiredAt              int64
	Nonce                  int64
	Sig                    []byte
}

func ValidateWithdrawNftTxInfo(txInfo *WithdrawNftTxInfo) error {
	// AccountIndex
	if txInfo.AccountIndex < minAccountIndex {
		return fmt.Errorf("AccountIndex should not be less than %d", minAccountIndex)
	}
	if txInfo.AccountIndex > maxAccountIndex {
		return fmt.Errorf("AccountIndex should not be larger than %d", maxAccountIndex)
	}

	// CreatorAccountIndex
	if txInfo.CreatorAccountIndex < minAccountIndex {
		return fmt.Errorf("CreatorAccountIndex should not be less than %d", minAccountIndex)
	}
	if txInfo.CreatorAccountIndex > maxAccountIndex {
		return fmt.Errorf("CreatorAccountIndex should not be larger than %d", maxAccountIndex)
	}

	// CreatorAccountNameHash
	if !IsValidHashBytes(txInfo.CreatorAccountNameHash) {
		return fmt.Errorf("CreatorAccountNameHash(%s) is invalid", hex.EncodeToString(txInfo.CreatorAccountNameHash))
	}

	// CreatorTreasuryRate
	if txInfo.CreatorTreasuryRate < minTreasuryRate {
		return fmt.Errorf("CreatorTreasuryRate should  not be less than %d", minTreasuryRate)
	}
	if txInfo.CreatorTreasuryRate > maxTreasuryRate {
		return fmt.Errorf("CreatorTreasuryRate should not be larger than %d", maxTreasuryRate)
	}

	// NftIndex
	if txInfo.NftIndex < minNftIndex {
		return fmt.Errorf("NftIndex should not be less than %d", minNftIndex)
	}
	if txInfo.NftIndex > maxNftIndex {
		return fmt.Errorf("NftIndex should not be larger than %d", maxNftIndex)
	}

	// NftContentHash
	if !IsValidHashBytes(txInfo.NftContentHash) {
		return fmt.Errorf("NftContentHash(%s) is invalid", hex.EncodeToString(txInfo.NftContentHash))
	}

	// NftL1Address
	if txInfo.NftL1Address != "" && !IsValidL1Address(txInfo.NftL1Address) {
		return fmt.Errorf("NftL1Address(%s) is invalid", txInfo.NftL1Address)
	}

	// NftL1TokenId
	if txInfo.NftL1TokenId != nil && txInfo.NftL1TokenId.Cmp(big.NewInt(0)) < 0 {
		return fmt.Errorf("NftL1TokenId should not be less than 0")
	}

	// CollectionId
	if txInfo.CollectionId < minCollectionId {
		return fmt.Errorf("CollectionId should not be less than %d", minCollectionId)
	}
	if txInfo.CollectionId > maxCollectionId {
		return fmt.Errorf("CollectionId should not be larger than %d", maxCollectionId)
	}

	// ToAddress
	if !IsValidL1Address(txInfo.ToAddress) {
		return fmt.Errorf("ToAddress(%s) is invalid", txInfo.ToAddress)
	}

	// GasAccountIndex
	if txInfo.GasAccountIndex < minAccountIndex {
		return fmt.Errorf("GasAccountIndex should not be less than %d", minAccountIndex)
	}
	if txInfo.GasAccountIndex > maxAccountIndex {
		return fmt.Errorf("GasAccountIndex should not be larger than %d", maxAccountIndex)
	}

	// GasFeeAssetId
	if txInfo.GasFeeAssetId < minAssetId {
		return fmt.Errorf("GasFeeAssetId should not be less than %d", minAssetId)
	}
	if txInfo.GasFeeAssetId > maxAssetId {
		return fmt.Errorf("GasFeeAssetId should not be larger than %d", maxAssetId)
	}

	// GasFeeAssetAmount
	if txInfo.GasFeeAssetAmount == nil {
		return fmt.Errorf("GasFeeAssetAmount should not be nil")
	}
	if txInfo.GasFeeAssetAmount.Cmp(minPackedFeeAmount) < 0 {
		return fmt.Errorf("GasFeeAssetAmount should not be less than %s", minPackedFeeAmount.String())
	}
	if txInfo.GasFeeAssetAmount.Cmp(maxPackedFeeAmount) > 0 {
		return fmt.Errorf("GasFeeAssetAmount should not be larger than %s", maxPackedFeeAmount.String())
	}

	// ExpiredAt
	if txInfo.ExpiredAt <= 0 {
		return fmt.Errorf("ExpiredAt should be larger than 0")
	}

	// Nonce
	if txInfo.Nonce < minNonce {
		return fmt.Errorf("Nonce should not be less than %d", minNonce)
	}

	return nil
}

func ComputeWithdrawNftMsgHash(txInfo *WithdrawNftTxInfo, hFunc hash.Hash) (msgHash []byte, err error) {
	hFunc.Reset()
	var buf bytes.Buffer
	packedFee, err := ToPackedFee(txInfo.GasFeeAssetAmount)
	if err != nil {
		log.Println("[ComputeTransferMsgHash] unable to packed amount", err.Error())
		return nil, err
	}
	WriteInt64IntoBuf(&buf, txInfo.AccountIndex)
	WriteInt64IntoBuf(&buf, txInfo.NftIndex)
	buf.Write(PaddingAddressToBytes32(txInfo.ToAddress))
	WriteInt64IntoBuf(&buf, txInfo.GasAccountIndex)
	WriteInt64IntoBuf(&buf, txInfo.GasFeeAssetId)
	WriteInt64IntoBuf(&buf, packedFee)
	WriteInt64IntoBuf(&buf, txInfo.ExpiredAt)
	WriteInt64IntoBuf(&buf, txInfo.Nonce)
	WriteInt64IntoBuf(&buf, ChainId)
	hFunc.Write(buf.Bytes())
	msgHash = hFunc.Sum(nil)
	return msgHash, nil
}
