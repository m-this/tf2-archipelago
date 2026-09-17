#pragma once
/* Force-included into every source when building a SourceMod extension for
 * Windows with clang-cl. It fixes two places where hl2sdk-tf2 was written for
 * Microsoft's compiler and clang, even in MSVC mode, disagrees.
 *
 * Both are fixed by including the real header first, adjusted, and letting its
 * include guard turn every later include of it into nothing. That works however
 * the source reaches the header, <platform.h>, <tier0/platform.h> or a quoted
 * relative path, which a shim placed on the include path does not: it only
 * catches the one spelling it is named after.
 *
 * RESTRICT. bitbuf.h declares bf_write::WriteUBitLong without RESTRICT and
 * defines it with one. MSVC lets that pass; clang reads __restrict on a member
 * function as part of its type and calls the two a conflict. It is an optimiser
 * hint about `this` and nothing else, so it is dropped.
 *
 * m128_f32. ssemath.h reads an __m128 lane as a.m128_f32[i], a union member
 * Microsoft's compiler puts on the type and clang does not: __m128 is a builtin
 * vector there with no members. Its own POSIX branch does the same job with a
 * reinterpret_cast. POSIX appears four times in that file and all four are those
 * accessors, so it is defined for that one include and put back, because
 * everywhere else in the SDK POSIX means "not Windows". What ssemath.h includes
 * is included first, without it, so threadtools.h keeps its Windows branch.
 *
 * The SigMod port hit the same two in the same headers.
 */

#include <tier0/platform.h>
#undef RESTRICT
#define RESTRICT

#include <mathlib/vector.h>
#include <mathlib/mathlib.h>

#ifndef POSIX
#define POSIX 1
#include <mathlib/ssemath.h>
#undef POSIX
#else
#include <mathlib/ssemath.h>
#endif
