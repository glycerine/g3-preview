package pdr_test

// Source: src/muz/pdr/pdr_reachable_cache.cpp cache modes and symbolic
// disjunction-chain reachability checks.

import (
	"testing"

	"github.com/glycerine/g3/pkg/muz/pdr"
)

func TestReachableCacheModes(t *testing.T) {
	for _, mode := range []pdr.ReachableCacheMode{pdr.ReachableNoCache, pdr.ReachableHashCache, pdr.ReachableConstraintCache} {
		t.Run(map[pdr.ReachableCacheMode]string{pdr.ReachableNoCache: "none", pdr.ReachableHashCache: "hash", pdr.ReachableConstraintCache: "constraint"}[mode], func(t *testing.T) {
			command, manager, contexts := pdrContextFixture(t, 0)
			truth, err := manager.MkTrue()
			pdrMust(t, err)
			cache, err := pdr.NewReachableCache(manager, contexts, truth, mode)
			pdrMust(t, err)
			p := pdrBool(t, manager, "reachable-p")
			q := pdrBool(t, manager, "reachable-q")
			notP, err := manager.MkNot(p)
			pdrMust(t, err)
			if found, x := cache.IsReachable(p); x != nil || found {
				t.Fatal("empty cache", found, x)
			}
			pdrMust(t, cache.AddReachable(p))
			found, err := cache.IsReachable(p)
			pdrMust(t, err)
			if found != (mode != pdr.ReachableNoCache) {
				t.Fatal("inserted cube", mode, found)
			}
			found, err = cache.IsReachable(notP)
			pdrMust(t, err)
			if found {
				t.Fatal("incompatible cube reported reachable")
			}
			pdrMust(t, cache.AddReachable(q))
			found, err = cache.IsReachable(q)
			pdrMust(t, err)
			if found != (mode != pdr.ReachableNoCache) {
				t.Fatal("second cube", mode, found)
			}
			stats, err := cache.Statistics()
			pdrMust(t, err)
			if mode != pdr.ReachableNoCache && (stats.Inserts != 2 || stats.Hits != 2 || stats.Misses != 2) {
				t.Fatal("cache statistics", stats)
			}
			pdrMust(t, cache.ResetStatistics())
			stats, err = cache.Statistics()
			pdrMust(t, err)
			if stats != (pdr.ReachableCacheStatistics{}) {
				t.Fatal("statistics reset", stats)
			}
			pdrMust(t, cache.Close())
			pdrMust(t, contexts.Close())
			pdrMust(t, command.Close())
		})
	}
}

func TestReachableConstraintCacheHonorsBackgroundAndInit(t *testing.T) {
	command, manager, contexts := pdrContextFixture(t, 1)
	p := pdrBool(t, manager, "reachable-background-p")
	q := pdrBool(t, manager, "reachable-background-q")
	cache, err := pdr.NewReachableCache(manager, contexts, p, pdr.ReachableConstraintCache)
	pdrMust(t, err)
	pdrMust(t, cache.AddInit(q))
	conjunction, err := manager.MkAnd(p, q)
	pdrMust(t, err)
	found, err := cache.IsReachable(conjunction)
	pdrMust(t, err)
	if !found {
		t.Fatal("background-compatible init state was not reachable")
	}
	notP, err := manager.MkNot(p)
	pdrMust(t, err)
	found, err = cache.IsReachable(notP)
	pdrMust(t, err)
	if found {
		t.Fatal("background-incompatible state was reachable")
	}
	pdrMust(t, cache.Close())
	pdrMust(t, contexts.Close())
	pdrMust(t, command.Close())
}
