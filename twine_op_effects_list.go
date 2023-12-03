package etxt

import "fmt"

// Tricky cases:
// - During a double pass effect first measuring pass, we can find effects
//   that are closed before reaching a line break or a final pop. We soft
//   pop these effects, but then if a new effect is added, the head needs
//   to go *after* the soft popped effect. This means that between tail and
//   head we can find "holes" with soft popped events, and that the active
//   head can jump multiple places at once (so, we need to transform that
//   using a for loop).

// Related to twineOperator.
//
// Why do we need a sophisticated structure instead of a simple []effectOperationData?
// Well... for double passes we need to keep the effect content widths stored, and in
// fact we can optimize by not recalculating everything again, but simply keeping the
// effectOperationData memorized and just recalling it at the right time. But this also
// creates other problems as we go back and forth, and it can be hard to keep track
// of the "head" of the list, some effect pop operations could create "holes" in the
// slice, etc. So, we have a full fledged struct instead and provide the relevant
// operations in a more structured manner.
type twineOperatorEffectsList struct {
	activeHeadIndex uint16 // newest *active* effect index
	rawHeadIndex uint16 // newest effect index (it can be soft-popped)
	tailIndex uint16 // oldest active effect index
	freeIndex uint16 // 65535 if none. necessary due to the pops that happen while drawing
	                 // double pass effects, where we might have to hard pop some effects
	                 // while other soft popped effects remain ahead of us
	activeCount uint16
	activeDoublePassEffectCount uint16
	totalCount uint16
	list []effectOperationData
}

func (self *twineOperatorEffectsList) debugStr() string {
	return fmt.Sprintf(
		"effectsList{ activeHeadIndex: %d, rawHeadIndex: %d, tailIndex: %d, " + 
		"freeIndex: %d, activeCount: %d, dpCount: %d, totalCount: %d, len(list) = %d }",
		self.activeHeadIndex, self.rawHeadIndex, self.tailIndex, self.freeIndex,
		self.activeCount, self.activeDoublePassEffectCount, self.totalCount, len(self.list),
	)
}

func (self *twineOperatorEffectsList) Initialize() {
	self.rawHeadIndex    = 65535
	self.activeHeadIndex = 65535
	self.tailIndex       = 65535
	self.freeIndex       = 65535
	if cap(self.list) < 8 {
		self.list = make([]effectOperationData, 0, 8)
	} else {
		self.list = self.list[ : 0]
	}

	self.totalCount = 0
	self.activeCount = 0
	self.activeDoublePassEffectCount = 0
}

func (self *twineOperatorEffectsList) ActiveCount() uint16 {
	return self.activeCount
}

func (self *twineOperatorEffectsList) ActiveDoublePassEffectsCount() uint16 {
	return self.activeDoublePassEffectCount
}

func (self *twineOperatorEffectsList) OnLastDoublePassEffect() bool {
	return (
		self.activeDoublePassEffectCount == 1 &&
		self.activeHeadIndex != 65535 &&
		self.list[self.activeHeadIndex].mode == DoublePass)
}

func (self *twineOperatorEffectsList) AssertAllEffectsActive() {
	if self.activeCount == self.totalCount { return }
	panic("twineOperatorEffectsList.AssertAllEffectsActive() failure")
}

func (self *twineOperatorEffectsList) Head() *effectOperationData {
	if self.activeHeadIndex == 65535 { return nil }
	return &self.list[self.activeHeadIndex]
}

// Only active effects will be reported. The pointer can't be stored / kept.
func (self *twineOperatorEffectsList) Each(fn func(*effectOperationData)) {
	remainingActiveEffects := self.activeCount
	var index uint16 = self.tailIndex
	for remainingActiveEffects > 0 {
		effect := &self.list[index]
		index = effect.linkNext
		if effect.softPopped { continue }
		remainingActiveEffects -= 1
		fn(effect)
	}
}

func (self *twineOperatorEffectsList) EachReverse(fn func(*effectOperationData)) {
	remainingActiveEffects := self.activeCount
	var index uint16 = self.activeHeadIndex
	for remainingActiveEffects > 0 {
		effect := &self.list[index]
		index = effect.linkPrev
		if effect.softPopped { continue }
		remainingActiveEffects -= 1
		fn(effect)
	}
}

func (self *twineOperatorEffectsList) Push(effectValue effectOperationData) {
	// get new index
	var newHeadIndex uint16
	if self.freeIndex != 65535 {
		newHeadIndex = self.freeIndex
		self.freeIndex = self.list[newHeadIndex].linkPrev
		self.list[newHeadIndex] = effectValue
	} else {
		if len(self.list) >= 64000 { panic("max number of nested effects in twine (64k) exceeded") }
		self.list = append(self.list, effectValue)
		newHeadIndex = uint16(len(self.list) - 1)
	}

	// store effect and set relevant fields for linking
	effect := &self.list[newHeadIndex]
	effect.linkPrev = self.rawHeadIndex
	effect.linkNext = 65535 // mark "no next"
	effect.softPopped = false
	if self.rawHeadIndex != 65535 {
		self.list[self.rawHeadIndex].linkNext = newHeadIndex
	}
	if self.tailIndex == 65535 {
		self.tailIndex = newHeadIndex
	}
	self.rawHeadIndex = newHeadIndex
	self.activeHeadIndex = newHeadIndex
	self.totalCount += 1
	self.increaseActiveCount(effect)
}

// On double passes, we don't need to Push() effects that we previously hit
// already, we only need to recall them. If this doesn't work, then you should
// Push() manually. The returned value will be nil if no recall is possible.
// The returned value can't be stored, only used temporarily before calling
// the next twineOperatorEffectsList method.
func (self *twineOperatorEffectsList) TryRecallNext() *effectOperationData {
	var recallIndex uint16
	if self.activeHeadIndex != 65535 {
		recallIndex = self.list[self.activeHeadIndex].linkNext
		if recallIndex == 65535 { return nil }
	} else {
		if self.totalCount == 0 { return nil }
		if self.tailIndex == 65535 { panic("broken code") }
		recallIndex = self.tailIndex
	}

	effect := &self.list[recallIndex]
	if !effect.softPopped { panic("broken code") } // TODO: delete later
	effect.softPopped = false
	self.increaseActiveCount(effect)
	self.activeHeadIndex = recallIndex
	return effect
}

func (self *twineOperatorEffectsList) SoftPop() *effectOperationData {
	if self.activeHeadIndex == 65535 { panic("no effect left to pop") }
	effectIndex := self.activeHeadIndex
	effect := &self.list[effectIndex]
	if effect.softPopped { panic("broken code") } // TODO: delete later
	effect.softPopped = true
	self.decreaseActiveCount(effect)
	self.refreshActiveHeadIndexFromPop(effect)
	return effect
}

// If you want to do something with an effect before it's hard popped, you
// should use twineOperatorEffectsList.Head() to get the effect.
func (self *twineOperatorEffectsList) HardPop() {
	if self.activeHeadIndex == 65535 { panic("no effect left to pop") }
	effectIndex := self.activeHeadIndex
	effect := &self.list[effectIndex]
	if effect.softPopped { panic("broken code") } // TODO: delete later
	if self.totalCount == 0 { panic("broken code") }
	self.totalCount -= 1
	self.decreaseActiveCount(effect)
	self.refreshActiveHeadIndexFromPop(effect)

	// readjust linking to head, tail and between effect nodes
	if effect.linkPrev == 65535 {
		self.tailIndex = effect.linkNext
	} else {
		self.list[effect.linkPrev].linkNext = effect.linkNext
	}
	if effect.linkNext == 65535 {
		self.rawHeadIndex = self.activeHeadIndex
	} else {
		self.list[effect.linkNext].linkPrev = effect.linkPrev
	}

	// free "node"
	if effectIndex == uint16(len(self.list) - 1) {
		self.list = self.list[ : len(self.list) - 1]
	} else if self.freeIndex == 65535 {
		effect.spacing = nil // cleanup
		self.freeIndex = effectIndex
		effect.linkPrev = 65535
	} else {
		// readjust linking for free indices. the first conditional branch
		// could be actually applied unconditionally and it would work, but
		// I think adding the branches leads to better ordering?
		if effectIndex == self.freeIndex { panic("broken code") } // TODO: delete later
		effect.spacing = nil // cleanup
		if effectIndex < self.freeIndex {
			effect.linkPrev = self.freeIndex
			self.freeIndex = effectIndex
		} else { // effectIndex > self.freeIndex
			effect.linkPrev = self.list[self.freeIndex].linkPrev
			self.list[self.freeIndex].linkPrev = effectIndex
		}
	}
}

// --- helper methods ---

func (self *twineOperatorEffectsList) decreaseActiveCount(effect *effectOperationData) {
	// note: total count is not modified here, do manually when necessary
	if self.activeCount == 0 { panic("broken code") } // TODO: delete later
	self.activeCount -= 1
	if effect.mode == DoublePass {
		if self.activeDoublePassEffectCount == 0 { panic("broken code") } // TODO: delete later
		self.activeDoublePassEffectCount -= 1
	}
}

// Important: must also manually call self.totalCount += 1 if necessary.
func (self *twineOperatorEffectsList) increaseActiveCount(effect *effectOperationData) {
	self.activeCount += 1
	if effect.mode == DoublePass {
		self.activeDoublePassEffectCount += 1
	}
}

// Notice: the tail index must be managed manually.
func (self *twineOperatorEffectsList) refreshActiveHeadIndexFromPop(effect *effectOperationData) {
	for {
		index := effect.linkPrev
		if index == 65535 {
			self.activeHeadIndex = 65535
			return
		}
		
		effect = &self.list[index]
		if !effect.softPopped {
			self.activeHeadIndex = index
			return
		}
	}
}
