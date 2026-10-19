// Package apex provides HTTP routing, middleware, request contexts, and HTTP/HTTPS
// servers for the Vroomy ecosystem and standalone Go applications.
//
// Routes support static paths, named parameters, and wildcards. Warmed dispatch
// reuses pooled contexts and parameter storage. On ARM64 and AMD64, selected
// routing searches use SIMD, with a portable fallback for other architectures
// and purego builds.
package apex
