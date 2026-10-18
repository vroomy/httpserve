//go:build !purego

#include "textflag.h"

// func routeIndexByteSSE2(value string, target byte) int
// The Go wrapper supplies 1..31 bytes whose first byte does not match.
// Every load is within the string. Only baseline AMD64 instructions are used;
// SSE2 needs no runtime feature detection or higher GOAMD64 level.
TEXT ·routeIndexByteSSE2(SB), NOSPLIT, $0-32
	MOVQ value_base+0(FP), SI
	MOVQ value_len+8(FP), BX
	MOVBLZX target+16(FP), AX
	CMPQ BX, $4
	JB tiny
	IMULL $0x01010101, AX
	MOVD AX, X0
	PSHUFL $0, X0, X0
	CMPQ BX, $16
	JB short

	MOVOU (SI), X1
	PCMPEQB X0, X1
	PMOVMSKB X1, DX
	BSFL DX, DX
	JNZ result
	CMPQ BX, $16
	JE missing
	// Overlap the first vector with the last safe 16-byte load.
	MOVOU -16(SI)(BX*1), X1
	PCMPEQB X0, X1
	PMOVMSKB X1, DX
	BSFL DX, DX
	JZ missing
	LEAQ -16(BX)(DX*1), DX
	JMP result

tiny:
	CMPQ BX, $1
	JE missing
	CMPB 1(SI), AL
	JE second
	CMPQ BX, $2
	JE missing
	CMPB 2(SI), AL
	JNE missing
	MOVQ $2, DX
	JMP result

second:
	MOVQ $1, DX
	JMP result

short:
	// Pack the first and last 8 or 4 bytes into a vector. Prefer a match
	// in the first half, then translate a second-half hit to its offset.
	CMPQ BX, $8
	JB short4
	MOVQ (SI), X1
	MOVQ -8(SI)(BX*1), X2
	PUNPCKLQDQ X2, X1
	PCMPEQB X0, X1
	PMOVMSKB X1, DX
	BSFL DX, DX
	JZ missing
	CMPL DX, $8
	JB result
	LEAQ -16(BX)(DX*1), DX
	JMP result

short4:
	MOVL (SI), CX
	MOVL -4(SI)(BX*1), DX
	SHLQ $32, DX
	ORQ CX, DX
	MOVQ DX, X1
	PCMPEQB X0, X1
	PMOVMSKB X1, DX
	// The upper eight bytes were not loaded and must not match a NUL target.
	ANDL $0xff, DX
	BSFL DX, DX
	JZ missing
	CMPL DX, $4
	JB result
	LEAQ -8(BX)(DX*1), DX

result:
	MOVQ DX, ret+24(FP)
	RET

missing:
	MOVQ $-1, ret+24(FP)
	RET
