// Licensed under the MIT license.
// Licensed under the MIT license; see LICENSE.txt.

// Package g3 is the in-progress Go translation of this repository's patched
// Z3 4.7.1 solver. Source components remain unimplemented unless recorded in
// port_manifest.json; no solver API or satisfiability result is available yet.
// Independent utilities do not import this package. Internal handles use slots
// in release builds and slot/generation pairs under the g3checked build tag.
package g3
