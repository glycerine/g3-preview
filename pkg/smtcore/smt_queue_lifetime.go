// Source: src/smt/smt_context.cpp:44-92,135-138,3281-3293,3367-3394;
// src/smt/smt_case_split_queue.cpp mk_case_split_queue and borrowed m_params.
package smtcore

func effectiveSMTCaseSplitStrategy(params SMTParamsState) uint32 {
	strategy := params.CaseSplitStrategy
	if strategy >= 3 && strategy <= 5 && (params.RelevancyLvl < 2 || params.AutoConfig) {
		return 0
	}
	return strategy
}

// Kernel contexts construct their queue once. Live updates change parameters,
// while the installed concrete queue class and phase-cache counters survive.
func (k *SMTKernel) configureQueueHeuristics(params SMTParamsState, setup bool) error {
	queue := k.caseSplitQueue
	initial := queue == nil
	strategy := effectiveSMTCaseSplitStrategy(params)
	if queue != nil {
		strategy = queue.strategy
		if queue.liveParams != nil {
			if err := queue.liveParams.open(); err != nil {
				return err
			}
			params = queue.liveParams.state
		}
	}
	if err := k.internalizer.configureBooleanHeuristicsForQueue(params, strategy, initial || setup); err != nil {
		return err
	}
	if initial {
		var err error
		queue, err = NewSMTActivityCaseSplitQueue(k.internalizer)
		if err != nil {
			return err
		}
		if err = queue.ConfigurePhases(params); err != nil {
			return err
		}
		k.caseSplitQueue = queue
	} else if err := queue.updateLivePhases(&params); err != nil {
		return err
	}
	queue.liveParams = k.smtParams
	return nil
}

func (q *SMTActivityCaseSplitQueue) refreshLivePhases() error {
	if q.liveParams == nil {
		return nil
	}
	if err := q.liveParams.open(); err != nil {
		return err
	}
	return q.updateLivePhases(&q.liveParams.state)
}
