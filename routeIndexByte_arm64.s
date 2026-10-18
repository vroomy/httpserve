//go:build !purego

#include "textflag.h"

// func routeIndexByteNEON(value string, target byte) int
// The Go wrapper supplies 1..31 bytes whose first byte does not match.
// Every load is within the string, including unaligned strings and tails
// adjacent to an inaccessible page.
TEXT ·routeIndexByteNEON(SB), NOSPLIT, $0-32
	MOVD value_base+0(FP), R0
	MOVD value_len+8(FP), R1
	MOVBU target+16(FP), R2
	CMP $4, R1
	BLO tiny
	VMOV R2, V0.B16
	CMP $16, R1
	BLO short

	MOVD R0, R3
	ADD R0, R1, R4
	SUB $16, R4, R4 // last safe vector start

loop:
	VLD1 (R3), [V1.B16]
	VCMEQ V0.B16, V1.B16, V1.B16
	// Narrow each pair of byte masks to two nibbles, preserving byte order.
	VSHRN $4, V1.H8, V2.B8
	VMOV V2.D[0], R5
	CBNZ R5, found
	CMP R4, R3
	BEQ missing
	MOVD R4, R3 // overlap the previous vector, never overread the tail
	B loop

found:
	RBIT R5, R5
	CLZ R5, R5
	SUB R0, R3, R3
	ADD R5>>2, R3, R5
	MOVD R5, ret+24(FP)
	RET

tiny:
	CMP $1, R1
	BEQ missing
	MOVBU 1(R0), R3
	CMP R2, R3
	BEQ second
	CMP $2, R1
	BEQ missing
	MOVBU 2(R0), R3
	CMP R2, R3
	BNE missing
	MOVD $2, R5
	B result

second:
	MOVD $1, R5
	B result

short:
	// Pack the first and last 8 or 4 bytes into a vector. Overlap is
	// intentional: prefer the first half, then translate a second-half hit
	// back to its original offset. No padding or adjacent memory is read.
	CMP $8, R1
	BLO short4
	MOVD (R0), R5
	SUB $8, R1, R3
	MOVD (R0)(R3), R6
	VMOV R5, V1.D[0]
	VMOV R6, V1.D[1]
	VCMEQ V0.B16, V1.B16, V1.B16
	VSHRN $4, V1.H8, V2.B8
	VMOV V2.D[0], R5
	CBZ R5, missing
	RBIT R5, R5
	CLZ R5, R5
	LSR $2, R5, R5
	CMP $8, R5
	BLO result
	SUB $16, R1, R1
	ADD R1, R5, R5
	B result

short4:
	MOVWU (R0), R5
	SUB $4, R1, R3
	MOVWU (R0)(R3), R6
	ORR R6<<32, R5, R5
	VMOV R5, V1.D[0]
	VCMEQ V0.B8, V1.B8, V1.B8
	VMOV V1.D[0], R5
	CBZ R5, missing
	RBIT R5, R5
	CLZ R5, R5
	LSR $3, R5, R5
	CMP $4, R5
	BLO result
	SUB $8, R1, R1
	ADD R1, R5, R5
	B result

result:
	MOVD R5, ret+24(FP)
	RET

missing:
	MOVD $-1, R5
	MOVD R5, ret+24(FP)
	RET
