package liner

// Done returns a closed channel if not editing,
// or an open channel that will be closed when editing is done.
func (s *State) Done() chan struct{} {
	s.doneMutex.Lock()
	defer s.doneMutex.Unlock()

	if s.done == nil {
		s.done = make(chan struct{})
		close(s.done)
	}
	return s.done
}

type actCode int

const (
	noAct actCode = iota

	hidePrompt
	showPrompt
	chgPrompt
)

type asyncAction struct {
	act  actCode
	data interface{}
}

func (s *State) performAction(act actCode, data interface{}) {
	actReq := &asyncAction{
		act, data,
	}
	select {
	case s.actIn <- actReq:
		// the requested action went into prompting function
		for {
			// todo dead waiting possibile here ? need to add timeout etc. ?
			if actDid := <-s.actOut; actDid == actReq {
				return
			}
		}
	case <-s.Done():
		// not editing
		return
	}
}

// HidePrompt returns after the prompt and partial input is cleared if under line editing,
// or returns immediately if no editing undergoing.
func (s *State) HidePrompt() {
	s.performAction(hidePrompt, nil)
}

// ShowPrompt returns after the prompt and partial input is refreshed if under line editing,
// or returns immediately if no editing undergoing.
func (s *State) ShowPrompt() {
	s.performAction(showPrompt, nil)
}

// ChangePrompt change the prompt and restore partial input if under line editing,
// or returns immediately if no editing undergoing.
func (s *State) ChangePrompt(prompt string) {
	s.performAction(chgPrompt, prompt)
}
