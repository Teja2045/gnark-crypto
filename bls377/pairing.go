// Copyright 2020 ConsenSys AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by gurvy/internal/generators DO NOT EDIT

package bls377

// FinalExponentiation computes the final expo x**(p**6-1)(p**2+1)(p**4 - p**2 +1)/r
func (curve *Curve) FinalExponentiation(z *PairingResult, _z ...*PairingResult) PairingResult {
	var result PairingResult
	result.Set(z)

	// if additional parameters are provided, multiply them into z
	for _, e := range _z {
		result.Mul(&result, e)
	}

	result.FinalExponentiation(&result)

	return result
}

// FinalExponentiation sets z to the final expo x**((p**12 - 1)/r), returns z
func (z *PairingResult) FinalExponentiation(x *PairingResult) *PairingResult {
	// For BLS curves use Section 3 of https://eprint.iacr.org/2016/130.pdf; "hard part" is Algorithm 1 of https://eprint.iacr.org/2016/130.pdf
	var result PairingResult
	result.Set(x)

	// memalloc
	var t [6]PairingResult

	// buf = x**(p^6-1)
	t[0].FrobeniusCube(&result).
		FrobeniusCube(&t[0])

	result.Inverse(&result)
	t[0].Mul(&t[0], &result)

	// x = (x**(p^6-1)) ^(p^2+1)
	result.FrobeniusSquare(&t[0]).
		Mul(&result, &t[0])

	// hard part (up to permutation)
	// performs the hard part of the final expo
	// Algorithm 1 of https://eprint.iacr.org/2016/130.pdf
	// The result is the same as p**4-p**2+1/r, but up to permutation (it's 3* (p**4 -p**2 +1 /r)), ok since r=1 mod 3)

	t[0].InverseUnitary(&result).Square(&t[0])
	t[5].Expt(&result)
	t[1].CyclotomicSquare(&t[5])
	t[3].Mul(&t[0], &t[5])

	t[0].Expt(&t[3])
	t[2].Expt(&t[0])
	t[4].Expt(&t[2])

	t[4].Mul(&t[1], &t[4])
	t[1].Expt(&t[4])
	t[3].InverseUnitary(&t[3])
	t[1].Mul(&t[3], &t[1])
	t[1].Mul(&t[1], &result)

	t[0].Mul(&t[0], &result)
	t[0].FrobeniusCube(&t[0])

	t[3].InverseUnitary(&result)
	t[4].Mul(&t[3], &t[4])
	t[4].Frobenius(&t[4])

	t[5].Mul(&t[2], &t[5])
	t[5].FrobeniusSquare(&t[5])

	t[5].Mul(&t[5], &t[0])
	t[5].Mul(&t[5], &t[4])
	t[5].Mul(&t[5], &t[1])

	result.Set(&t[5])

	z.Set(&result)
	return z
}

// MillerLoop Miller loop
func (curve *Curve) MillerLoop(P G1Affine, Q G2Affine, result *PairingResult) *PairingResult {

	// init result
	result.SetOne()

	if P.IsInfinity() || Q.IsInfinity() {
		return result
	}

	// the line goes through QCur and QNext
	var QCur, QNext, QNextNeg G2Jac
	var QNeg G2Affine

	// Stores -Q
	QNeg.Neg(&Q)

	// init QCur with Q
	QCur.FromAffine(&Q)

	var lEval lineEvalRes

	// Miller loop
	for i := len(curve.loopCounter) - 2; i >= 0; i-- {

		QNext.Set(&QCur)
		QNext.DoubleAssign()
		QNextNeg.Neg(&QNext)

		result.Square(result)

		// evaluates line though Qcur,2Qcur at P
		lineEvalJac(QCur, QNextNeg, &P, &lEval)
		lEval.mulAssign(result)

		if curve.loopCounter[i] == 1 {
			// evaluates line through 2Qcur, Q at P
			lineEvalAffine(QNext, Q, &P, &lEval)
			lEval.mulAssign(result)

			QNext.AddMixed(&Q)

		} else if curve.loopCounter[i] == -1 {
			// evaluates line through 2Qcur, -Q at P
			lineEvalAffine(QNext, QNeg, &P, &lEval)
			lEval.mulAssign(result)

			QNext.AddMixed(&QNeg)
		}
		QCur.Set(&QNext)
	}

	return result
}

// lineEval computes the evaluation of the line through Q, R (on the twist) at P
// Q, R are in jacobian coordinates
// The case in which Q=R=Infinity is not handled as this doesn't happen in the SNARK pairing
func lineEvalJac(Q, R G2Jac, P *G1Affine, result *lineEvalRes) {
	// converts _Q and _R to projective coords
	var _Q, _R G2Proj
	_Q.FromJacobian(&Q)
	_R.FromJacobian(&R)

	// line eq: w^3*(_Qy_Rz-_Qz_Ry)x +  w^2*(_Qz_Rx - _Qx_Rz)y + w^5*(_Qx_Ry-_Qy_Rxz)
	// result.r1 = _Qy_Rz-_Qz_Ry
	// result.r0 = _Qz_Rx - _Qx_Rz
	// result.r2 = _Qx_Ry-_Qy_Rxz

	result.r1.Mul(&_Q.Y, &_R.Z)
	result.r0.Mul(&_Q.Z, &_R.X)
	result.r2.Mul(&_Q.X, &_R.Y)

	_Q.Z.Mul(&_Q.Z, &_R.Y)
	_Q.X.Mul(&_Q.X, &_R.Z)
	_Q.Y.Mul(&_Q.Y, &_R.X)

	result.r1.Sub(&result.r1, &_Q.Z)
	result.r0.Sub(&result.r0, &_Q.X)
	result.r2.Sub(&result.r2, &_Q.Y)

	// multiply P.Z by coeffs[2] in case P is infinity
	result.r1.MulByElement(&result.r1, &P.X)
	result.r0.MulByElement(&result.r0, &P.Y)
	//result.r2.MulByElement(&result.r2, &P.Z)
}

// Same as above but R is in affine coords
func lineEvalAffine(Q G2Jac, R G2Affine, P *G1Affine, result *lineEvalRes) {

	var _Q G2Proj
	_Q.FromJacobian(&Q)

	// line eq: w^3*(QyRz-QzRy)x +  w^2*(QzRx - QxRz)y + w^5*(QxRy-QyRxz)
	// result.r1 = QyRz-QzRy
	// result.r0 = QzRx - QxRz
	// result.r2 = QxRy-QyRxz

	result.r1.Set(&_Q.Y)
	result.r0.Mul(&_Q.Z, &R.X)
	result.r2.Mul(&_Q.X, &R.Y)

	_Q.Z.Mul(&_Q.Z, &R.Y)
	_Q.Y.Mul(&_Q.Y, &R.X)

	result.r1.Sub(&result.r1, &_Q.Z)
	result.r0.Sub(&result.r0, &_Q.X)
	result.r2.Sub(&result.r2, &_Q.Y)

	// multiply P.Z by coeffs[2] in case P is infinity
	result.r1.MulByElement(&result.r1, &P.X)
	result.r0.MulByElement(&result.r0, &P.Y)
	// result.r2.MulByElement(&result.r2, &P.Z)
}

type lineEvalRes struct {
	r0 G2CoordType // c0.b1
	r1 G2CoordType // c1.b1
	r2 G2CoordType // c1.b2
}

func (l *lineEvalRes) mulAssign(z *PairingResult) *PairingResult {

	var a, b, c PairingResult
	a.MulByVW(z, &l.r1)
	b.MulByV(z, &l.r0)
	c.MulByV2W(z, &l.r2)
	z.Add(&a, &b).Add(z, &c)

	return z
}

// MulByVW set z to x*(y*v*w) and return z
// here y*v*w means the PairingResult element with C1.B1=y and all other components 0
func (z *PairingResult) MulByVW(x *PairingResult, y *G2CoordType) *PairingResult {
	var result PairingResult
	var yNR G2CoordType
	yNR.MulByNonResidue(y)
	result.C0.B0.Mul(&x.C1.B1, &yNR)
	result.C0.B1.Mul(&x.C1.B2, &yNR)
	result.C0.B2.Mul(&x.C1.B0, y)
	result.C1.B0.Mul(&x.C0.B2, &yNR)
	result.C1.B1.Mul(&x.C0.B0, y)
	result.C1.B2.Mul(&x.C0.B1, y)
	z.Set(&result)
	return z
}

// MulByV set z to x*(y*v) and return z
// here y*v means the PairingResult element with C0.B1=y and all other components 0
func (z *PairingResult) MulByV(x *PairingResult, y *G2CoordType) *PairingResult {
	var result PairingResult
	var yNR G2CoordType
	yNR.MulByNonResidue(y)
	result.C0.B0.Mul(&x.C0.B2, &yNR)
	result.C0.B1.Mul(&x.C0.B0, y)
	result.C0.B2.Mul(&x.C0.B1, y)
	result.C1.B0.Mul(&x.C1.B2, &yNR)
	result.C1.B1.Mul(&x.C1.B0, y)
	result.C1.B2.Mul(&x.C1.B1, y)
	z.Set(&result)
	return z
}

// MulByV2W set z to x*(y*v^2*w) and return z
// here y*v^2*w means the PairingResult element with C1.B2=y and all other components 0
func (z *PairingResult) MulByV2W(x *PairingResult, y *G2CoordType) *PairingResult {
	var result PairingResult
	var yNR G2CoordType
	yNR.MulByNonResidue(y)
	result.C0.B0.Mul(&x.C1.B0, &yNR)
	result.C0.B1.Mul(&x.C1.B1, &yNR)
	result.C0.B2.Mul(&x.C1.B2, &yNR)
	result.C1.B0.Mul(&x.C0.B1, &yNR)
	result.C1.B1.Mul(&x.C0.B2, &yNR)
	result.C1.B2.Mul(&x.C0.B0, y)
	z.Set(&result)
	return z
}

const tAbsVal uint64 = 9586122913090633729

// Expt set z to x^t in PairingResult and return z
func (z *PairingResult) Expt(x *PairingResult) *PairingResult {

	// tAbsVal in binary: 1000010100001000110000000000000000000000000000000000000000000001
	// drop the low 46 bits (all 0 except the least significant bit): 100001010000100011 = 136227
	// Shortest addition chains can be found at https://wwwhomes.uni-bielefeld.de/achim/addition_chain.html

	var result, x33 PairingResult

	// a shortest addition chain for 136227
	result.Set(x)                    // 0                1
	result.CyclotomicSquare(&result) // 1( 0)            2
	result.CyclotomicSquare(&result) // 2( 1)            4
	result.CyclotomicSquare(&result) // 3( 2)            8
	result.CyclotomicSquare(&result) // 4( 3)           16
	result.CyclotomicSquare(&result) // 5( 4)           32
	result.Mul(&result, x)           // 6( 5, 0)        33
	x33.Set(&result)                 // save x33 for step 14
	result.CyclotomicSquare(&result) // 7( 6)           66
	result.CyclotomicSquare(&result) // 8( 7)          132
	result.CyclotomicSquare(&result) // 9( 8)          264
	result.CyclotomicSquare(&result) // 10( 9)          528
	result.CyclotomicSquare(&result) // 11(10)         1056
	result.CyclotomicSquare(&result) // 12(11)         2112
	result.CyclotomicSquare(&result) // 13(12)         4224
	result.Mul(&result, &x33)        // 14(13, 6)      4257
	result.CyclotomicSquare(&result) // 15(14)         8514
	result.CyclotomicSquare(&result) // 16(15)        17028
	result.CyclotomicSquare(&result) // 17(16)        34056
	result.CyclotomicSquare(&result) // 18(17)        68112
	result.Mul(&result, x)           // 19(18, 0)     68113
	result.CyclotomicSquare(&result) // 20(19)       136226
	result.Mul(&result, x)           // 21(20, 0)    136227

	// the remaining 46 bits
	for i := 0; i < 46; i++ {
		result.CyclotomicSquare(&result)
	}
	result.Mul(&result, x)

	z.Set(&result)
	return z
}
