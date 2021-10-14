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

package std

import "github.com/consensys/gnark/std/algebra/twistededwards"

/*
	ElGamalEncConstraints describes ElGamal Enc in circuit
*/
type ElGamalEncConstraints struct {
	CL Point // Pk^r
	CR Point // g^r Waste^b
}

func negElgamal(cs *ConstraintSystem, C ElGamalEncConstraints) ElGamalEncConstraints {
	return ElGamalEncConstraints{
		CL: *C.CL.Neg(cs, &C.CL),
		CR: *C.CR.Neg(cs, &C.CR),
	}
}

func enc(cs *ConstraintSystem, h Point, b Variable, r Variable, pk Point, params twistededwards.EdCurve) ElGamalEncConstraints {
	var CL, gr, CR Point
	CL.ScalarMulNonFixedBase(cs, &pk, r, params)
	gr.ScalarMulFixedBase(cs, params.BaseX, params.BaseY, r, params)
	CR.ScalarMulNonFixedBase(cs, &h, b, params)
	CR.AddGeneric(cs, &CR, &gr, params)
	return ElGamalEncConstraints{CL: CL, CR: CR}
}

func encAdd(cs *ConstraintSystem, C, CDelta ElGamalEncConstraints, params twistededwards.EdCurve) ElGamalEncConstraints {
	C.CL.AddGeneric(cs, &C.CL, &CDelta.CL, params)
	C.CR.AddGeneric(cs, &C.CR, &CDelta.CR, params)
	return C
}

func encSub(cs *ConstraintSystem, C, CDelta ElGamalEncConstraints, params twistededwards.EdCurve) ElGamalEncConstraints {
	var CL, CR Point
	CL.AddGeneric(cs, &C.CL, CDelta.CL.Neg(cs, &CDelta.CL), params)
	CR.AddGeneric(cs, &C.CR, CDelta.CR.Neg(cs, &CDelta.CR), params)
	return ElGamalEncConstraints{CL: CL, CR: CR}
}

func zeroElgamal(cs *ConstraintSystem) ElGamalEncConstraints {
	return ElGamalEncConstraints{CL: zeroPoint(cs), CR: zeroPoint(cs)}
}

func selectElgamal(cs *ConstraintSystem, flag Variable, a, b ElGamalEncConstraints) ElGamalEncConstraints {
	CLX := cs.Select(flag, a.CL.X, b.CL.X)
	CLY := cs.Select(flag, a.CL.Y, b.CL.Y)
	CRX := cs.Select(flag, a.CR.X, b.CR.X)
	CRY := cs.Select(flag, a.CR.Y, b.CR.Y)
	return ElGamalEncConstraints{CL: Point{X: CLX, Y: CLY}, CR: Point{X: CRX, Y: CRY}}
}

func printEnc(cs *ConstraintSystem, a ElGamalEncConstraints) {
	cs.Println(a.CL.X)
	cs.Println(a.CL.Y)
	cs.Println(a.CR.X)
	cs.Println(a.CR.Y)
}
